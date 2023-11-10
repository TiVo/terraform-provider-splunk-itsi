package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

var supported_execute_actions = map[string]struct{}{
	"slack_adv":                         {},
	"email":                             {},
	"bigpanda_stateful":                 {},
	"itsi_event_action_send_to_phantom": {},
	"itsi_event_action_link_url":        {},
	"itsi_sample_event_action_ping":     {},
	"script":                            {},
	"pagerduty_v2":                      {},
}

type SDK_ISSUE_588 struct {
	message string
}

func (i SDK_ISSUE_588) Error() string {
	return "Deeply nested TypeMap under TypeSet will generate an empty distribution element on update of the nested attribute."
}

func notableEventAggregationPolicyTFFormat(b *models.Base) (string, error) {
	res := ResourceNotableEventAggregationPolicy()
	resData := res.Data(nil)
	d := populateNotableEventAggregationPolicyResourceData(context.Background(), b, resData)
	if len(d) > 0 {
		err := d[0].Validate()
		if err != nil {
			return "", err
		}
		return "", errors.New(d[0].Summary)
	}
	resourcetpl, err := NewResourceTemplate(resData, res.Schema, "title", "notable_event_aggregation_policy")
	if err != nil {
		return "", err
	}

	templateResource, err := newTemplate(resourcetpl)
	if err != nil {
		log.Fatal(err)
	}
	var tpl bytes.Buffer
	err = templateResource.Execute(&tpl, resourcetpl)
	if err != nil {
		return "", err
	}

	return cleanerRegex.ReplaceAllString(tpl.String(), ""), nil
}

func notableEventAggregationPolicyBase(clientConfig models.ClientConfig, key string, title string) *models.Base {
	base := models.NewBase(clientConfig, key, title, "notable_event_aggregation_policy")
	return base
}

