package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
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

				// if len(expected_overloaded_urgency) > 0 {
				// 	if urgency, ok := parentResource.Primary.Attributes["service_depends_on.0.kpis."+strconv.Itoa(i)+
				// 		".overload_urgencies."+leafKPIID]; !ok || urgency != strconv.Itoa(expected_overloaded_urgency[0]) {
				// 		return fmt.Errorf("%s mismatch: Missing expected overloaded_urgency %s\n", leafKPIID,
				// 			parentResource.Primary.Attributes["service_depends_on.0.kpis."+strconv.Itoa(i)])
				// 	}
				// }
			}
			return nil
		}
		return fmt.Errorf("shkpi_id mismatch: Missing shkpi_id %s\n", leafKPIID)
	}

}

func TestAccResourceServiceKpisLifecycle(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckResourceDestroy(resourceNameService, "test_kpis"),
			testAccCheckResourceDestroy(resourceNameKPIBaseSearch, "test_kpis_linked_kpibs_1"),
			testAccCheckResourceDestroy(resourceNameEntityType, "test_kpis_linked_kpibs_2"),
			testAccCheckResourceDestroy(resourceNameKPIThresholdTemplate, "test_kpis_static"),
			testAccCheckResourceDestroy(resourceNameKPIThresholdTemplate, "test_kpis_kpi_threshold_template_1"),
		),
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
						plancheck.ExpectUnknownValue("itsi_service.test_kpis", tfjsonpath.New("kpi").AtSliceIndex(0).AtMapKey("id")),
						plancheck.ExpectUnknownValue("itsi_service.test_kpis", tfjsonpath.New("kpi").AtSliceIndex(1).AtMapKey("id")),
						plancheck.ExpectUnknownValue("itsi_service.test_kpis", tfjsonpath.New("kpi").AtSliceIndex(2).AtMapKey("id")),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.#", "3"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.0.title", "KPI 1"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.0.base_search_metric", "metric 1.1"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.0.base_search_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.0.threshold_template_id"),

					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.1.title", "KPI 2"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.1.base_search_metric", "metric 1.2"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.1.base_search_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.1.threshold_template_id"),

					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.2.title", "KPI 3"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.2.base_search_metric", "metric 2.1"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.2.base_search_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.2.threshold_template_id"),

					SaveKpiIds,
				),
			},
			// add kpi
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),

						plancheck.ExpectKnownValue("itsi_service.test_kpis", tfjsonpath.New("kpi").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
						plancheck.ExpectKnownValue("itsi_service.test_kpis", tfjsonpath.New("kpi").AtSliceIndex(1).AtMapKey("id"), knownvalue.NotNull()),
						plancheck.ExpectKnownValue("itsi_service.test_kpis", tfjsonpath.New("kpi").AtSliceIndex(2).AtMapKey("id"), knownvalue.NotNull()),

						plancheck.ExpectUnknownValue("itsi_service.test_kpis", tfjsonpath.New("kpi").AtSliceIndex(3).AtMapKey("id")),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.#", "4"),
					resource.TestCheckResourceAttrWith("itsi_service.test_kpis", "kpi.0.id", verifyKpiId(0)),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.0.title", "KPI 1"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.0.base_search_metric", "metric 1.1"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.0.base_search_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.0.threshold_template_id"),

					resource.TestCheckResourceAttrWith("itsi_service.test_kpis", "kpi.1.id", verifyKpiId(1)),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.1.title", "KPI 2"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.1.base_search_metric", "metric 1.2"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.1.base_search_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.1.threshold_template_id"),

					resource.TestCheckResourceAttrWith("itsi_service.test_kpis", "kpi.2.id", verifyKpiId(2)),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.2.title", "KPI 3"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.2.base_search_metric", "metric 2.1"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.2.base_search_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.2.threshold_template_id"),

					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.3.id"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.3.title", "KPI 4"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.3.base_search_metric", "metric 2.2"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.3.base_search_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.3.threshold_template_id"),

					SaveKpiIds,
				),
			}, // change metric of KPI 1 (ID regenerated), unit & description of KPI 2 (ID should stay the same), remove KPI 3
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),

						plancheck.ExpectUnknownValue("itsi_service.test_kpis", tfjsonpath.New("kpi").AtSliceIndex(0).AtMapKey("id")),
						plancheck.ExpectKnownValue("itsi_service.test_kpis", tfjsonpath.New("kpi").AtSliceIndex(1).AtMapKey("id"), knownvalue.NotNull()),
						plancheck.ExpectKnownValue("itsi_service.test_kpis", tfjsonpath.New("kpi").AtSliceIndex(2).AtMapKey("id"), knownvalue.NotNull()),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.#", "3"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.0.title", "KPI 1"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.0.base_search_metric", "metric 1.3"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.0.base_search_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.0.threshold_template_id"),

					resource.TestCheckResourceAttrWith("itsi_service.test_kpis", "kpi.1.id", verifyKpiId(1)),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.1.title", "KPI 2"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.1.base_search_metric", "metric 1.2"),

					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.1.description", "test"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.1.urgency", "3"),

					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.1.base_search_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.1.threshold_template_id"),

					resource.TestCheckResourceAttrWith("itsi_service.test_kpis", "kpi.2.id", verifyKpiId(2)),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.2.title", "KPI 3"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.2.base_search_metric", "metric 2.1"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.2.base_search_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.2.threshold_template_id"),
				),
			}, // remove kpis
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.#", "0"),
				),
			},
		},
	})
}

func TestAccResourceServiceKpisUnknown(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckResourceDestroy(resourceNameService, "TestAcc_Test_service_kpis"),
			testAccCheckResourceDestroy(resourceNameKPIBaseSearch, "TestAcc_linked_base_search_1"),
			testAccCheckResourceDestroy(resourceNameKPIThresholdTemplate, "TestAcc_stdev_test_linked_kpi_threshold_template_1"),
			testAccCheckResourceDestroy(resourceNameKPIThresholdTemplate, "TestAcc_stdev_test_linked_kpi_threshold_template_2"),
		),
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "title", "TestAcc_Test_service_kpis"),
				),
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "title", "TestAcc_Test_service_kpis"),
				),
			},
		},
	})
}

func verifyKpiId(index int) resource.CheckResourceAttrWithFunc {
	return func(value string) error {
		fmt.Printf("Verifying KPI Ids on index %d\n", index)
		if len(PREV_KPI_IDS) < index {
			return fmt.Errorf("Unexpected lenght of the state array: PREV_KPI_IDS")
		}

		if value != PREV_KPI_IDS[index] {
			return fmt.Errorf("Failed: expected kpi.id %s, got %s", PREV_KPI_IDS[index], value)
		}

		return nil
	}
}

var PREV_KPI_IDS []string

func SaveKpiIds(s *terraform.State) error {
	fmt.Printf("Saving current KPI Ids\n")
	PREV_KPI_IDS = []string{}
	resource, ok := s.RootModule().Resources["itsi_service.test_kpis"]
	if !ok {
		return fmt.Errorf("Not found: itsi_service.test_kpis")
	}
	kpiLength, err := strconv.Atoi(resource.Primary.Attributes["kpi.#"])
	if err != nil {
		return fmt.Errorf("Kpi depends on length not found")
	}
	for i := 0; i < kpiLength; i++ {
		kpiId := resource.Primary.Attributes["kpi."+strconv.Itoa(i)+".id"]
		PREV_KPI_IDS = append(PREV_KPI_IDS, kpiId)
		fmt.Printf("Saving Prev State: Adding id %s\n", kpiId)
	}

	return nil
}
