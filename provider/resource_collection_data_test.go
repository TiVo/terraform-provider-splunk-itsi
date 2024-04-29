package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

// var testAccCollectionDataLifecycle_collectionName = testAccResourceTitle("collection_data_test")
var testAccCollectionDataLifecycle_collectionDataScope = testAccResourceTitle("collection_data_test")

func TestResourceCollectionDataSchema(t *testing.T) {
	testResourceSchema(t, new(resourceCollectionData))
}

func TestResourceCollectionDataPlan(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
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

func TestAccResourceCollectionDataLifecycle(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckResourceDestroy(resourceNameCollectionData, testAccCollectionDataLifecycle_collectionDataScope),
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_collection_data.test", "scope", testAccCollectionDataLifecycle_collectionDataScope),
					testAccCheckResourceExists(resourceNameCollectionData, testAccCollectionDataLifecycle_collectionDataScope),
				),
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_collection_data.test", "scope", testAccCollectionDataLifecycle_collectionDataScope),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}
