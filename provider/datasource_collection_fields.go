package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

func DatasourceCollectionFields() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to retrieve the list of field names of a collection.",
		ReadContext: collectionFieldsRead,
		Schema: map[string]*schema.Schema{
			"collection_name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Name of the collection",
				ValidateFunc: validateStringIdentifier(),
			},
			"collection_app": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "itsi",
				Description:  "App the collection belongs to",
				ValidateFunc: validateStringIdentifier(),
			},
			"collection_owner": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "nobody",
				Description:  "Owner of the collection",
				ValidateFunc: validateStringIdentifier(),
			},
			"fields": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Collection fields",
			},
		},
	}
}

func collectionApiDataRead(ctx context.Context, d *schema.ResourceData, m interface{}) (api *models.CollectionApi, err error) {
	tflog.Info(ctx, "RSRC COLLECTION:   read ("+d.Get("collection_name").(string)+")")
	// To read the whole collection, we use...
	//   storage/collections/data/{collection} -- GET
	api, _ = collection(ctx, d, "collection_data", m)
	return api.Read(ctx)
}

func collectionFieldsRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	tflog.Info(ctx, "RSRC COLLECTION: ENTRIES FIELDS READ", map[string]interface{}{"d": fmt.Sprintf("%+v", d)})
	if diags = collectionCheck(ctx, d, m); diags != nil {
		return
	}

	c, err := collectionApiDataRead(ctx, d, m)
	if err != nil {
		diags = append(diags, errorf("Unable to read/dump collection model: "+err.Error()))
		return
	}
	if c == nil || c.Data == nil {
		d.SetId("")
		return nil
	}
	if err := populateCollectionFieldsDatasource(ctx, c, d); err != nil {
		diags = append(diags, errorf("Unable to populate resource: "+err.Error()))
	}
	return
}

func populateCollectionFieldsDatasource(ctx context.Context, c *models.CollectionApi, d *schema.ResourceData) (err error) {
	var obj interface{}
	if obj, err = c.Unmarshal(c.Body); err != nil {
		return err
	}
	arr, ok := obj.([]interface{})
	if !ok {
		return fmt.Errorf("expected array body return type")
	}

	tflog.Trace(ctx, "RSRC COLLECTION:   populate", map[string]interface{}{"arr": arr})
	fieldsMap := map[string]bool{}

	for _, item := range arr {
		item_, ok := item.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected map in array body return type")
		}
		row := []map[string]interface{}{}
		for k := range item_ {
			// Perhaps do not include internal fields...
			if k[0] != '_' {
				fieldsMap[k] = true
			}
		}
		tflog.Trace(ctx, "RSRC COLLECTION:     item", map[string]interface{}{"item": item_, "row": row})
	}

	fields := []string{}
	for k := range fieldsMap {
		fields = append(fields, k)
	}

	sort.Strings(fields)

	if err = d.Set("collection_name", c.Data["collection_name"]); err != nil {
		return err
	}

	if err = d.Set("fields", fields); err != nil {
		return err
	}

	tflog.Debug(ctx, "RSRC COLLECTION FIELDS:   populate", map[string]interface{}{"fields": fields})

	d.SetId(c.RESTKey)
	return nil
}