func ResourceNotableEventAggregationPolicy() *schema.Resource {
	actionSchema := map[string]*schema.Schema{
		"execute_on": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "ExecutionCriteria is essentially the criteria answering: \"on which events is ActionItem applicable?\".",
			Default:          "GROUP",
			ValidateDiagFunc: util.CheckInputValidString([]string{"GROUP", "ALL", "FILTER", "THIS"}),
		},
		"notable_severity_change": {
			Type:             schema.TypeString,
			Optional:         true,
			ValidateDiagFunc: util.CheckInputValidString(*util.GetSupportedSeverities())},
		"notable_status_change": {
			Type:             schema.TypeString,
			Optional:         true,
			ValidateDiagFunc: util.CheckInputValidString(*util.GetSupportedStatuses()),
		},
		"notable_owner_change": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"notable_event_comment": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"itsi_event_action_send_to_phantom": {
			Type:        schema.TypeSet,
			Description: "Run a script stored in $SPLUNK_HOME/bin/scripts. Note: DEPRECATED",
			MaxItems:    1,
			Optional:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"ph_server": {
						Description: "The Phantom server to which to send the ITSI episode",
						Type:        schema.TypeString,
						Required:    true,
					},
					"ph_label": {
						Description: "Label which determines which playbooks to trigger.",
						Type:        schema.TypeString,
						Required:    true,
					},
				},
			},
		},
		"script": {
			Type:        schema.TypeSet,
			Description: "Run a script stored in $SPLUNK_HOME/bin/scripts. Note: DEPRECATED",
			MaxItems:    1,
			Optional:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"filename": {
						Type:     schema.TypeString,
						Required: true,
					},
				},
			},
		},
		"itsi_event_action_link_url": {
			Type: schema.TypeSet,
			Description: `Set options to associate an episode with an external URL.
			  Follow this stanza name with any number of the following
			  attribute/value pairs.
			  If you do not specify an entry for each attribute, Splunk will
			  use the default value.`,
			MaxItems: 1,
			Optional: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"operation": {
						Type:     schema.TypeString,
						Optional: true,
						Description: `Specifies the type of action to take on the URL.
							If "upsert", ITSI inserts or updates existing fields.
							If "delete", ITSI deletes the URL.`,
						Default:      "upsert",
						ValidateFunc: validation.StringInSlice([]string{"delete", "upsert"}, false),
					},
					"kwargs": {
						Type:        schema.TypeString,
						Description: "A dictionary of additional fields to pass to the URL.",
						Optional:    true,
						Default:     "",
					},
					"url": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The external link for drilldown purposes. The URL must start with with http:// or https://. Otherwise it is interpreted as a relative URI.",
					},
					"url_description": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "A description of the link destination. For display purposes only.",
						Default:     "",
					},
				},
			},
		},
		"itsi_sample_event_action_ping": {
			Type:        schema.TypeSet,
			MaxItems:    1,
			Description: "Given one or more ITSI episodes, ping the `host` in it.",
			Optional:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"host_to_ping": {
						Type:        schema.TypeString,
						Description: "Type the event field that contains the host that you want to ping in the Host field. For example, %server%.",
						Optional:    true,
						Default:     "%orig_host%",
					},
				},
			},
		},
		"bigpanda_stateful": {
			Type:        schema.TypeSet,
			Description: "Send a Stateful BigPanda Alert",
			MaxItems:    1,
			Optional:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"api_token": {
						Type:     schema.TypeString,
						Required: true,
					},
					"api_url": {
						Type:     schema.TypeString,
						Optional: true,
						Default:  "https://api.bigpanda.io",
					},
					"app_key": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "Override the integration for this alert by entering the App Key of another integration in BigPanda",
					},
					"host": {
						Type:     schema.TypeString,
						Optional: true,
						Default:  "",
						Description: `Main object that caused the alert. Can be the associated host or, if a host isn't relevant, 
							a service or an application. Ex: $result.bp_desc$`,
					},
					"description": {
						Type:        schema.TypeString,
						Optional:    true,
						Default:     "",
						Description: `Brief summary (max. 2048 characters) of the alert for certain monitoring tools. Ex: itsi-episode-status`,
					},
					"check": {
						Type:        schema.TypeString,
						Optional:    true,
						Default:     "",
						Description: `Secondary object or sub-item that caused the alert (often shown as an incident subtitle in BigPanda).`,
					},
					"status": {
						Type:        schema.TypeString,
						Optional:    true,
						Default:     "",
						Description: `Ex: $result.bp_status$. Status of the alert. One of [ ok, critical, warning, acknowledged ].`,
					},
					"parameters": {
						Type:        schema.TypeMap,
						Optional:    true,
						Description: "Extra parameters.",
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
					},
				},
			},
		},
		"pagerduty_v2": {
			Type:        schema.TypeSet,
			Description: "Send a Stateful PagerDuty Alert",
			MaxItems:    1,
			Optional:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"integration_url": {
						Type:     schema.TypeString,
						Optional: true,
						Default:  "https://events.pagerduty.com/v2/enqueue",
					},
					"default_routing_key": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "Default Integration for this alert by entering the App Key of another integration in PagerDuty",
					},
					"routing_key": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "Override the integration for this alert by entering the App Key of another integration in PagerDuty",
					},
					"source": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "Unique location of the affected system, often a hostname or FQDN",
					},
					"dedup_key": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "Deduplication key for correlating events.",
					},
					"severity": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "Severity of the described event; one of critical, error, warning, info, or ok (all clear)",
					},
					"summary": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "Text summary of the event",
					},
					"custom_details": {
						Type:     schema.TypeString,
						Optional: true,
						Default:  "",
						Description: `Additional details about the event and affected system. Must be valid JSON.
							Other standard Events v2 API parameters (timestamp, component, group, class, images, links)
							may be included in this JSON and will be properly sent.`,
					},
					"trigger_state": {
						Type:     schema.TypeString,
						Optional: true,
						Default:  "",
						Description: `The "trigger state" for this run of the alert check. This is used to implement stateful alerting.
							This parameter should contain a value that changes if and only if an update event should be sent to PagerDuty.
							If left empty, the value of the severity parameter will be used.
							Another common implementation is to compute the trigger condition (true or false) in SPL and place that value in a field that is specified here (eg. "$result.trigger$").
							If you wish to send updates to PagerDuty upon other changes as well (besides the threshold condition), you can concatenate the values of those additional fields here also. 
							For example, if you set this to "$result.threshold$,$result.severity$" then you can send updates to PagerDuty when the trigger threshold remains exceeded, but the severity increases (or decreases).`,
					},
					"pagerduty_service": {
						Type:        schema.TypeString,
						Optional:    true,
						Default:     "unknown",
						Description: `Technical Service in PagerDuty to which events should be routed`,
					},
				},
			},
		},
		"slack_adv": {
			Type:        schema.TypeSet,
			Description: "Send an advanced message to a Slack channel",
			MaxItems:    1,
			Optional:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"from_user": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "The name of the user sending the Slack message. By default this is \"Splunk\".",
						Default:     "Splunk",
					},
					"from_user_icon": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "URL to an icon to show as the avatar for the Slack message. By default this is a Splunk icon.",
						Default:     "https://s3-us-west-1.amazonaws.com/ziegfried-apps/slack-alerts/splunk-icon.png",
					},
					"webhook_url": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The webhook URL to send the Slack message requests to. This can be obtained by creating a new \"Incoming webhook\" integration in Slack.",
					},
					"channel": {
						Type:        schema.TypeString,
						Description: "Slack channel (or channels, separated by commas) to send the message to (should start with # or @); can include tokens for expansion, see below.",
						Required:    true,
					},
					"payload": {
						Type:        schema.TypeString,
						Description: "Enter the Slack message payload to send to the Slack channel(s). The payload can include tokens that will be expanded using the results of the search. The payload must follow the Slack message payload format: https://api.slack.com/reference/messaging/payload",
						Required:    true,
					},
					"webhook_url_override": {
						Type:        schema.TypeString,
						Optional:    true,
						Default:     "",
						Description: "You can override the Slack webhook URL here if you need to send the alert message to a different Slack team.",
					},
				},
			},
		},
		"email": {
			Type:        schema.TypeSet,
			Description: "Send email",
			MaxItems:    1,
			Optional:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"to": {
						Type:     schema.TypeSet,
						Required: true,
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
					},
					"cc": {
						Type:     schema.TypeSet,
						Optional: true,
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
					},
					"bcc": {
						Type:     schema.TypeSet,
						Optional: true,
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
					},
					"content_type": {
						Type:         schema.TypeString,
						Optional:     true,
						ValidateFunc: validation.StringInSlice([]string{"html", "plain"}, false),
					},
					"priority": {
						Type:             schema.TypeString,
						Optional:         true,
						Default:          "critical",
						ValidateDiagFunc: util.CheckInputValidString(*util.GetSupportedSeverities()),
					},
					"subject": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The email subject can include tokens that insert text based on the results of the search. e.g. $result.fieldname$",
					},
					"message": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The email message can include tokens that insert text based on the results of the search. e.g. Status $result.slack_footer_msg$ $result.email_msg$ Dashboard: $result.dashboard$ Runbook: $result.runbook$ View in Splunk ITSI: https://itsi_url/itsi_event_management",
					},
				},
			},
		},
	}
	criteriaSchema := map[string]*schema.Schema{
		"condition": {
			Computed:    true,
			Type:        schema.TypeString,
			Description: "Computed depends of the criteria type. In case of activation_criteria condition equals AND, otherwise - OR.",
		},
		"clause": {
			Optional: true,
			Type:     schema.TypeSet,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"condition": {
						Type:             schema.TypeString,
						Optional:         true,
						Default:          "AND",
						ValidateDiagFunc: util.CheckInputValidString([]string{"AND"}),
					},
					"notable_event_field": {
						Type:     schema.TypeSet,
						Required: true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"field": {
									Type:     schema.TypeString,
									Required: true,
								},
								"operator": {
									Type:             schema.TypeString,
									Required:         true,
									ValidateDiagFunc: util.CheckInputValidString([]string{"=", "!=", ">=", "<=", ">", "<"}),
								},
								"value": {
									Type:        schema.TypeString,
									Optional:    true,
									Description: "The field's value wildcard to match. Ex: *.",
								},
							},
						},
					},
				},
			},
		},
		"pause": {
			Optional:    true,
			Type:        schema.TypeSet,
			Description: "Corresponds to the statement: if the flow of events into the episode paused for %%param.pause%% seconds.",
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"limit": {
						Type:     schema.TypeInt,
						Required: true,
					},
				},
			},
		},
		"duration": {
			Optional:    true,
			Type:        schema.TypeSet,
			Description: "Corresponds to the statement: if this episode existed for %%duration%% seconds",
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"limit": {
						Type:     schema.TypeInt,
						Required: true,
					},
				},
			},
		},
		"notable_event_count": {
			Optional:    true,
			Type:        schema.TypeSet,
			Description: "Corresponds to the statement: if the number of events in this episode is %%operator%% %%limit%%.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"operator": {
						Type:             schema.TypeString,
						Required:         true,
						ValidateDiagFunc: util.CheckInputValidString([]string{"==", "!=", ">=", "<=", ">", "<"}),
					},
					"limit": {
						Type:     schema.TypeInt,
						Required: true,
					},
				},
			},
		},
		"breaking_criteria": {
			Optional:    true,
			Type:        schema.TypeSet,
			Description: "Corresponds to the statement: if the episode is broken. Note: applicable only for the Activation Criteria.",
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"config": {
						Type:     schema.TypeString,
						Optional: true,
						Default:  "",
					},
				},
			},
		},
	}
	return &schema.Resource{
		Description:   "Manages an Notable Event Aggregation Policy object within ITSI.",
		CreateContext: notableEventAggregationPolicyCreate,
		ReadContext:   notableEventAggregationPolicyRead,
		UpdateContext: notableEventAggregationPolicyUpdate,
		DeleteContext: notableEventAggregationPolicyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: notableEventAggregationPolicyImport,
		},
		Schema: map[string]*schema.Schema{
			"title": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The title of the notable event aggregation policy.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The description of the notable event aggregation policy.",
			},
			"disabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"priority": {
				Type:             schema.TypeInt,
				Optional:         true,
				Default:          5,
				ValidateDiagFunc: util.CheckInputValidInt(0, 10),
			},
			"split_by_field": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Field episode to split by",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"filter_criteria": {
				Type:     schema.TypeSet,
				MaxItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: criteriaSchema,
				},
				Description: "FilterCriteria represents the criteria which is responsible for tagging an incoming notable event with an existing policy.",
			},
			"breaking_criteria": {
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: criteriaSchema,
				},
				Description: "BreakingCriteria represents the criteria which retires an active group.",
			},
			"entity_factor_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"run_time_based_actions_once": {
				Type:     schema.TypeBool,
				Optional: true,
				Description: `if you create an action to add a comment after an episode has existed for 60 seconds, a comment will only be added once for the episode. There are 2 conditions that trigger time-based actions:
					- The episode existed for (duration)
					- The flow of events into the episode paused for (pause)`,
				Default: true,
			},
			"service_topology_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"group_title": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The default title of each episode created by the notable event aggregation policy. (Episode Title)",
				Default:     "%title%",
			},
			"group_description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The default owner of each episode created by the notable event aggregation policy. (Episode Description)",
				Default:     "%description%",
			},
			"group_severity": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "%severity%",
				Description: "The default severity of each episode created by the notable event aggregation policy. (Episode Severity)",
				ValidateDiagFunc: util.CheckInputValidString(append(*util.GetSupportedSeverities(), []string{
					"%severity%",
					"%last_severity%",
					"%lowest_severity%",
					"%highest_severity%"}...)),
			},
			"group_assignee": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The default owner of each episode created by the notable event aggregation policy. (Episode Asignee)",
				Default:     "%owner%",
			},
			"group_status": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The default status of each episode created by the notable event aggregation policy.  (Episode Asignee)",
				Default:     "%owner%",
				ValidateDiagFunc: util.CheckInputValidString(append(*util.GetSupportedStatuses(), []string{
					"%status%",
					"%last_status%"}...)),
			},
			"group_instruction": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The default instructions of each episode created by the notable event aggregation policy.",
				Default:     "%instruction%",
				ValidateDiagFunc: util.CheckInputValidString([]string{
					"%itsi_instruction%",
					"%last_instruction%",
					"%all_instruction%",
					"%custom_instruction%"}),
			},
			"group_custom_instruction": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The custom instruction of each episode created by the notable event aggregation policy.",
				Default:     "",
			},
			"group_dashboard": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Customize the Episode dashboard using a JSON-formatted dashboard definition. The first notable event's fields are available to use as tokens in the dashboard.",
				Default:     "",
			},
			"group_dashboard_context": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "Dashboard Tokens",
				Default:          "first",
				ValidateDiagFunc: util.CheckInputValidString([]string{"first", "last"}),
			},
			"rule": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"_key": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"description": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"title": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"priority": {
							Type:             schema.TypeInt,
							Optional:         true,
							Default:          5,
							ValidateDiagFunc: util.CheckInputValidInt(0, 10),
						},
						"activation_criteria": {
							Type:        schema.TypeSet,
							Description: "ActivationCriteria represents the criteria satisfying which a Rule is activated for an incoming notable event or an existing group of notables.",
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: criteriaSchema,
							},
						},
						"actions": {
							Type:     schema.TypeSet,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"condition": {
										Type:         schema.TypeString,
										Optional:     true,
										Default:      "AND",
										ValidateFunc: validation.StringInSlice([]string{"AND"}, false),
									},
									"item": {
										Type:     schema.TypeSet,
										Required: true,
										Elem: &schema.Resource{
											Schema: actionSchema,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func criteria(criteria_type string, criteria_data *schema.Set) (criteria map[string]interface{}, err error) {
	tf_criteria := criteria_data.List()
	if len(tf_criteria) != 1 {
		return nil, &SDK_ISSUE_588{}
	}
	_criteria := tf_criteria[len(tf_criteria)-1].(map[string]interface{})
	criteria = make(map[string]interface{})

	conflicting_item_types := []string{}
	switch criteria_type {
	case "filter_criteria":
		criteria["condition"] = "OR"
		conflicting_item_types = append(conflicting_item_types, []string{"pause", "duration", "notable_event_count", "breaking_criteria"}...)
	case "breaking_criteria":
		criteria["condition"] = "OR"
		conflicting_item_types = append(conflicting_item_types, "breaking_criteria")
	case "activation_criteria":
		criteria["condition"] = "AND"

	default:
		return nil, fmt.Errorf("unsupported criteria type: %s", criteria_type)
	}

	criteria["items"] = []interface{}{}
	delete(_criteria, "condition")

	for item_type, item_body := range _criteria {
		_item_body := item_body.(*schema.Set).List()
		for _, _item := range _item_body {
			item := map[string]interface{}{
				"type": item_type,
			}
			for _, conflicting_item_type := range conflicting_item_types {
				if item_type == conflicting_item_type {
					return nil, fmt.Errorf("unsupported: %s item cannot be child of %s", item_type, criteria_type)
				}
			}
			switch item_type {
			case "clause":
				clause := _item.(map[string]interface{})
				clause_items := []interface{}{}
				for _, clause_item := range clause["notable_event_field"].(*schema.Set).List() {
					_clause_item := clause_item.(map[string]interface{})
					clause_items = append(clause_items, map[string]interface{}{
						"type": "notable_event_field",
						"config": map[string]interface{}{
							"field":    _clause_item["field"].(string),
							"operator": _clause_item["operator"].(string),
							"value":    _clause_item["value"].(string),
						},
					})
				}
				item["config"] = map[string]interface{}{
					"condition": clause["condition"].(string),
					"items":     clause_items,
				}
			case "pause", "duration":
				item["config"] = map[string]interface{}{
					"limit": _item.(map[string]interface{})["limit"].(int),
				}

			case "notable_event_count":
				notable_event_count := _item.(map[string]interface{})
				item["config"] = map[string]interface{}{
					"operator": notable_event_count["operator"].(string),
					"limit":    notable_event_count["limit"].(int),
				}

			case "breaking_criteria":
				//noop
			default:
				return nil, fmt.Errorf("unsupported criteria item: %s", item_type)
			}
			criteria["items"] = append(criteria["items"].([]interface{}), item)
		}
	}

	return
}

func notableEventAggregationPolicy(ctx context.Context, d *schema.ResourceData, clientConfig models.ClientConfig) (config *models.Base, err error) {
	body := map[string]interface{}{}

	body["object_type"] = "notable_event_aggregation_policy"
	body["title"] = d.Get("title").(string)
	body["description"] = d.Get("description").(string)
	body["disabled"] = 0
	if disabled := d.Get("disabled"); disabled.(bool) {
		body["disabled"] = 1
	}

	bool_fields := []string{"run_time_based_actions_once", "service_topology_enabled", "entity_factor_enabled"}
	for _, bool_field := range bool_fields {
		body[bool_field] = d.Get(bool_field).(bool)
	}
	body["priority"] = d.Get("priority").(int)
	//group_status
	json_group_severity := d.Get("group_severity").(string)
	if value, ok := util.SeverityMap[json_group_severity]; ok {
		json_group_severity = strconv.Itoa(value.SeverityValue)
	}
	body["group_severity"] = json_group_severity

	json_group_status := d.Get("group_status").(string)
	if value, ok := util.StatusInfoMap[json_group_severity]; ok {
		json_group_status = strconv.Itoa(value)
	}
	body["group_status"] = json_group_status

	group_fields := []string{"group_assignee", "group_custom_instruction",
		"group_dashboard", "group_dashboard_context", "group_description", "group_instruction", "group_title"}

	for _, group_field := range group_fields {
		body[group_field] = d.Get(group_field).(string)
	}

	// TODO: support smart mode
	body["ace_enabled"] = 0

	split_by_field := []string{}
	for _, split := range d.Get("split_by_field").(*schema.Set).List() {
		split_by_field = append(split_by_field, split.(string))
	}

	body["split_by_field"] = strings.Join(split_by_field, ",")

	body["filter_criteria"], err = criteria("filter_criteria", d.Get("filter_criteria").(*schema.Set))
	if err != nil {
		return nil, err
	}

	body["breaking_criteria"], err = criteria("breaking_criteria", d.Get("breaking_criteria").(*schema.Set))
	if err != nil {
		return nil, err
	}
	rules := []interface{}{}
	for _, rule := range d.Get("rule").(*schema.Set).List() {
		rule_json := map[string]interface{}{}
		_rule := rule.(map[string]interface{})

		rule_json["activation_criteria"], err = criteria("activation_criteria", _rule["activation_criteria"].(*schema.Set))
		switch err.(type) {
		case nil: //nothing
		case *SDK_ISSUE_588:
			continue
		default:
			return nil, err
		}
		if key, ok := _rule["_key"]; ok && key != "" {
			rule_json["_key"] = key
		} else {
			rule_json["_key"], err = uuid.GenerateUUID()
			if err != nil {
				return nil, err
			}
		}
		rule_json["title"] = _rule["title"].(string)
		rule_json["description"] = _rule["description"].(string)
		rule_json["priority"] = _rule["priority"].(int)

		rule_json["actions"] = []interface{}{}

		for _, action := range _rule["actions"].(*schema.Set).List() {
			action_json := map[string]interface{}{}
			_action := action.(map[string]interface{})
			action_json["condition"] = _action["condition"].(string)
			action_json["items"] = []interface{}{}
			for _, item := range _action["item"].(*schema.Set).List() {
				_item := item.(map[string]interface{})
				execute_on := _item["execute_on"].(string)
				delete(_item, "execute_on")
				item_json := map[string]interface{}{}

				item_json["execution_criteria"] = map[string]interface{}{
					"execute_on": execute_on,
				}
				if notable_event_comment, ok := _item["notable_event_comment"]; ok && notable_event_comment != "" {
					item_json["type"] = "notable_event_comment"
					item_json["config"] = map[string]interface{}{
						"value": notable_event_comment.(string),
					}
				} else if notable_severity_change, ok := _item["notable_severity_change"]; ok && notable_severity_change != "" {
					item_json["type"] = "notable_event_change"
					item_json["config"] = map[string]interface{}{
						"field": "severity",
						"value": strconv.FormatInt(int64(util.SeverityMap[notable_severity_change.(string)].SeverityValue), 10),
					}
				} else if notable_status_change, ok := _item["notable_status_change"]; ok && notable_status_change != "" {
					item_json["type"] = "notable_event_change"
					item_json["config"] = map[string]interface{}{
						"field": "status",
						"value": strconv.FormatInt(int64(util.StatusInfoMap[notable_status_change.(string)]), 10),
					}
				} else if notable_owner_change, ok := _item["notable_owner_change"]; ok && notable_owner_change != "" {
					item_json["type"] = "notable_event_change"
					item_json["config"] = map[string]interface{}{
						"field": "owner",
						"value": notable_owner_change.(string),
					}
				} else {
					item_json["type"] = "notable_event_execute_action"
					params := map[string]interface{}{}
					var action_type = ""
					for supported_execute_action := range supported_execute_actions {
						if action_type_params := _item[supported_execute_action].(*schema.Set).List(); len(action_type_params) == 1 {
							action_type = supported_execute_action

							for k, v := range action_type_params[0].(map[string]interface{}) {
								params[k] = v
							}
						}
					}
					action_prefix := "action." + action_type + "."
					if action_type != "" {
						switch action_type {
						case "email":
							for _, addr := range []string{"to", "cc", "bcc"} {
								emails := []string{}
								if _addr, ok := params[addr]; ok {
									for _, email := range _addr.(*schema.Set).List() {
										emails = append(emails, email.(string))
									}
									params[addr] = strings.Join(emails, ",")
								}
							}
						case "bigpanda_stateful":
							action_prefix += "param."
							if tf_extra_params, ok := params["parameters"]; ok {
								json_extra_params := ""

								for k, v := range tf_extra_params.(map[string]interface{}) {
									json_extra_params += fmt.Sprintf("%s='%s' ", k, v.(string))

								}
								params["parameters"] = json_extra_params
							}

						case "script":
							// no additional actions required
						default:
							action_prefix += "param."
						}

					} else {
						return nil, fmt.Errorf("unsupported action type %s", action_type)
					}
					// TF to JSON schema modifications (ex: convert email.to from set to string)

					params_json := map[string]interface{}{}
					for key, value := range params {
						params_json[action_prefix+key] = value
					}

					bytes, err := json.Marshal(params_json)
					if err != nil {
						return nil, err
					}

					item_json["config"] = map[string]interface{}{
						"name":   action_type,
						"params": string(bytes),
					}
				}
				action_json["items"] = append(action_json["items"].([]interface{}), item_json)
			}

			rule_json["actions"] = append(rule_json["actions"].([]interface{}), action_json)
		}
		rules = append(rules, rule_json)
	}
	body["rules"] = rules
	base := notableEventAggregationPolicyBase(clientConfig, d.Id(), d.Get("title").(string))
	err = base.PopulateRawJSON(ctx, body)

	return base, err
}

func criteriaResourceData(itsi_criteria map[string]interface{}) (tf_criteria map[string]interface{}, err error) {
	tf_criteria = make(map[string]interface{})
	tf_criteria["condition"] = itsi_criteria["condition"]
	if items, ok := itsi_criteria["items"]; ok {
		for _, item := range items.([]interface{}) {
			_item := item.(map[string]interface{})
			tf_item := map[string]interface{}{}
			tf_item_type := _item["type"].(string)
			switch tf_item_type {
			case "notable_event_count":
				config := _item["config"].(map[string]interface{})
				tf_item["operator"] = config["operator"].(string)
				switch t := config["limit"].(type) {
				case float64:
					tf_item["limit"] = config["limit"].(float64)
				case string:
					tf_item["limit"], err = strconv.Atoi(config["limit"].(string))
				default:
					return nil, fmt.Errorf("unsupported notable_event_count limit type: %t", t)
				}

			case "pause", "duration":
				config := _item["config"].(map[string]interface{})
				switch config["limit"].(type) {
				case float64:
					tf_item["limit"] = config["limit"].(float64)
				case string:
					tf_item["limit"], err = strconv.Atoi(config["limit"].(string))
				default:
					return nil, fmt.Errorf("unsupported %s limit type", _item["type"].(string))
				}
			case "breaking_criteria":
				tf_item["config"] = ""
			case "clause":
				config := _item["config"].(map[string]interface{})
				tf_item["condition"] = config["condition"]
				tf_item["notable_event_field"] = []interface{}{}
				for _, clause_item := range config["items"].([]interface{}) {
					_clause_item := clause_item.(map[string]interface{})

					if _clause_item_config, ok := _clause_item["config"]; ok {
						clause_item_config := _clause_item_config.(map[string]interface{})
						tf_item["notable_event_field"] = append(tf_item["notable_event_field"].([]interface{}), map[string]interface{}{
							"value":    clause_item_config["value"].(string),
							"operator": clause_item_config["operator"].(string),
							"field":    clause_item_config["field"].(string),
						})
					} else {
						return nil, fmt.Errorf("unsupported notable_event_field criteria: no config found %s", _clause_item)
					}
				}
			}
			if _, ok := tf_criteria[tf_item_type]; ok {
				tf_criteria[tf_item_type] = append(tf_criteria[tf_item_type].([]interface{}), tf_item)
			} else {
				tf_criteria[tf_item_type] = []interface{}{tf_item}
			}
		}
	} else {
		return nil, fmt.Errorf("unsupported criteria without items")
	}
	return
}

func populateNotableEventAggregationPolicyResourceData(ctx context.Context, b *models.Base, d *schema.ResourceData) (diags diag.Diagnostics) {
	interfaceMap, err := b.RawJson.ToInterfaceMap()
	if err != nil {
		return diag.FromErr(err)
	}

	for _, f := range []string{"title", "description"} {
		err = d.Set(f, interfaceMap[f])
		if err != nil {
			return diag.FromErr(err)
		}
	}
	switch t := interfaceMap["priority"].(type) {
	case float64:
		err = d.Set("priority", interfaceMap["priority"].(float64))
		if err != nil {
			return diag.FromErr(err)
		}
	case string:
		priority := interfaceMap["priority"].(string)
		if priority == "" {
			err = d.Set("priority", 0)
			if err != nil {
				return diag.FromErr(err)
			}
		} else {
			_priority, err := strconv.Atoi(priority)
			if err != nil {
				return diag.FromErr(err)
			}
			err = d.Set("priority", _priority)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	default:
		diag.FromErr(fmt.Errorf("usupported type %s for priority", t))
	}

	if split_by_field, ok := interfaceMap["split_by_field"]; ok && split_by_field != "" {
		err = d.Set("split_by_field", strings.Split(split_by_field.(string), ","))
	}

	if err != nil {
		return diag.FromErr(err)
	}
	group_severity := interfaceMap["group_severity"].(string)
	severity_label, err := strconv.Atoi(group_severity)
	if err == nil {
		severity_info, err := util.GetSeverityInfoByValue(severity_label)
		if err != nil {
			return diag.FromErr(err)
		}
		err = d.Set("group_severity", severity_info.SeverityLabel)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		err = d.Set("group_severity", group_severity)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	group_status := interfaceMap["group_status"].(string)
	group_status_label, err := strconv.Atoi(group_status)
	if err == nil {
		status_label, err := util.GetStatusInfoByValue(group_status_label)
		if err != nil {
			return diag.FromErr(err)
		}
		err = d.Set("group_status", status_label)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		err = d.Set("group_status", group_status)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	group_fields := []string{"group_assignee", "group_custom_instruction",
		"group_dashboard", "group_dashboard_context", "group_description", "group_instruction", "group_title"}
	for _, group_field := range group_fields {
		if json_value, ok := interfaceMap[group_field]; ok {
			err = d.Set(group_field, json_value.(string))
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}
	bool_fields := []string{"disabled", "run_time_based_actions_once", "service_topology_enabled", "entity_factor_enabled"}
	for _, bool_field := range bool_fields {
		switch t := interfaceMap[bool_field].(type) {
		case bool:
			err = d.Set(bool_field, interfaceMap[bool_field].(bool))
			if err != nil {
				return diag.FromErr(err)
			}
		case float64:
			err = d.Set(bool_field, interfaceMap[bool_field].(float64) > 0)
			if err != nil {
				return diag.FromErr(err)
			}
		case string:
			disabled, err := strconv.Atoi(interfaceMap[bool_field].(string))
			if err != nil {
				return diag.FromErr(err)
			}
			err = d.Set(bool_field, disabled > 0)
			if err != nil {
				return diag.FromErr(err)
			}

		default:
			diag.FromErr(fmt.Errorf("usupported type %s for priority", t))
		}
	}

	filter_criteria, err := criteriaResourceData(interfaceMap["filter_criteria"].(map[string]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("filter_criteria", []interface{}{filter_criteria})
	if err != nil {
		return diag.FromErr(err)
	}

	breaking_criteria, err := criteriaResourceData(interfaceMap["breaking_criteria"].(map[string]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("breaking_criteria", []interface{}{breaking_criteria})
	if err != nil {
		return diag.FromErr(err)
	}
	tf_rules := []interface{}{}
	for _, rule := range interfaceMap["rules"].([]interface{}) {
		tf_rule := map[string]interface{}{}
		itsi_rule := rule.(map[string]interface{})

		tf_rule["_key"] = itsi_rule["_key"]
		tf_rule["description"] = itsi_rule["description"]
		tf_rule["title"] = itsi_rule["title"]
		switch t := itsi_rule["priority"].(type) {
		case float64:
			tf_rule["priority"] = itsi_rule["priority"].(float64)
		case string:
			priority := itsi_rule["priority"].(string)
			if priority == "" {
				tf_rule["priority"] = 0
			} else {
				tf_rule["priority"], err = strconv.Atoi(priority)
				if err != nil {
					return diag.FromErr(err)
				}
			}
		default:
			diag.FromErr(fmt.Errorf("usupported type for rule priority %s", t))
		}

		activation_criteria, err := criteriaResourceData(itsi_rule["activation_criteria"].(map[string]interface{}))
		if err != nil {
			return diag.FromErr(err)
		}
		tf_rule["activation_criteria"] = []interface{}{activation_criteria}
		tf_rule["actions"] = []interface{}{}
		for _, action := range itsi_rule["actions"].([]interface{}) {
			itsi_action := action.(map[string]interface{})
			tf_action := map[string]interface{}{
				"condition": itsi_action["condition"].(string),
				"item":      []interface{}{},
			}
			for _, item := range itsi_action["items"].([]interface{}) {
				itsi_item := item.(map[string]interface{})
				tf_item := map[string]interface{}{}

				if execution_criteria, ok := itsi_item["execution_criteria"]; ok {
					if execute_on, ok := execution_criteria.(map[string]interface{})["execute_on"]; ok {
						tf_item["execute_on"] = execute_on.(string)
					}
				}
				switch itsi_item["type"].(string) {
				case "notable_event_comment":
					if config, ok := itsi_item["config"]; ok {
						if value, ok := config.(map[string]interface{})["value"]; ok {
							tf_item["notable_event_comment"] = value.(string)
						}
					}
					if _, ok := tf_item["notable_event_comment"]; !ok {
						return diag.FromErr(fmt.Errorf("wrong notable_event_comment action %s", itsi_item))
					}

				case "notable_event_change":
					if config, ok := itsi_item["config"]; ok {
						if field, ok := config.(map[string]interface{})["field"]; ok {

							value := config.(map[string]interface{})["value"]
							switch field.(string) {
							case "severity":
								severity_v := -1
								switch value.(type) {
								case string:
									severity_v, err = strconv.Atoi(value.(string))
									if err != nil {
										return diag.FromErr(err)
									}
								case float64:
									severity_v = int(value.(float64))
								}

								severity_m, err := util.GetSeverityInfoByValue(severity_v)
								tf_item["notable_severity_change"] = severity_m.SeverityLabel
								if err != nil {
									return diag.FromErr(err)
								}

							case "status":
								status_v := -1
								switch value.(type) {
								case string:
									status_v, err = strconv.Atoi(value.(string))
									if err != nil {
										return diag.FromErr(err)
									}
								case float64:
									status_v = int(value.(float64))
								}
								if err != nil {
									return diag.FromErr(err)
								}
								tf_item["notable_status_change"], err = util.GetStatusInfoByValue(status_v)
								if err != nil {
									return diag.FromErr(err)
								}
							case "owner":
								tf_item["notable_owner_change"] = value.(string)
							default:
								return diag.FromErr(fmt.Errorf("unsupported notable_event_change action %s", itsi_item))
							}
						} else {
							return diag.FromErr(fmt.Errorf("unsupported notable_event_change structure, missed field"))
						}
					} else {
						return diag.FromErr(fmt.Errorf("unsupported notable_event_change structure, missed config"))
					}
				case "notable_event_execute_action":
					if config, ok := itsi_item["config"]; ok {
						if name, ok := config.(map[string]interface{})["name"]; ok {
							_name := name.(string)
							if params, ok := config.(map[string]interface{})["params"]; ok {
								if _, ok := supported_execute_actions[_name]; ok {
									_params := params.(string)
									/*params_unquoted, err := strconv.Unquote(_params)
									if err != nil {
										return diag.FromErr(err)
									}*/
									var execute_action map[string]interface{}
									err = json.Unmarshal([]byte(_params), &execute_action)
									if err != nil {
										return diag.FromErr(err)
									}

									tf_execute_action := map[string]interface{}{}

									// trim action,%%action_name%% prefix
									for key, value := range execute_action {
										trimmed_key := strings.TrimPrefix(key, "action."+_name+".")
										// applicable to slack_adv, bigpanda_stateful
										trimmed_key = strings.TrimPrefix(trimmed_key, "param.")
										switch {
										case _name == "email" && (trimmed_key == "to" || trimmed_key == "bcc" || trimmed_key == "cc"):
											if value != "" {
												names := strings.Split(value.(string), ",")
												for i := range names {
													names[i] = strings.TrimSpace(names[i])
												}

												tf_execute_action[trimmed_key] = names
											}
										case _name == "bigpanda_stateful" && trimmed_key == "parameters":
											rex := regexp.MustCompile(`([^ =]*)[ ]*=[ ]*'([^']*)' `)
											matches := rex.FindAllStringSubmatch(value.(string), -1)
											extra_params := map[string]interface{}{}
											for _, v := range matches {
												extra_params[v[1]] = v[2]
											}
											tf_execute_action[trimmed_key] = extra_params
										default:
											tf_execute_action[trimmed_key] = value
										}

									}
									tf_item[_name] = []interface{}{tf_execute_action}
								}

								if _, ok := tf_item[_name]; !ok {
									return diag.FromErr(fmt.Errorf("unsupported notable_execute_action action %s", itsi_item))
								}

							} else {
								return diag.FromErr(fmt.Errorf("unsupported notable_execute_action structure, missed param"))
							}
						} else {
							return diag.FromErr(fmt.Errorf("unsupported notable_execute_action structure, missed name"))
						}
					} else {
						return diag.FromErr(fmt.Errorf("unsupported notable_execute_action structure, missed config"))
					}
				default:
					return diag.FromErr(fmt.Errorf("unsupported action %s", itsi_item))

				}
				tf_action["item"] = append(tf_action["item"].([]interface{}), tf_item)
			}

			tf_rule["actions"] = append(tf_rule["actions"].([]interface{}), tf_action)
		}

		tf_rules = append(tf_rules, tf_rule)
	}
	err = d.Set("rule", tf_rules)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(b.RESTKey)
	return nil
}

func notableEventAggregationPolicyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	template, err := notableEventAggregationPolicy(ctx, d, m.(models.ClientConfig))
	if err != nil {
		return diag.FromErr(err)
	}
	b, err := template.Create(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	b.Read(ctx)
	return populateNotableEventAggregationPolicyResourceData(ctx, b, d)
}

func notableEventAggregationPolicyRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	base := notableEventAggregationPolicyBase(m.(models.ClientConfig), d.Id(), d.Get("title").(string))
	b, err := base.Find(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	if b == nil || b.RawJson == nil {
		d.SetId("")
		return nil
	}
	return populateNotableEventAggregationPolicyResourceData(ctx, b, d)
}

func notableEventAggregationPolicyUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	clientConfig := m.(models.ClientConfig)
	base := notableEventAggregationPolicyBase(clientConfig, d.Id(), d.Get("title").(string))
	existing, err := base.Find(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	if existing == nil {
		return notableEventAggregationPolicyCreate(ctx, d, m)
	}

	template, err := notableEventAggregationPolicy(ctx, d, clientConfig)
	if err != nil {
		return diag.FromErr(err)
	}
	return diag.FromErr(template.Update(ctx))
}

func notableEventAggregationPolicyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	base := notableEventAggregationPolicyBase(m.(models.ClientConfig), d.Id(), d.Get("title").(string))
	existing, err := base.Find(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	if existing == nil {
		return diag.Errorf("Unable to find neap model")
	}
	return diag.FromErr(existing.Delete(ctx))
}

func notableEventAggregationPolicyImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	b := notableEventAggregationPolicyBase(m.(models.ClientConfig), "", d.Id())
	b, err := b.Find(ctx)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, err
	}
	diags := populateNotableEventAggregationPolicyResourceData(ctx, b, d)
	for _, d := range diags {
		if d.Severity == diag.Error {
			return nil, fmt.Errorf(d.Summary)
		}
	}

	if d.Id() == "" {
		return nil, nil
	}
	return []*schema.ResourceData{d}, nil
}
