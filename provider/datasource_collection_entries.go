package provider

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"

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
	if err := populateCollectionEntriesDatasource(ctx, c, d); err != nil {
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

func populateCollectionEntriesDatasource(ctx context.Context, c *models.CollectionApi, d *schema.ResourceData) (err error) {
	var obj interface{}
	if obj, err = c.Unmarshal(c.Body); err != nil {
		return err
	}
	arr, ok := obj.([]interface{})
	if !ok {
		return fmt.Errorf("expected array body return type")
	}

	tflog.Trace(ctx, "RSRC COLLECTION:   populate", map[string]interface{}{"arr": arr})
	data := make([][]map[string]interface{}, 0, len(arr))

	for _, item := range arr {
		item_, ok := item.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected map in array body return type")
		}
		row := []map[string]interface{}{}
		for k, v := range item_ {
			// Perhaps do not include internal fields...
			if k[0] != '_' || k == "_idx" || (k == "_key") {
				m := map[string]interface{}{"name": k}
				if singleValue, ok := v.(string); ok {
					m["values"] = []string{singleValue}
				} else if multiValue, ok := v.([]interface{}); ok {
					m["values"] = multiValue
				} else {
					return fmt.Errorf("invalid collection value %#v", v)
				}

				row = append(row, m)
			}
		}
		tflog.Trace(ctx, "RSRC COLLECTION:     item", map[string]interface{}{"item": item_, "row": row})
		data = append(data, row)
	}

	if c.Data == nil {
		c.Data = make(map[string]interface{})
	}
	c.Data["data"] = data

	if err = d.Set("collection_name", c.Data["collection_name"]); err != nil {
		return err
	}

	tflog.Debug(ctx, "RSRC COLLECTION:   populate", map[string]interface{}{"data": data})

	// Reorder the data by our saved index ordering...
	rowIndex, idxPos := make([]int, len(data)), make([]int, len(data))
	for i, row := range data {
		for j, item := range row {
			if item["name"] == "_idx" {
				idx, err := strconv.Atoi(item["values"].([]string)[0])
				if err != nil {
					return err
				}
				rowIndex[i] = idx
				idxPos[i] = j
				break
			}
		}
	}

	//Now remove the artificial "_idx" field...
	for i := 0; i < len(data); i++ {
		data[i] = append(data[i][:idxPos[i]], data[i][idxPos[i]+1:]...)
	}

	sort.SliceStable(data, func(i, j int) bool {
		return rowIndex[i] < rowIndex[j]
	})

	if err = d.Set("data", data); err != nil {
		return err
	}

	d.SetId(c.RESTKey)
	return nil
}
