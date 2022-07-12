package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

func entityTFFormat(b *models.Base) (string, error) {
	res := ResourceEntity()
	resData := res.Data(nil)
	d := populateEntityResourceData(context.Background(), b, resData)
	if len(d) > 0 {
		err := d[0].Validate()
		if err != nil {
			return "", err
		}
		return "", errors.New(d[0].Summary)
	}
	resourcetpl, err := NewResourceTemplate(resData, res.Schema, "title", "itsi_entity")
	if err != nil {
		return "", err
	}

	templateResource, err := newTemplate(resourcetpl)
	if err != nil {
		log.Fatal(err)
	}
	var tpl bytes.Buffer
	err = templateResource.Execute(&tpl, resourcetpl)
	if err != nil {
		return "", err
	}

	return cleanerRegex.ReplaceAllString(tpl.String(), ""), nil
}

func entityBase(clientConfig models.ClientConfig, key string, title string) *models.Base {
	base := models.NewBase(clientConfig, key, title, "entity")
	return base
}

func ResourceEntity() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages an Entity object within ITSI.",
		CreateContext: entityCreate,
		ReadContext:   entityRead,
		UpdateContext: entityUpdate,
		DeleteContext: entityDelete,
		Importer: &schema.ResourceImporter{
			StateContext: entityImport,
		},
		Schema: map[string]*schema.Schema{
			"title": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the entity. Can be any unique value.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "User defined description of the entity.",
			},
			"aliases": {
				Type:        schema.TypeMap,
				Required:    true,
				Description: "Map of Field/Value pairs that identify the entity",
			},
			"info": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Map of Field/Value pairs that provide information/description for the entity",
			},
			"entity_type_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Array of _key values for each entity type associated with the entity.",
			},
		},
	}
}

func entity(d *schema.ResourceData, clientConfig models.ClientConfig) (config *models.Base, err error) {
	body := map[string]interface{}{}

	body["object_type"] = "entity"
	body["sec_grp"] = "default_itsi_security_group"
	body["title"] = d.Get("title").(string)
	body["description"] = d.Get("description").(string)

	idFields, infoFields := []string{}, []string{}
	idValues, infoValues := []string{}, []string{}

	idFieldsSet, infoFieldsSet := map[string]bool{}, map[string]bool{}
	idValuesSet, infoValuesSet := map[string]bool{d.Get("title").(string): true}, map[string]bool{}
	aliases := d.Get("aliases").(map[string]interface{})
	info := d.Get("info").(map[string]interface{})

	for k, v := range aliases {
		idFieldsSet[k] = true
		idValuesSet[v.(string)] = true
		body[k] = []string{v.(string)}
	}

	for k := range idFieldsSet {
		idFields = append(idFields, k)
	}

	for k := range idValuesSet {
		idValues = append(idValues, k)
	}

	for k, v := range info {
		infoFieldsSet[k] = true
		infoValuesSet[v.(string)] = true
		body[k] = []string{v.(string)}
	}

	for k := range infoFieldsSet {
		infoFields = append(infoFields, k)
	}

	for k := range idValuesSet {
		infoValues = append(infoValues, k)
	}

	body["identifier"] = map[string][]string{"fields": idFields, "values": idValues}
	body["informational"] = map[string][]string{"fields": infoFields, "values": infoValues}
	body["entity_type_ids"] = d.Get("entity_type_ids").(*schema.Set).List()

	by, err := json.Marshal(body)
	if err != nil {
		return
	}
	base := entityBase(clientConfig, d.Id(), d.Get("title").(string))
	err = json.Unmarshal(by, &base.RawJson)
	if err != nil {
		return nil, err
	}
	return base, nil
}

func entityCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	template, err := entity(d, m.(models.ClientConfig))
	tflog.Info(ctx, "ENTITY: create", map[string]interface{}{"TFID": template.TFID, "err": err})
	if err != nil {
		return diag.FromErr(err)
	}
	b, err := template.Create(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	b.Read(ctx)
	return populateEntityResourceData(ctx, b, d)
}

func entityRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	base := entityBase(m.(models.ClientConfig), d.Id(), d.Get("title").(string))
	tflog.Info(ctx, "ENTITY: read", map[string]interface{}{"TFID": base.TFID})
	b, err := base.Find(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	if b == nil || b.RawJson == nil {
		d.SetId("")
		return nil
	}
	return populateEntityResourceData(ctx, b, d)
}

func populateEntityResourceData(ctx context.Context, b *models.Base, d *schema.ResourceData) (diags diag.Diagnostics) {
	by, err := b.RawJson.MarshalJSON()
	if err != nil {
		return diag.FromErr(err)
	}
	var interfaceMap map[string]interface{}
	err = json.Unmarshal(by, &interfaceMap)
	if err != nil {
		return diag.FromErr(err)
	}

	for _, f := range []string{"title", "description"} {
		err = d.Set(f, interfaceMap[f])
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if v, ok := interfaceMap["entity_type_ids"]; ok && v != nil {
		entityTypeIds := v.([]interface{})
		if len(entityTypeIds) > 0 {
			err = d.Set("entity_type_ids", entityTypeIds)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	for tfField, itsiField := range map[string]string{"aliases": "identifier", "info": "informational"} {
		tfMap := map[string]string{}
		itsiObject, ok := interfaceMap[itsiField].(map[string]interface{})
		if !ok {
			tflog.Warn(ctx, "ENTITY: populate",
				map[string]interface{}{"TFID": b.TFID, "b": b, "map": interfaceMap, "field": itsiField})
			return diag.Errorf("entity resource (%v): type assertion failed for '%v' field", b.RESTKey, itsiField)
		}

		for _, k := range itsiObject["fields"].([]interface{}) {
			itsiValues, ok := interfaceMap[k.(string)].([]interface{})
			if !ok {

				return diag.Errorf("entity resource (%v): type assertion failed for '%v/fields' field", b.RESTKey, itsiField)
			}

			if len(itsiValues) == 0 {
				return diag.Errorf("entity resource (%v): missing value for '%v/fields/%v' field", b.RESTKey, itsiField, k.(string))

			}

			tfMap[k.(string)] = itsiValues[0].(string)
		}

		if err = d.Set(tfField, tfMap); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(b.RESTKey)
	return nil
}

func entityUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	clientConfig := m.(models.ClientConfig)
	base := entityBase(clientConfig, d.Id(), d.Get("title").(string))
	tflog.Info(ctx, "ENTITY: update", map[string]interface{}{"TFID": base.TFID})
	existing, err := base.Find(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	if existing == nil {
		return entityCreate(ctx, d, m)
	}

	template, err := entity(d, clientConfig)
	if err != nil {
		return diag.FromErr(err)
	}
	return diag.FromErr(template.Update(ctx))
}

func entityDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	base := entityBase(m.(models.ClientConfig), d.Id(), d.Get("title").(string))
	tflog.Info(ctx, "ENTITY: delete", map[string]interface{}{"TFID": base.TFID})
	existing, err := base.Find(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	if existing == nil {
		return diag.Errorf("Unable to find entity model")
	}
	return diag.FromErr(existing.Delete(ctx))
}

func entityImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	b := entityBase(m.(models.ClientConfig), "", d.Id())
	b, err := b.Find(ctx)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, err
	}
	diags := populateEntityResourceData(ctx, b, d)
	for _, d := range diags {
		if d.Severity == diag.Error {
			return nil, fmt.Errorf(d.Summary)
		}
	}

	if d.Id() == "" {
		return nil, nil
	}
	return []*schema.ResourceData{d}, nil
}
