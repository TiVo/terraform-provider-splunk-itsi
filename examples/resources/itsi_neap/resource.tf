resource "itsi_neap" "policy_example" {
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
        notable_status_change = "closed"
      }
      item {
        notable_event_comment = "Episode status automatically changed to Closed due to NEAP action triggered when episode is broken."
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
        email {
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
        }
      }
      item {
        bigpanda_stateful {
          api_token   = "override with your token"
          app_key     = "override with your app key"
          check       = "itsi-episode-status"
          description = "$result.bp_desc$"
          parameters = {
            alert_group      = "$result.alert_group$"
            primary_property = "alert_group"
          }
          status = "$result.bp_status$"
        }
      }
      item {
        slack_adv {
          channel     = "%slack_channel%"
          payload     = <<-EOT
                        {
                        	"blocks": [
                        		{
                        			"type": "section",
                        			"text": {
                        				"type": "mrkdwn",
                        				"text": "*Status*:  %slack_change_icon% *Actions*: %slack_actions_msg%"
                        			}
                        		}
                        	]
                        }
                        EOT
          webhook_url = "https://hooks.slack.com/services/your_webhook"
        }
      }
      item {
        itsi_event_action_link_url {
          url             = "https://your_ref_link.com"
          url_description = "some description"
        }
      }
      item {
        script {
          filename = "test.sh"
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
        notable_severity_change = "low"
      }
      item {
        notable_event_comment = "Changed episode priority to Low"
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
