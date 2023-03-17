package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

func DatasourceKPIBaseSearch() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to get the ID of an available KPI Base Search.",
		ReadContext: DatasourceKPIBaseSearchRead,
		Schema: map[string]*schema.Schema{
			"title": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The title of the KPI Base Search",
			},
		},
	}
}

func DatasourceKPIBaseSearchRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	base := kpiBaseSearchBase(m.(models.ClientConfig), d.Id(), d.Get("title").(string))
	b, err := base.Find(ctx)
	if err != nil {
		return append(diags, diag.Errorf("%s", err)...)
	}
	if b == nil {
		d.SetId("")
		return nil
	}

	err = populateKPIBaseSearchDatasourceData(b, d)
	if err != nil {
		return append(diags, diag.Errorf("%s", err)...)
	}
	return
}

func populateKPIBaseSearchDatasourceData(b *models.Base, d *schema.ResourceData) error {
	interfaceMap, err := b.RawJson.ToInterfaceMap()
	if err != nil {
		return err
	}

	err = d.Set("title", interfaceMap["title"])
	if err != nil {
		return err
	}

	d.SetId(b.RESTKey)
	return nil
}
