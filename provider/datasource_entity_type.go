package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

func DatasourceEntityType() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to get the ID of an available entity type.",
		ReadContext: entityTypeRead,
		Schema: map[string]*schema.Schema{
			"title": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the entity type",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "_key value for the entity type",
			},
		},
	}
}

func entityTypeBase(clientConfig models.ClientConfig, key string, title string) *models.Base {
	base := models.NewBase(clientConfig, key, title, "entity_type")
	return base
}

func entityTypeRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	base := entityTypeBase(m.(models.ClientConfig), d.Id(), d.Get("title").(string))
	b, err := base.Find(ctx)
	if err != nil {
		return append(diags, diag.Errorf("%s", err)...)
	}
	if b == nil {
		d.SetId("")
		return nil
	}

	err = populateEntityTypeResourceData(b, d)
	if err != nil {
		return append(diags, diag.Errorf("%s", err)...)
	}
	return
}

func populateEntityTypeResourceData(b *models.Base, d *schema.ResourceData) error {
	by, err := b.RawJson.MarshalJSON()
	if err != nil {
		return err
	}
	var interfaceMap map[string]interface{}
	err = json.Unmarshal(by, &interfaceMap)
	if err != nil {
		return err
	}

	err = d.Set("title", interfaceMap["title"])
	if err != nil {
		return err
	}

	err = d.Set("id", interfaceMap["_key"])
	if err != nil {
		return err
	}

	d.SetId(b.RESTKey)
	return nil
}
