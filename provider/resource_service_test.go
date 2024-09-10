package provider

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
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
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckResourceDestroy(resourceNameService, "TestAcc_Test_Service_Create_filter_test"),
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
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckResourceDestroy(resourceNameService, "TestAcc_Test_Tag_Lifecycle"),
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

func testCheckServiceDependsOnMatch(t *testing.T, child_name string, expected_overloaded_urgency ...int) resource.TestCheckFunc {
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
				t.Logf("PASSED: Leaf shkpi_id %s, Parent's dependent kpis %s", leafKPIID, parentResource.Primary.Attributes["service_depends_on.0.kpis"])

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

func TestAccResourceServiceDeletedInUI(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckResourceDestroy(resourceNameService, "TestAcc_ResourceServiceDeletedInUI_service"),
			testAccCheckResourceDestroy(resourceNameKPIBaseSearch, "TestAcc_ResourceServiceDeletedInUI_helper_kpi_bs"),
			testAccCheckResourceDestroy(resourceNameKPIThresholdTemplate, "TestAcc_ResourceServiceDeletedInUI_helper_threshold_template"),
		),
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check:                    resource.ComposeTestCheckFunc(),
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				SkipFunc: func() (bool, error) {
					return true, EmulateUiDelete(t, "TestAcc_ResourceServiceDeletedInUI_service", "service")
				},
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check:                    resource.ComposeTestCheckFunc(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("itsi_service.test_ui_delete", plancheck.ResourceActionCreate),
					},
				},
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check:                    resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func EmulateUiDelete(t *testing.T, title string, object_type string) error {
	ctx := context.Background()
	t.Logf("Skip function, emulating removing from UI.")
	client := configureTestClient()
	base := models.NewBase(client, "", title, object_type)
	base, err := base.Find(ctx)
	if err != nil {
		t.Logf("%s %s not found: %s", object_type, title, err.Error())
		return err
	}

	t.Logf("Firing the search")
	diags := base.Delete(ctx)
	if diags.HasError() {
		t.Logf("%s %s failed to delete: %s", object_type, title, diags[0].Summary())
		return fmt.Errorf(diags[0].Summary())
	}
	t.Logf("%s %s was deleted successfully", object_type, title)
	return nil

}

func configureTestClient() models.ClientConfig {
	client := models.ClientConfig{}
	insecure := configBoolValueWithEnvFallback(types.BoolNull(), envITSIInsecure)
	client.BearerToken = configStringValueWithEnvFallback(types.StringNull(), envITSIAccessToken)
	client.User = configStringValueWithEnvFallback(types.StringNull(), envITSIUser)
	client.Password = configStringValueWithEnvFallback(types.StringNull(), envITSIPassword)
	client.Host = configStringValueWithEnvFallback(types.StringNull(), envITSIHost)
	client.Port = int(configIntValueWithEnvFallback(types.Int64Null(), envITSIPort))
	client.Timeout = defaultTimeout
	client.SkipTLS = insecure
	client.RetryPolicy = retryPolicy
	client.Concurrency = clientConcurrency

	return client

}

