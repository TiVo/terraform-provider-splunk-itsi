package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

func TestResourceCollection(t *testing.T) {
	//collectionName := "itsi_splunk_collection.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: util.Dedent(`
					provider "itsi" {
						host     = "itsi.example.com"
						user     = "user"
						password = "password"
						port     = 8089
						timeout  = 20
					}

					resource "itsi_splunk_collection" "test" {
						name = "example_collection"
					}
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				// Check: resource.ComposeAggregateTestCheckFunc(
				// 	resource.TestCheckResourceAttr(collectionName, "name3", "example_collection2"),
				// ),
				// ConfigPlanChecks: resource.ConfigPlanChecks{
				// 	PreApply: []plancheck.PlanCheck{
				// 		plancheck.ExpectKnownValue(collectionName, tfjsonpath.New("name"), knownvalue.StringExact("example_collection")),
				// 	},
				// },
			},
		},
	})
}
