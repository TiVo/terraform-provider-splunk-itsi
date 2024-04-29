package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

func TestResourceServiceSchema(t *testing.T) {
	testResourceSchema(t, new(resourceService))
}

func TestResourceServicePlan(t *testing.T) {
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

					resource "itsi_service" "Test-custom-static-threshold" {
					  enabled = true
					  entity_rules {
					    rule {
					      field      = "host"
					      field_type = "alias"
					      rule_type  = "matches"
					      value      = "example.com"
					    }
					  }
					  kpi {
					    base_search_id     = "625f502d7e6e1a37ea062eff"
					    base_search_metric = "host_count"
					    search_type           = "shared_base"
					    threshold_template_id = "123"
					    title                 = "Test custom static threshold KPI 1"
					    type                  = "kpis_primary"
					    urgency               = 5
					  }
					  security_group = "default_itsi_security_group"
					  title          = "Test custom static threshold"
					}
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccServiceResourceEntityFilter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				//ExpectNonEmptyPlan: true,
				Config: util.Dedent(`
                    provider "itsi" {}

					resource "itsi_service" "service_create_filter_test" {
						title    = "Test Service Create filter test"
						description = "Terraform unit test"
						entity_rules {
						  rule {
							  field      = "entityTitle"
							  field_type = "alias"
							  rule_type  = "matches"
							  value      = "android_streamer"
						  }
						  rule {
							  field      = "entityField"
							  field_type = "info"
							  rule_type  = "not"
							  value      = "android_mobile"
						  }
						}
					  }
                `),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.#", "2"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.0.field", "entityField"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.0.field_type", "info"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.0.rule_type", "not"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.0.value", "android_mobile"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.1.field_type", "alias"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.1.rule_type", "matches"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.1.value", "android_streamer"),
				),
			},
		},
	})
}

func TestAccServiceResourceTagCreate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				//ExpectNonEmptyPlan: true,
				Config: util.Dedent(`
                    provider "itsi" {}

					resource "itsi_service" "service_create_tag_test" {
						title    = "Test Tag Creation"
						description = "Terraform unit test"
						tags = ["tag1", "tag2", "tag3"]
					}
                `),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_service.service_create_tag_test", "description", "Terraform unit test"),
					resource.TestCheckResourceAttr("itsi_service.service_create_tag_test", "tags.0", "tag1"),
					resource.TestCheckResourceAttr("itsi_service.service_create_tag_test", "tags.1", "tag2"),
					resource.TestCheckResourceAttr("itsi_service.service_create_tag_test", "tags.2", "tag3"),
				),
			},
			{
				Destroy: true,
			},
		},
	})
}

func TestAccServiceResourceHierarchyCreate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		PreCheck:                 func() { testAccPreCheck(t) },

		Steps: []resource.TestStep{
			{
				//ExpectNonEmptyPlan: true,
				Config: util.Dedent(`
                    provider "itsi" {}

                    resource "itsi_service" "service_create_parent" {
                        title = "Service Test on Create Parent"
                        service_depends_on {
                            kpis = [
								itsi_service.service_create_leaf.shkpi_id
							]
                            service = itsi_service.service_create_leaf.id
                        }
                    }

                    resource "itsi_service" "service_create_leaf" {
                        title = "Service Test on Create Leaf"
                    }
                `),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_service.service_create_leaf", "title", "Service Test on Create Leaf"),
					resource.TestCheckResourceAttr("itsi_service.service_create_parent", "title", "Service Test on Create Parent"),
					resource.TestCheckResourceAttrSet("itsi_service.service_create_leaf", "shkpi_id"),
					testCheckServiceShkpiIdMatch,
				),
			},
			{
				Destroy: true,
			},
		},
	})
}

// testCheckServiceShkpiIdMatch checks if the shkpi_id of leaf is the same as the one in parent's service_depends_on.kpis
func testCheckServiceShkpiIdMatch(s *terraform.State) error {
	leafResource, ok := s.RootModule().Resources["itsi_service.service_create_leaf"]
	if !ok {
		return fmt.Errorf("Not found: itsi_service.service_create_leaf")
	}
	leafKPIID := leafResource.Primary.Attributes["shkpi_id"]

	parentResource, ok := s.RootModule().Resources["itsi_service.service_create_parent"]
	if !ok {
		return fmt.Errorf("Not found: itsi_service.service_create_parent")
	}
	parentKPIID := parentResource.Primary.Attributes["service_depends_on.0.kpis.0"]

	if leafKPIID != parentKPIID {
		return fmt.Errorf("shkpi_id mismatch: Leaf shkpi_id %s, Parent's dependent kpis %s", leafKPIID, parentKPIID)
	} else {
		fmt.Printf("PASSED: Leaf shkpi_id %s, Parent's dependent kpis %s", leafKPIID, parentKPIID)
	}
	return nil
}
