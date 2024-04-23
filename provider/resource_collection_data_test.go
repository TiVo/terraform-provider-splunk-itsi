package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

func TestResourceCollectionDataSchema(t *testing.T) {
	testResourceSchema(t, new(resourceCollectionData))
}

func TestResourceCollectionDataPlan(t *testing.T) {
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

					resource "itsi_collection_data" "test_data" {
					  scope = "example_scope"
					  collection {
					    name = "collection-data-test"
					  }

					  entry {
					    data = jsonencode({
					      name  = "abc"
					      color = [["123"]]
					    })
					  }
					}
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
