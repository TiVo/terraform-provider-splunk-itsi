package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
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

func TestAccResourceServiceEntityFiltersLifecycle(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckResourceDestroy(resourceNameService, "service_create_filter_test"),
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.#", "1"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.0.field", "entityTitle"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.0.field_type", "alias"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.0.rule_type", "matches"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.0.value", "android_streamer"),
				),
			},
			// add another rule
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.#", "2"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.1.field", "entityTitle"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.1.field_type", "alias"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.1.rule_type", "matches"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.1.value", "android_streamer"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.0.field", "entityField"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.0.field_type", "info"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.0.rule_type", "not"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.0.value", "android_mobile"),
				),
			},
			// add entity_rule, update rule
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.1.rule.#", "1"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.1.rule.0.field", "entityTitle"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.1.rule.0.field_type", "title"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.1.rule.0.rule_type", "matches"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.1.rule.0.value", "android_tivoos"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.#", "1"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.0.field", "entityField"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.0.field_type", "info"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.0.rule_type", "not"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.0.value", "android_mobile"),
				),
			},
			// remove everything
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.#", "0"),
					resource.TestCheckNoResourceAttr("itsi_service.service_create_filter_test", "entity_rules.#"),
				),
			},
			// add entity_rules to the existing resource
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.1.rule.#", "1"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.1.rule.0.field", "entityTitle"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.1.rule.0.field_type", "title"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.1.rule.0.rule_type", "matches"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.1.rule.0.value", "android_tivoos"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.#", "1"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.0.field", "entityField"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.0.field_type", "info"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.0.rule_type", "not"),
					resource.TestCheckResourceAttr("itsi_service.service_create_filter_test", "entity_rules.0.rule.0.value", "android_mobile"),
				),
			},
		},
	})
}

func TestAccResourceServiceTagsLifecycle(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckResourceDestroy(resourceNameService, "service_create_tag_test"),
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_service.service_create_tag_test", "tags.0", "tag1"),
					resource.TestCheckResourceAttr("itsi_service.service_create_tag_test", "tags.1", "tag2"),
					resource.TestCheckResourceAttr("itsi_service.service_create_tag_test", "tags.2", "tag3"),
				),
			},
			// Tag was removed
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_service.service_create_tag_test", "tags.#", "2"),
					resource.TestCheckResourceAttr("itsi_service.service_create_tag_test", "tags.0", "tag1"),
					resource.TestCheckResourceAttr("itsi_service.service_create_tag_test", "tags.1", "tag3"),
				),
			},
			// Tag was added
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_service.service_create_tag_test", "tags.#", "3"),
					resource.TestCheckResourceAttr("itsi_service.service_create_tag_test", "tags.0", "tag1"),
					resource.TestCheckResourceAttr("itsi_service.service_create_tag_test", "tags.1", "tag3"),
					resource.TestCheckResourceAttr("itsi_service.service_create_tag_test", "tags.2", "tag5"),
				),
			},
			// All tags were removed [== null]
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_service.service_create_tag_test", "tags.#", "0"),
				),
			},
			// Tags were added to the existed resource
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_service.service_create_tag_test", "tags.#", "2"),
					resource.TestCheckResourceAttr("itsi_service.service_create_tag_test", "tags.0", "tag6"),
					resource.TestCheckResourceAttr("itsi_service.service_create_tag_test", "tags.1", "tag7"),
				),
			},
		},
	})
}

func TestAccResourceServiceDependsOnLifecycle(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckResourceDestroy(resourceNameService, "service_create_parent"),
			testAccCheckResourceDestroy(resourceNameService, "service_create_leaf"),
			testAccCheckResourceDestroy(resourceNameService, "service_create_leaf_overloaded"),
		),
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("itsi_service.service_create_leaf", "shkpi_id"),
					resource.TestCheckResourceAttr("itsi_service.service_create_parent", "service_depends_on.#", "1"),
					testCheckServiceDependsOnMatch("itsi_service.service_create_leaf"),
				),
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("itsi_service.service_create_leaf", "shkpi_id"),
					resource.TestCheckResourceAttrSet("itsi_service.service_create_leaf_overloaded", "shkpi_id"),
					resource.TestCheckResourceAttr("itsi_service.service_create_parent", "service_depends_on.#", "2"),
					testCheckServiceDependsOnMatch("itsi_service.service_create_leaf"),
					testCheckServiceDependsOnMatch("itsi_service.service_create_leaf_overloaded", 8),
				),
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("itsi_service.service_create_leaf", "shkpi_id"),
					resource.TestCheckResourceAttr("itsi_service.service_create_parent", "service_depends_on.#", "0"),
				),
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("itsi_service.service_create_leaf", "shkpi_id"),
					resource.TestCheckResourceAttrSet("itsi_service.service_create_leaf_overloaded", "shkpi_id"),
					resource.TestCheckResourceAttr("itsi_service.service_create_parent", "service_depends_on.#", "2"),
					testCheckServiceDependsOnMatch("itsi_service.service_create_leaf"),
					testCheckServiceDependsOnMatch("itsi_service.service_create_leaf_overloaded", 8),
				),
			},
		},
	})
}
func testCheckServiceDependsOnMatch(child_name string, expected_overloaded_urgency ...int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		leafResource, ok := s.RootModule().Resources[child_name]
		if !ok {
			return fmt.Errorf("Not found: itsi_service.service_create_leaf")
		}
		leafKPIID := leafResource.Primary.Attributes["shkpi_id"]

		parentResource, ok := s.RootModule().Resources["itsi_service.service_create_parent"]
		if !ok {
			return fmt.Errorf("Not found: itsi_service.service_create_parent")
		}
		kpiLength, err := strconv.Atoi(parentResource.Primary.Attributes["service_depends_on.0.kpis.#"])
		if err != nil {
			return fmt.Errorf("Kpi depends on length not found")
		}
		for i := 0; i < kpiLength; i++ {
			parentKPIID := parentResource.Primary.Attributes["service_depends_on.0.kpis."+strconv.Itoa(i)]
			if leafKPIID == parentKPIID {
				fmt.Printf("PASSED: Leaf shkpi_id %s, Parent's dependent kpis %s\n", leafKPIID, parentResource.Primary.Attributes["service_depends_on.0.kpis"])

				if len(expected_overloaded_urgency) > 0 {
					if urgency, ok := parentResource.Primary.Attributes["service_depends_on.0.kpis."+strconv.Itoa(i)+
						".overload_urgencies."+leafKPIID]; !ok || urgency != strconv.Itoa(expected_overloaded_urgency[0]) {
						return fmt.Errorf("%s mismatch: Missing expected overloaded_urgency %s", leafKPIID,
							parentResource.Primary.Attributes["service_depends_on.0.kpis."+strconv.Itoa(i)])
					}
				}
			}
			return nil
		}
		return fmt.Errorf("shkpi_id mismatch: Missing shkpi_id %s", leafKPIID)
	}

}
