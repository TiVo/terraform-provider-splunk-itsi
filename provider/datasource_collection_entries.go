package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

func DataSourceCollectionEntries() *schema.Resource {
	return &schema.Resource{
		Description: "Read KV store collection records.",
		ReadContext: collectionEntriesQuery,
		Schema: map[string]*schema.Schema{
			"collection_name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Name of the collection containing this entry",
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
			"query": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Query to filter the data requested",
			},
			"data": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeSet,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"name": {
								Type:        schema.TypeString,
								Computed:    true,
								Description: "Name of the field",
							},
							"values": {
								Type:        schema.TypeList,
								Computed:    true,
								Description: "Values of the field",
								Elem: &schema.Schema{
									Type: schema.TypeString,
								},
							},
						},
					},
				},
				Description: "Collection data for all entries",
			},
		},
	}
}

func collectionEntriesQuery(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	tflog.Info(ctx, "RSRC COLLECTION: ENTRIES READ", map[string]interface{}{"d": fmt.Sprintf("%+v", d)})
	if diags = collectionCheck(ctx, d, m); diags != nil {
		return
	}

	c, err := collectionApiEntriesQuery(ctx, d, m)
	if err != nil {
		diags = append(diags, errorf("Unable to read/dump collection model: "+err.Error()))
		return
	}
	if c == nil || c.Data == nil {
		d.SetId("")
		return nil
	}
	if err := populateCollectionEntriesData(ctx, c, d, false); err != nil {
		diags = append(diags, errorf("Unable to populate resource: "+err.Error()))
	}
	return
}

func collectionApiEntriesQuery(ctx context.Context, d *schema.ResourceData, m interface{}) (api *models.CollectionApi, err error) {
	tflog.Info(ctx, "RSRC COLLECTION:   entries read ("+d.Get("collection_name").(string)+")")
	// To read the contents of a collection, we use...
	//   storage/collections/data/{collection} -- GET
	api, _ = collectionDataEntries(ctx, d, "collection_data", m)

	query := api.Data["query"].(string)
	api.Params = "query=" + url.QueryEscape(query)

	return api.Read(ctx)
}

// Create a "collection API model" that can help us query APIs,
// passing no body in the query...
func collectionDataEntries(ctx context.Context, d *schema.ResourceData, object_type string, m interface{}) (config *models.CollectionApi, err error) {
	clientConfig := m.(models.ClientConfig)

	name, err := getCollectionField(d, "name")
	if err != nil {
		return
	}
	app, err := getCollectionField(d, "app")
	if err != nil {
		return
	}
	owner, err := getCollectionField(d, "owner")
	if err != nil {
		return
	}

	c := models.NewCollection(clientConfig, name, app, owner, name, object_type)

	data := make(map[string]interface{})
	data["collection_name"] = d.Get("collection_name").(string)
	data["query"] = d.Get("query").(string)
	c.Data = data

	tflog.Trace(ctx, "RSRC COLLECTION:     api model ("+object_type+")",
		map[string]interface{}{"key": c.RESTKey, "data": data})
	return c, nil
}
