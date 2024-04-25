package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

var testAccEntityLifecycle_entityTitle = testAccResourceTitle("ExampleHost")

func TestResourceEntitySchema(t *testing.T) {
	testResourceSchema(t, new(resourceEntity))
}

func TestResourceEntityPlan(t *testing.T) {
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

					resource "itsi_entity" "test" {
						title       = "example.com"
						description = "example.com host"

						aliases = {
							"entityTitle" = "example"
						}

						info = {
							"env" : "test"
							"entityType" : "123"
						}
					}
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccResourceEntityLifecycle(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckEntityDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_entity.test", "title", testAccEntityLifecycle_entityTitle),
					resource.TestCheckResourceAttr("itsi_entity.test", "description", "entityTest.example.com"),
					testAccCheckEntityExists,
				),
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_entity.test", "description", "TEST DESCRIPTION update"),
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

func testAccCheckEntityExists(s *terraform.State) error {
	return testAccCheckResourceExists(s, resourceNameEntity, testAccEntityLifecycle_entityTitle)
}

func testAccCheckEntityDestroy(s *terraform.State) error {
	return testAccCheckResourceDestroy(s, resourceNameEntity, testAccEntityLifecycle_entityTitle)
}
