---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "itsi_notable_event_aggregation_policy Resource - itsi"
subcategory: ""
description: |-
  Manages a Notable Event Aggregation Policy object within ITSI.
---

# itsi_notable_event_aggregation_policy (Resource)

Manages a Notable Event Aggregation Policy object within ITSI.

## Example Usage

```terraform
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
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `title` (String) The title of the notable event aggregation policy.

### Optional

- `breaking_criteria` (Block, Optional) Criteria to break an episode.
When the criteria is met, the current episode ends and a new one is created. (see [below for nested schema](#nestedblock--breaking_criteria))
- `description` (String) The description of the notable event aggregation policy.
- `disabled` (Boolean) Whether the notable event aggregation policy is disabled.
- `entity_factor_enabled` (Boolean) Whether the entity factor is enabled.
- `filter_criteria` (Block, Optional) Criteria to include events in an episode.
Any notable event that matches the criteria is included in the episode. (see [below for nested schema](#nestedblock--filter_criteria))
- `group_assignee` (String) The default owner of each episode created by the notable event aggregation policy. (Episode Asignee)
- `group_custom_instruction` (String) The custom instruction of each episode created by the notable event aggregation policy.
- `group_dashboard` (String) Customize the Episode dashboard using a JSON-formatted dashboard definition. The first notable event's fields are available to use as tokens in the dashboard.
- `group_dashboard_context` (String) Dashboard Tokens
- `group_description` (String) The description of each episode created by the notable event aggregation policy. (Episode Description)
- `group_instruction` (String) The default instructions of each episode created by the notable event aggregation policy.
- `group_severity` (String) The default severity of each episode created by the notable event aggregation policy. (Episode Severity)
- `group_status` (String) The default status of each episode created by the notable event aggregation policy.  (Episode Status)
- `group_title` (String) The default title of each episode created by the notable event aggregation policy. (Episode Title)
- `priority` (Number)
- `rule` (Block List) (see [below for nested schema](#nestedblock--rule))
- `run_time_based_actions_once` (Boolean) If you create an action to add a comment after an episode has existed for 60 seconds, a comment will only be added once for the episode.
There are 2 conditions that trigger time-based actions:
- The episode existed for (duration)
- The flow of events into the episode paused for (pause)
- `service_topology_enabled` (Boolean) Whether the service topology is enabled.
- `split_by_field` (Set of String) Fields to split an episode by.
- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))

### Read-Only

- `id` (String) ID of the NEAP.

<a id="nestedblock--breaking_criteria"></a>
### Nested Schema for `breaking_criteria`

Optional:

- `clause` (Block Set) A set of conditions that would be evaluated against the notable event fields. (see [below for nested schema](#nestedblock--breaking_criteria--clause))
- `duration` (Block Set) Corresponds to the statement: if the episode existed for %%param.duration%% seconds. (see [below for nested schema](#nestedblock--breaking_criteria--duration))
- `notable_event_count` (Block Set) Corresponds to the statement: if the number of events in this episode is %%operator%% %%limit%%. (see [below for nested schema](#nestedblock--breaking_criteria--notable_event_count))
- `pause` (Block Set) Corresponds to the statement: if the flow of events into the episode paused for %%param.pause%% seconds. (see [below for nested schema](#nestedblock--breaking_criteria--pause))

Read-Only:

- `breaking_criteria` (Block Set) Corresponds to the statement: if the episode is broken. Note: applicable only for the Activation Criteria. (see [below for nested schema](#nestedblock--breaking_criteria--breaking_criteria))
- `condition` (String) Computed depends of the criteria type. In case of activation_criteria condition equals AND, otherwise - OR.

<a id="nestedblock--breaking_criteria--clause"></a>
### Nested Schema for `breaking_criteria.clause`

Optional:

- `condition` (String)
- `notable_event_field` (Block Set) (see [below for nested schema](#nestedblock--breaking_criteria--clause--notable_event_field))

<a id="nestedblock--breaking_criteria--clause--notable_event_field"></a>
### Nested Schema for `breaking_criteria.clause.notable_event_field`

Required:

- `field` (String)
- `operator` (String)
- `value` (String) A wildcard pattern to match against a field value. E.g.: "*"



<a id="nestedblock--breaking_criteria--duration"></a>
### Nested Schema for `breaking_criteria.duration`

Required:

- `limit` (Number)


<a id="nestedblock--breaking_criteria--notable_event_count"></a>
### Nested Schema for `breaking_criteria.notable_event_count`

Required:

- `limit` (Number)
- `operator` (String)


<a id="nestedblock--breaking_criteria--pause"></a>
### Nested Schema for `breaking_criteria.pause`

Required:

- `limit` (Number)


<a id="nestedblock--breaking_criteria--breaking_criteria"></a>
### Nested Schema for `breaking_criteria.breaking_criteria`

Read-Only:

- `config` (String)



<a id="nestedblock--filter_criteria"></a>
### Nested Schema for `filter_criteria`

Optional:

- `clause` (Block Set) A set of conditions that would be evaluated against the notable event fields. (see [below for nested schema](#nestedblock--filter_criteria--clause))
- `duration` (Block Set) Corresponds to the statement: if the episode existed for %%param.duration%% seconds. (see [below for nested schema](#nestedblock--filter_criteria--duration))
- `notable_event_count` (Block Set) Corresponds to the statement: if the number of events in this episode is %%operator%% %%limit%%. (see [below for nested schema](#nestedblock--filter_criteria--notable_event_count))
- `pause` (Block Set) Corresponds to the statement: if the flow of events into the episode paused for %%param.pause%% seconds. (see [below for nested schema](#nestedblock--filter_criteria--pause))

Read-Only:

- `breaking_criteria` (Block Set) Corresponds to the statement: if the episode is broken. Note: applicable only for the Activation Criteria. (see [below for nested schema](#nestedblock--filter_criteria--breaking_criteria))
- `condition` (String) Computed depends of the criteria type. In case of activation_criteria condition equals AND, otherwise - OR.

<a id="nestedblock--filter_criteria--clause"></a>
### Nested Schema for `filter_criteria.clause`

Optional:

- `condition` (String)
- `notable_event_field` (Block Set) (see [below for nested schema](#nestedblock--filter_criteria--clause--notable_event_field))

<a id="nestedblock--filter_criteria--clause--notable_event_field"></a>
### Nested Schema for `filter_criteria.clause.notable_event_field`

Required:

- `field` (String)
- `operator` (String)
- `value` (String) A wildcard pattern to match against a field value. E.g.: "*"



<a id="nestedblock--filter_criteria--duration"></a>
### Nested Schema for `filter_criteria.duration`

Required:

- `limit` (Number)


<a id="nestedblock--filter_criteria--notable_event_count"></a>
### Nested Schema for `filter_criteria.notable_event_count`

Required:

- `limit` (Number)
- `operator` (String)


<a id="nestedblock--filter_criteria--pause"></a>
### Nested Schema for `filter_criteria.pause`

Required:

- `limit` (Number)


<a id="nestedblock--filter_criteria--breaking_criteria"></a>
### Nested Schema for `filter_criteria.breaking_criteria`

Read-Only:

- `config` (String)



<a id="nestedblock--rule"></a>
### Nested Schema for `rule`

Optional:

- `actions` (Block List) (see [below for nested schema](#nestedblock--rule--actions))
- `activation_criteria` (Block, Optional) Criteria to activate the NEAP Action. (see [below for nested schema](#nestedblock--rule--activation_criteria))
- `description` (String) The description of the notable event aggregation policy rule.
- `priority` (Number) The priority of the notable event aggregation policy rule.
- `title` (String) The title of the notable event aggregation policy rule.

Read-Only:

- `id` (String) ID of the notable event aggregation policy rule.

<a id="nestedblock--rule--actions"></a>
### Nested Schema for `rule.actions`

Optional:

- `item` (Block Set) (see [below for nested schema](#nestedblock--rule--actions--item))

Read-Only:

- `condition` (String)

<a id="nestedblock--rule--actions--item"></a>
### Nested Schema for `rule.actions.item`

Optional:

- `change_owner` (String) Change the owner of the episode to the specified value.
- `change_severity` (String) Change the severity of the episode to the specified value.
- `change_status` (String) Change the status of the episode to the specified value.
- `comment` (String) Add a comment to the episode.
- `custom` (Block Set) (see [below for nested schema](#nestedblock--rule--actions--item--custom))
- `execute_on` (String) ExecutionCriteria is essentially the criteria answering: "on which events is ActionItem applicable".

<a id="nestedblock--rule--actions--item--custom"></a>
### Nested Schema for `rule.actions.item.custom`

Required:

- `config` (String) JSON-encoded custom action configuration.
- `type` (String) The name of the custom action.




<a id="nestedblock--rule--activation_criteria"></a>
### Nested Schema for `rule.activation_criteria`

Optional:

- `clause` (Block Set) A set of conditions that would be evaluated against the notable event fields. (see [below for nested schema](#nestedblock--rule--activation_criteria--clause))
- `duration` (Block Set) Corresponds to the statement: if the episode existed for %%param.duration%% seconds. (see [below for nested schema](#nestedblock--rule--activation_criteria--duration))
- `notable_event_count` (Block Set) Corresponds to the statement: if the number of events in this episode is %%operator%% %%limit%%. (see [below for nested schema](#nestedblock--rule--activation_criteria--notable_event_count))
- `pause` (Block Set) Corresponds to the statement: if the flow of events into the episode paused for %%param.pause%% seconds. (see [below for nested schema](#nestedblock--rule--activation_criteria--pause))

Read-Only:

- `breaking_criteria` (Block Set) Corresponds to the statement: if the episode is broken. Note: applicable only for the Activation Criteria. (see [below for nested schema](#nestedblock--rule--activation_criteria--breaking_criteria))
- `condition` (String) Computed depends of the criteria type. In case of activation_criteria condition equals AND, otherwise - OR.

<a id="nestedblock--rule--activation_criteria--clause"></a>
### Nested Schema for `rule.activation_criteria.clause`

Optional:

- `condition` (String)
- `notable_event_field` (Block Set) (see [below for nested schema](#nestedblock--rule--activation_criteria--clause--notable_event_field))

<a id="nestedblock--rule--activation_criteria--clause--notable_event_field"></a>
### Nested Schema for `rule.activation_criteria.clause.notable_event_field`

Required:

- `field` (String)
- `operator` (String)
- `value` (String) A wildcard pattern to match against a field value. E.g.: "*"



<a id="nestedblock--rule--activation_criteria--duration"></a>
### Nested Schema for `rule.activation_criteria.duration`

Required:

- `limit` (Number)


<a id="nestedblock--rule--activation_criteria--notable_event_count"></a>
### Nested Schema for `rule.activation_criteria.notable_event_count`

Required:

- `limit` (Number)
- `operator` (String)


<a id="nestedblock--rule--activation_criteria--pause"></a>
### Nested Schema for `rule.activation_criteria.pause`

Required:

- `limit` (Number)


<a id="nestedblock--rule--activation_criteria--breaking_criteria"></a>
### Nested Schema for `rule.activation_criteria.breaking_criteria`

Read-Only:

- `config` (String)




<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours).
- `delete` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours). Setting a timeout for a Delete operation is only applicable if changes are saved into state before the destroy operation occurs.
- `read` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours). Read operations occur during any refresh or planning operation when refresh is enabled.
- `update` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours).

## Import

Import is supported using the following syntax:

```shell
terraform import itsi_notable_event_aggregation_policy.example {{id}}
```