func TestAccResourceServiceKpisHandleUnknownTemplateId(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckResourceDestroy(resourceNameService, "TestAcc_ResourceServiceKpisHandleUnknownTemplateId_Test_service_kpis_2"),
			testAccCheckResourceDestroy(resourceNameKPIBaseSearch, "TestAcc_ResourceServiceKpisHandleUnknownTemplateId_linked_base_search_3"),
			testAccCheckResourceDestroy(resourceNameKPIBaseSearch, "TestAcc_ResourceServiceKpisHandleUnknownTemplateId_linked_base_search_4"),
			testAccCheckResourceDestroy(resourceNameKPIThresholdTemplate, "TestAcc_ResourceServiceKpisHandleUnknownTemplateId_static_kpi_threshold_template_1"),
			testAccCheckResourceDestroy(resourceNameKPIThresholdTemplate, "TestAcc_ResourceServiceKpisHandleUnknownTemplateId_stdev_test_linked_kpi_threshold_template_3"),
		),
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
			},
			// add kpi
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
						plancheck.ExpectKnownValue("itsi_service.test_kpis_2", tfjsonpath.New("kpi").AtSliceIndex(0).AtMapKey("threshold_template_id"), knownvalue.NotNull()),
						plancheck.ExpectUnknownValue("itsi_service.test_kpis_2", tfjsonpath.New("kpi").AtSliceIndex(1).AtMapKey("threshold_template_id")),
						plancheck.ExpectKnownValue("itsi_service.test_kpis_2", tfjsonpath.New("kpi").AtSliceIndex(2).AtMapKey("threshold_template_id"), knownvalue.NotNull()),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis_2", "kpi.0.threshold_template_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis_2", "kpi.1.threshold_template_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis_2", "kpi.2.threshold_template_id"),
				),
			}, // change metric of KPI 1 (ID regenerated), unit & description of KPI 2 (ID should stay the same), remove KPI 3
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check:                    resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestAccResourceServiceHandleUnknownKpiBaseSearchId(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckResourceDestroy(resourceNameService, "TestAcc_ResourceServiceHandleUnknownKpiBaseSearchId_Test_service_kpis_2"),
			testAccCheckResourceDestroy(resourceNameKPIBaseSearch, "TestAcc_ResourceServiceHandleUnknownKpiBaseSearchId_linked_base_search_4"),
			testAccCheckResourceDestroy(resourceNameKPIBaseSearch, "TestAcc_ResourceServiceHandleUnknownKpiBaseSearchId_linked_base_search_3"),
			testAccCheckResourceDestroy(resourceNameKPIThresholdTemplate, "TestAcc_ResourceServiceHandleUnknownKpiBaseSearchId_static_kpi_threshold_template_1"),
			testAccCheckResourceDestroy(resourceNameKPIThresholdTemplate, "TestAcc_ResourceServiceHandleUnknownKpiBaseSearchId_stdev_test_linked_kpi_threshold_template_3"),
		),
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				//Check:                    Sleep(t, 30*time.Second),
			},
			// add kpi linked to new kpi bs
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
						plancheck.ExpectKnownValue("itsi_service.test_kpis_2", tfjsonpath.New("kpi").AtSliceIndex(0).AtMapKey("base_search_id"), knownvalue.NotNull()),
						plancheck.ExpectKnownValue("itsi_service.test_kpis_2", tfjsonpath.New("kpi").AtSliceIndex(1).AtMapKey("base_search_id"), knownvalue.NotNull()),
						plancheck.ExpectUnknownValue("itsi_service.test_kpis_2", tfjsonpath.New("kpi").AtSliceIndex(2).AtMapKey("base_search_id")),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis_2", "kpi.0.base_search_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis_2", "kpi.1.base_search_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis_2", "kpi.2.base_search_id"),
				),
			}, // remove kpis
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check:                    resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestAccResourceServiceDependsOnLifecycle(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckResourceDestroy(resourceNameService, "TestAcc_Service_Test_on_Create_Parent"),
			testAccCheckResourceDestroy(resourceNameService, "TestAcc_Service_Test_on_Create_Leaf"),
			testAccCheckResourceDestroy(resourceNameService, "TestAcc_Service_Test_on_Create_Leaf_Overloaded"),
		),
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("itsi_service.service_create_leaf", "shkpi_id"),
					resource.TestCheckResourceAttr("itsi_service.service_create_parent", "service_depends_on.#", "1"),
					testCheckServiceDependsOnMatch(t, "itsi_service.service_create_leaf"),
				),
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("itsi_service.service_create_leaf", "shkpi_id"),
					resource.TestCheckResourceAttrSet("itsi_service.service_create_leaf_overloaded", "shkpi_id"),
					resource.TestCheckResourceAttr("itsi_service.service_create_parent", "service_depends_on.#", "2"),
					testCheckServiceDependsOnMatch(t, "itsi_service.service_create_leaf"),
					testCheckServiceDependsOnMatch(t, "itsi_service.service_create_leaf_overloaded", 8),
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
					testCheckServiceDependsOnMatch(t, "itsi_service.service_create_leaf"),
					testCheckServiceDependsOnMatch(t, "itsi_service.service_create_leaf_overloaded", 8),
				),
			},
		},
	})
}

