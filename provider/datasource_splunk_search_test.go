package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

func TestDataSourceSplunkSearchSchema(t *testing.T) {
	testDataSourceSchema(t, new(dataSourceSplunkSearch))
}

func TestAccDataSplunkSearch(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: util.Dedent(`
					data "itsi_splunk_search" "test" {
						search {
							query = "| makeresults count = 1 | eval a=123 | table a"
							earliest_time = "-1m"
							timeout       = 120
						}
					}
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.itsi_splunk_search.test", "results", `[{"a":"123"}]`),
				),
			},
		},
	})
}
