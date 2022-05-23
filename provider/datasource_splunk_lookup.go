package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

func DatasourceSplunkLookup() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to retrieve the contents of a Splunk lookup table.",
		ReadContext: splunkLookupRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the lookup that should be used.",
			},
			"splunk_user": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "nobody",
				Description: "The Splunk user in the context of which the search query should be performed.",
			},
			"splunk_app": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "search",
				Description: "The Splunk app in which the searh query should be performed.",
			},
			"data": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeMap,
				},
				Description: "Lookup data.",
			},
		},
	}
}

func splunkLookupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(models.ClientConfig)

	searches := []SplunkSearch{
		SplunkSearch{
			Query:               " | inputlookup " + d.Get("name").(string),
			User:                d.Get("splunk_user").(string),
			App:                 d.Get("splunk_app").(string),
			EarliestTime:        "0",
			LatestTime:          "@d",
			AllowPartialResults: false,
		},
	}

	req := NewSplunkRequest(client, searches, 1, nil)
	results, diags := req.Run(ctx)
	d.SetId(req.ID())
	err := d.Set("data", results)
	if err != nil {
		diags = append(diags, diag.Errorf("%s", err)...)
	}
	return
}
