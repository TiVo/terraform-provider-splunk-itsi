package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

func TestResourceEntitySchema(t *testing.T) {
	testResourceSchema(t, new(resourceEntity))
}

func TestResourceEntityPlan(t *testing.T) {
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

func TestAccResourceEntityCreate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		CheckDestroy:             testAccCheckEntityDestroy,
		Steps: []resource.TestStep{
			{
				Config: util.Dedent(`
					provider "itsi" {}
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
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_entity.test", "title", "example.com"),
					resource.TestCheckResourceAttr("itsi_entity.test", "description", "example.com host"),
				),
			},
		},
	})
}

func testAccCheckEntityDestroy(s *terraform.State) error {
	return testAccCheckResourceDestroy(s, resourceNameEntity, "example.com")
}

func testAccCheckResourceDestroy(s *terraform.State, resourcetype resourceName, resourceTitle string) error {
	for _, rs := range s.RootModule().Resources {
		if !(rs.Type == "itsi_"+string(resourcetype) && rs.Primary.Attributes["title"] == resourceTitle) {
			continue
		}

		base := models.NewBase(clientConfig, resourceTitle, resourceTitle, string(resourcetype))
		b, err := base.Find(context.Background())
		if err != nil {
			return err
		}
		if b != nil {
			return fmt.Errorf("Resource %s.%s still exists", resourcetype, resourceTitle)
		}
	}

	return nil
}
