resource "itsi_notable_event_aggregation_policy" "test" {
  title       = "TestAcc_ResourceNEAPLifecycle_neap_test"
  description = "TEST DESCRIPTION update"

  breaking_criteria {
    duration {
      limit = 86400
    }
  }
  disabled = true
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
}
