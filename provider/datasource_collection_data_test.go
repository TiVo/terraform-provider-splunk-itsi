package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

func TestDataSourceCollectionDataSchema(t *testing.T) {
	testDataSourceSchema(t, new(dataSourceCollectionData))
}

func TestAccDataCollectionData(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: util.Dedent(`

					data "itsi_collection_data" "test" {

						collection {
							name  = "itsi_team"
							app   = "SA-ITOA"
							owner = "nobody"
						}
						query = <<-EOT
							{"object_type": "team", "_key": "default_itsi_security_group"}
						EOT

						fields = [
							"_key",
							"title"
						]

					}
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.itsi_collection_data.test", "data", `[{"_key":"default_itsi_security_group","title":"Global"}]`),
				),
			},
		},
	})
}