func TestAccResourceServiceKpisLifecycle(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckResourceDestroy(resourceNameService, "TestAcc_ResourceServiceKpisLifecycle_Test_service_kpis"),
			testAccCheckResourceDestroy(resourceNameKPIBaseSearch, "TestAcc_ResourceServiceKpisLifecycle_linked_base_search_1"),
			testAccCheckResourceDestroy(resourceNameEntityType, "TestAcc_ResourceServiceKpisLifecycle_linked_base_search_2"),
			testAccCheckResourceDestroy(resourceNameKPIThresholdTemplate, "TestAcc_ResourceServiceKpisLifecycle_static_kpi_threshold_template_1"),
			testAccCheckResourceDestroy(resourceNameKPIThresholdTemplate, "TestAcc_ResourceServiceKpisLifecycle_stdev_test_linked_kpi_threshold_template_1"),
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

					SaveKpiIds(t),
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
					resource.TestCheckResourceAttrWith("itsi_service.test_kpis", "kpi.0.id", verifyKpiId(t, 0)),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.0.title", "KPI 1"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.0.base_search_metric", "metric 1.1"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.0.base_search_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.0.threshold_template_id"),

					resource.TestCheckResourceAttrWith("itsi_service.test_kpis", "kpi.1.id", verifyKpiId(t, 1)),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.1.title", "KPI 2"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.1.base_search_metric", "metric 1.2"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.1.base_search_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.1.threshold_template_id"),

					resource.TestCheckResourceAttrWith("itsi_service.test_kpis", "kpi.2.id", verifyKpiId(t, 2)),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.2.title", "KPI 3"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.2.base_search_metric", "metric 2.1"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.2.base_search_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.2.threshold_template_id"),

					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.3.id"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.3.title", "KPI 4"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.3.base_search_metric", "metric 2.2"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.3.base_search_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.3.threshold_template_id"),

					SaveKpiIds(t),
					//Sleep(t, 30*time.Second),
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
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectKnownValue("itsi_service.test_kpis", tfjsonpath.New("kpi").AtSliceIndex(1).AtMapKey("description"), knownvalue.StringExact("test")),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.#", "3"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.0.title", "KPI 1"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.0.base_search_metric", "metric 1.3"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.0.base_search_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.0.threshold_template_id"),

					resource.TestCheckResourceAttrWith("itsi_service.test_kpis", "kpi.1.id", verifyKpiId(t, 1)),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.1.title", "KPI 2"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.1.base_search_metric", "metric 1.2"),

					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.1.description", "test"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.1.urgency", "3"),

					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.1.base_search_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.1.threshold_template_id"),

					resource.TestCheckResourceAttrWith("itsi_service.test_kpis", "kpi.2.id", verifyKpiId(t, 2)),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.2.title", "KPI 3"),
					resource.TestCheckResourceAttr("itsi_service.test_kpis", "kpi.2.base_search_metric", "metric 2.1"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.2.base_search_id"),
					resource.TestCheckResourceAttrSet("itsi_service.test_kpis", "kpi.2.threshold_template_id"),
				),
			}, // remove description of KPI 2
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectKnownValue("itsi_service.test_kpis", tfjsonpath.New("kpi").AtSliceIndex(1).AtMapKey("description"), knownvalue.StringExact("")),
					},
				},
			},
			// remove kpis
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

func TestAccResourceServiceUnknownKPIs(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckResourceDestroy(resourceNameService, "TestAcc_ServiceUnknownKPIs_KPIBS"),
			testAccCheckResourceDestroy(resourceNameKPIBaseSearch, "TestAcc_ServiceUnknownKPIs_ThresholdTemplate"),
			testAccCheckResourceDestroy(resourceNameEntityType, "TestAcc_ServiceUnknownKPIs_Service"),
		),
		Steps: []resource.TestStep{
			{
				// Create a service with an unknown number of KPIs
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
			},
			{
				// Update the service with an unknown number of KPIs
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
			},
			{
				// Remove the service, so that other resources can be destroyed by the testing framework afterwards
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
			},
		},
	})
}

func verifyKpiId(t *testing.T, index int) resource.CheckResourceAttrWithFunc {
	return func(value string) error {
		t.Logf("Verifying KPI Ids on index %d", index)
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

func SaveKpiIds(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		t.Log("Saving current KPI Ids")
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
			t.Logf("Saving Prev State: Adding id %s", kpiId)
		}

		return nil
	}
}

// func Sleep(t *testing.T, d time.Duration) resource.TestCheckFunc {
// 	return func(s *terraform.State) error {
// 		t.Logf("Sleeping for %s ...", d.String())
// 		time.Sleep(d)
// 		return nil
// 	}
// }
