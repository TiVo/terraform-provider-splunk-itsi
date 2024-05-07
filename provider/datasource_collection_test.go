package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

func TestDataSourceCollectionSchema(t *testing.T) {
	testDataSourceSchema(t, new(dataSourceCollection))
}

func TestAccDataCollection(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: util.Dedent(`
					data "itsi_splunk_collection" "test" {
						name  = "itsi_team"
						app   = "SA-ITOA"
						owner = "nobody"
					}
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.itsi_splunk_collection.test", "name", "itsi_team"),
					resource.TestCheckResourceAttrSet("data.itsi_splunk_collection.test", "fields.#"),
				),
			},
		},
	})
}
