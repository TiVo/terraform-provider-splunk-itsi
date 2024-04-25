package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
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
