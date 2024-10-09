package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/tivo/terraform-provider-splunk-itsi/util"
)

func TestResourceNEAPSchema(t *testing.T) {
	testResourceSchema(t, new(resourceNEAP))
}

func TestResourceNEAPPlan(t *testing.T) {
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

					resource "itsi_notable_event_aggregation_policy" "policy_example" {
					  breaking_criteria {
					    duration {
					      limit = 86400
					    }
					  }
					  description = "This is the development version of the policy that will group together notable events by their alert_group field."
					  disabled    = true
					  filter_criteria {
					    clause {
					      notable_event_field {
					        field    = "alert_group"
					        operator = "="
					        value    = "*"
					      }
					      notable_event_field {
					        field    = "tier"
					        operator = "="
					        value    = "1"
					      }
					    }
					  }
					  group_instruction = "%last_instruction%"
					  group_severity    = "info"
					  group_status      = "%last_status%"
					  group_title       = "%alert_group%"
					  rule {
					    actions {
					      item {
					        change_status = "closed"
					      }
					      item {
					        comment = "Episode status automatically changed to Closed due to NEAP action triggered when episode is broken."
					      }
					    }
					    activation_criteria {
					      breaking_criteria {
					      }
					      notable_event_count {
					        limit    = 1
					        operator = "=="
					      }
					    }
					  }
					  rule {
					    actions {
					      item {
					        custom {
					          type = "email"
					          config = jsonencode({
					            content_type = "html"
					            message      = <<-EOT
					                          Status $result.slack_footer_msg$

					                          $result.email_msg$
					                          EOT
					            priority     = "normal"
					            subject      = "$result.slack_header$"
					            to = [
					              "teammember_1@outlook.com",
					              "teammember_2@outlook.com",
					            ]
					          })
					        }
					      }

					      item {
					        custom {
					          type = "itsi_event_action_link_url"
					          config = jsonencode({
					            "param.url"             = "https://your_ref_link.com"
					            "param.url_description" = "some description"
					          })
					        }
					      }

					      item {
					        custom {
					          type = "script"
					          config = jsonencode({
					            filename = "test.sh"
					          })
					        }
					      }
					    }

					    activation_criteria {
					      clause {
					        notable_event_field {
					          field    = "email_custom_action"
					          operator = "="
					          value    = "1"
					        }
					        notable_event_field {
					          field    = "level"
					          operator = "="
					          value    = "3"
					        }
					      }
					    }
					  }
					  rule {
					    actions {
					      item {
					        change_severity = "low"
					      }
					      item {
					        comment = "Changed episode priority to Low"
					      }
					    }
					    activation_criteria {
					      clause {
					        notable_event_field {
					          field    = "alert_value"
					          operator = ">="
					          value    = "100"
					        }
					        notable_event_field {
					          field    = "level"
					          operator = ">="
					          value    = "3"
					        }
					      }
					    }
					  }
					  run_time_based_actions_once = false
					  split_by_field = [
					    "alert_group", "host"
					  ]
					  title = "Dev: Episodes by Alert Group"
					}

				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccResourceNEAPLifecycle(t *testing.T) {
	t.Parallel()
	var testAccResourceNEAPLifecycle_NEAPTitle = testAccResourceTitle("ResourceNEAPLifecycle_neap_test")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckResourceDestroy(resourceNameNEAP, testAccResourceNEAPLifecycle_NEAPTitle),
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_notable_event_aggregation_policy.test", "title", testAccResourceNEAPLifecycle_NEAPTitle),
					resource.TestCheckResourceAttr("itsi_notable_event_aggregation_policy.test", "description", "abc"),
					//testAccCheckNEAPExists,
				),
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_notable_event_aggregation_policy.test", "description", "TEST DESCRIPTION update"),
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

var NEAP_ID string

func TestAccResourceNEAPDeletedInUI(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckResourceDestroy(resourceNameNEAP, "TestAcc_ResourceNEAPDeleted_UI_neap_test"),
		),
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						t.Log("Saving current NEAP ID")
						resource, ok := s.RootModule().Resources["itsi_notable_event_aggregation_policy.test"]
						if !ok {
							return fmt.Errorf("Not found: itsi_notable_event_aggregation_policy.test")
						}
						NEAP_ID = resource.Primary.Attributes["id"]

						return nil
					},
				),
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				SkipFunc: func() (bool, error) {
					return true, EmulateUiDelete(t, NEAP_ID, "notable_event_aggregation_policy")
				},
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check:                    resource.ComposeTestCheckFunc(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("itsi_notable_event_aggregation_policy.test", plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}
