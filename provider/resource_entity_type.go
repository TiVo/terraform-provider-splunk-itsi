package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

func entityTypeTFFormat(b *models.Base) (string, error) {
	res := ResourceEntityType()
	resData := res.Data(nil)
	d := populateEntityTypeResourceData(context.Background(), b, resData)
	if len(d) > 0 {
		err := d[0].Validate()
		if err != nil {
			return "", err
		}
		return "", errors.New(d[0].Summary)
	}
	resourcetpl, err := NewResourceTemplate(resData, res.Schema, "title", "itsi_entity_type")
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

func ResourceEntityType() *schema.Resource {
	return &schema.Resource{
		Description: `An entity_type defines how to classify a type of data source.
						For example, you can create a Linux, Windows, Unix/Linux add-on, VMware, or Kubernetes entity type.
						An entity type can include zero or more data drilldowns and zero or more dashboard drilldowns.
						You can use a single data drilldown or dashboard drilldown for multiple entity types.`,
		CreateContext: entityTypeCreate,
		ReadContext:   entityTypeRead,
		UpdateContext: entityTypeUpdate,
		DeleteContext: entityTypeDelete,
		Importer: &schema.ResourceImporter{
			StateContext: entityTypeImport,
		},
		Schema: map[string]*schema.Schema{
			"title": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the entity type.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "A description of the entity type.",
			},
			"vital_metric": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: getVitalMetricSchema(),
				},
				Description: `An array of vital metric objects. Vital metrics are statistical calculations based on 
							  SPL searches that represent the overall health of entities of that type.`,
			},
			"dashboard_drilldown": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: getDashboardDrilldownSchema(),
				},
				Description: "An array of dashboard drilldown objects. Each dashboard drilldown defines an internal or external resource you specify with a URL and parameters that map to one of an entity fields. The parameters are passed to the resource when you open the URL.",
			},
			"data_drilldown": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: getDataDrilldownSchema(),
				},
				Description: "An array of data drilldown objects. Each data drilldown defines filters for raw data associated with entities that belong to the entity type.",
			},
		},
	}
}

func getDataDrilldownSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"title": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Name of the drilldown.",
		},
		"type": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringInSlice([]string{"events", "metrics"}, false),
			Description:  "Type of raw data to associate with. Must be either metrics or events.",
		},
		"static_filters": {
			Type:        schema.TypeMap,
			Optional:    true,
			Description: "Filter down to a subset of raw data associated with the entity using static information like sourcetype.",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"entity_field_filter": {
			Type:     schema.TypeSet,
			Required: true,
			Description: `Further filter down to the raw data associated with the entity 
						  based on a set of selected entity alias or informational fields.`,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"data_field": {
						Type:     schema.TypeString,
						Required: true,
					},
					"entity_field": {
						Type:     schema.TypeString,
						Required: true,
					},
				},
			},
		},
	}
}

func getDashboardDrilldownSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"title": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Name of the dashboard.",
		},
		"base_url": {
			Type:     schema.TypeString,
			Optional: true,
			Description: `An internal or external URL that points to the dashboard. This setting exists because for internal purposes, navigation suggestions are treated as dashboards.
							This setting is only required if is_splunk_dashboard is false.`,
		},
		"dashboard_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: `A unique identifier for the xml dashboard.`,
		},
		"dashboard_type": {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      "navigation_link",
			ValidateFunc: validation.StringInSlice([]string{"xml_dashboard", "navigation_link"}, false),
			Description: `The type of dashboard being added. This element is required. The following options are available:
							xml_dashboard - a Splunk XML dashboard. Any dashboards you add must be of this type.
							navigation_link - a navigation URL. Should be used when base_url is specified.`,
		},
		"params": {
			Type:     schema.TypeMap,
			Optional: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Description: "A set of parameters for the entity dashboard drilldown that provide a mapping of a URL parameter and its alias",
		},
	}
}

func getVitalMetricSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"metric_name": {
			Type:     schema.TypeString,
			Required: true,
			Description: `The title of the vital metric. When creating vital metrics,
					  it's a best practice to include the aggregation method and the name of the metric being calculated.
					  For example, Average CPU usage.`,
		},
		"search": {
			Type:     schema.TypeString,
			Required: true,
			Description: `The search that computes the vital metric. The search must specify the following fields:
							- val for the value of the metric.
							- _time because the UI attempts to render changes over time. You can achieve this by adding span={time} to your search.
							- Fields as described in the split_by_fields configuration of this vital metric.
							For example, your search should be split by host,region if the split_by_fields configuration is [ "host", "region" ].`,
		},
		"matching_entity_fields": {
			Type:     schema.TypeMap,
			Required: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Description: `Specifies the aliases of an entity to use to match with the fields specified by the fields that the search configuration is split on.
						Make sure the value matches the split by fields in the actual search.
						For example:
							- search = "..... by InstanceId, region"
							- matching_entity_fields = {instance_id = "InstanceId", zone = "region"}.`,
		},
		"is_key": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
			Description: `Indicates if the vital metric specified is a key metric.
						A key metric calculates the distribution of entities associated with the entity type to indicate the overall health of the entity type.
						The key metric is rendered as a histogram in the Infrastructure Overview. Only one vital metric can have is_key set to true. `,
		},
		"unit": {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     "",
			Description: "The unit of the vital metric. For example, KB/s. ",
		},
		"alert_rule": {
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"suppress_time": {
						Type:        schema.TypeString,
						Optional:    true,
						Default:     "0",
						Description: "suppress the alert until this time",
					},
					"cron_schedule": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "frequency of alert search",
					},
					"is_enabled": {
						Type:        schema.TypeBool,
						Optional:    true,
						Default:     false,
						Description: "if alert is enabled",
					},
					"warning_threshold": {
						Type:     schema.TypeInt,
						Required: true,
					},
					"critical_threshold": {
						Type:     schema.TypeInt,
						Required: true,
					},
					"entity_filter": {
						Type:     schema.TypeSet,
						Optional: true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"field": {
									Type:     schema.TypeString,
									Required: true,
								},
								"value": {
									Type:     schema.TypeString,
									Required: true,
								},
								"field_type": {
									Type:         schema.TypeString,
									Required:     true,
									Description:  "Takes values alias, info or title specifying in which category of fields the field attribute is located.",
									ValidateFunc: validation.StringInSlice([]string{"alias", "entity_type", "info", "title"}, false),
								},
							},
						},
					},
				},
			},
		},
	}
}

func entityType(ctx context.Context, d *schema.ResourceData, clientConfig models.ClientConfig) (config *models.Base, err error) {
	body := map[string]interface{}{}

	body["object_type"] = "entity_type"
	body["sec_grp"] = "default_itsi_security_group"
	body["title"] = d.Get("title").(string)
	body["description"] = d.Get("description").(string)

	body_vital_metrics := []interface{}{}

	if vital_metrics, ok := d.GetOk("vital_metric"); ok {
		for _, vital_metric := range vital_metrics.(*schema.Set).List() {
			body_vital_metric := map[string]interface{}{}

			_vital_metric := vital_metric.(map[string]interface{})
			if metric_name := _vital_metric["metric_name"].(string); metric_name != "" {
				body_vital_metric["metric_name"] = metric_name
				body_vital_metric["search"] = _vital_metric["search"].(string)

				body_vital_metric["matching_entity_fields"] = []string{}
				body_vital_metric["split_by_fields"] = []string{}

				matching_entity_fields := _vital_metric["matching_entity_fields"].(map[string]interface{})

				for alias, split_by_field := range matching_entity_fields {
					body_vital_metric["matching_entity_fields"] = append(body_vital_metric["matching_entity_fields"].([]string), alias)
					body_vital_metric["split_by_fields"] = append(body_vital_metric["split_by_fields"].([]string), split_by_field.(string))
				}
				isKey := 0
				if _vital_metric["is_key"].(bool) {
					isKey = 1
				}
				body_vital_metric["is_key"] = isKey
				body_vital_metric["unit"] = _vital_metric["unit"].(string)
				body_alert_rule := map[string]interface{}{}
				for idx, alert_rule := range _vital_metric["alert_rule"].(*schema.Set).List() {
					if idx > 0 {
						return nil, fmt.Errorf("more than one alert rule is passed in metric: %s", metric_name)
					}
					_alert_rule := alert_rule.(map[string]interface{})

					body_alert_rule["suppress_time"] = _alert_rule["suppress_time"].(string)
					body_alert_rule["cron_schedule"] = _alert_rule["cron_schedule"].(string)
					body_alert_rule["is_enabled"] = _alert_rule["is_enabled"].(bool)

					warning_threshold := _alert_rule["warning_threshold"].(int)
					critical_threshold := _alert_rule["critical_threshold"].(int)
					if warning_threshold < critical_threshold {
						// greater than
						body_alert_rule["critical_threshold"] = []string{strconv.Itoa(critical_threshold), "+inf"}
						body_alert_rule["warning_threshold"] = []string{strconv.Itoa(warning_threshold), strconv.Itoa(critical_threshold)}
						body_alert_rule["info_threshold"] = []string{"-inf", strconv.Itoa(warning_threshold)}
					} else {
						// less than
						body_alert_rule["critical_threshold"] = []string{"-inf", strconv.Itoa(critical_threshold)}
						body_alert_rule["warning_threshold"] = []string{strconv.Itoa(critical_threshold), strconv.Itoa(warning_threshold)}
						body_alert_rule["info_threshold"] = []string{strconv.Itoa(warning_threshold), "+inf"}
					}
					body_entity_filters := []map[string]string{}

					if entity_filters, ok := _alert_rule["entity_filter"]; ok {
						for _, entity_filter := range entity_filters.(*schema.Set).List() {

							_entity_filter := entity_filter.(map[string]interface{})

							if field, _ := _entity_filter["field"].(string); field != "" {
								body_entity_filter := map[string]string{}
								body_entity_filter["field"] = field
								body_entity_filter["value"] = _entity_filter["value"].(string)
								body_entity_filter["field_type"] = _entity_filter["field_type"].(string)

								body_entity_filters = append(body_entity_filters, body_entity_filter)
							}
						}
					}
					body_alert_rule["entity_filter"] = body_entity_filters

				}
				body_vital_metric["alert_rule"] = body_alert_rule
				body_vital_metrics = append(body_vital_metrics, body_vital_metric)
			}
		}
	}
	body["vital_metrics"] = body_vital_metrics

	body_data_drilldowns := []interface{}{}
	if data_drilldowns, ok := d.GetOk("data_drilldown"); ok {
		for _, data_drilldown := range data_drilldowns.(*schema.Set).List() {
			body_data_drilldown := map[string]interface{}{}

			_data_drilldown := data_drilldown.(map[string]interface{})
			if data_drilldown_title := _data_drilldown["title"].(string); data_drilldown_title != "" {
				body_data_drilldown["title"] = data_drilldown_title
				body_data_drilldown["type"] = _data_drilldown["type"].(string)
				body_static_filters := []interface{}{}
				for _static_filter_key, _static_filter_value := range _data_drilldown["static_filters"].(map[string]interface{}) {
					body_static_filters = append(body_static_filters, map[string]interface{}{
						"type":   "include",
						"field":  _static_filter_key,
						"values": []string{_static_filter_value.(string)},
					})
				}
				if len(body_static_filters) == 1 {
					body_data_drilldown["static_filter"] = body_static_filters[0]
				} else {
					body_data_drilldown["static_filter"] = map[string]interface{}{
						"type":    "and",
						"filters": body_static_filters,
					}
				}

				body_entity_field_filters := []interface{}{}
				for _, entity_field_filter := range _data_drilldown["entity_field_filter"].(*schema.Set).List() {

					_entity_field_filter := entity_field_filter.(map[string]interface{})
					body_entity_field_filters = append(body_entity_field_filters, map[string]interface{}{
						"type":         "entity",
						"data_field":   _entity_field_filter["data_field"].(string),
						"entity_field": _entity_field_filter["entity_field"].(string),
					})
				}
				if len(body_entity_field_filters) > 1 {
					body_data_drilldown["entity_field_filter"] = map[string]interface{}{
						"type":    "and",
						"filters": body_entity_field_filters,
					}
				} else {
					body_data_drilldown["entity_field_filter"] = body_entity_field_filters[0]
				}

				body_data_drilldowns = append(body_data_drilldowns, body_data_drilldown)
			}
		}
	}
	body["data_drilldowns"] = body_data_drilldowns

	body_dashboard_drilldowns := []interface{}{}
	if dashboard_drilldowns, ok := d.GetOk("dashboard_drilldown"); ok {
		for _, dashboard_drilldown := range dashboard_drilldowns.(*schema.Set).List() {
			_dashboard_drilldown := dashboard_drilldown.(map[string]interface{})
			body_dashboard_drilldown := map[string]interface{}{}
			if _dashboard_drilldown_title, ok := _dashboard_drilldown["title"]; ok && _dashboard_drilldown_title != "" {
				body_dashboard_drilldown["title"] = _dashboard_drilldown_title

				body_dashboard_drilldown["dashboard_type"] = _dashboard_drilldown["dashboard_type"].(string)
				if body_dashboard_drilldown["dashboard_type"] != "xml_dashboard" {
					body_dashboard_drilldown["base_url"] = _dashboard_drilldown["base_url"].(string)
					body_dashboard_drilldown["id"] = _dashboard_drilldown_title.(string)
				} else {
					body_dashboard_drilldown["id"] = _dashboard_drilldown["dashboard_id"].(string)
					body_dashboard_drilldown["base_url"] = ""
				}
				body_params := []interface{}{}
				for alias, param := range _dashboard_drilldown["params"].(map[string]interface{}) {
					body_params = append(body_params, map[string]interface{}{
						"alias": alias,
						"param": param.(string),
					})
				}
				body_dashboard_drilldown["params"] = map[string]interface{}{
					"static_params":   map[string]interface{}{},
					"alias_param_map": body_params,
				}
				body_dashboard_drilldowns = append(body_dashboard_drilldowns, body_dashboard_drilldown)
			}
		}
	}
	body["dashboard_drilldowns"] = body_dashboard_drilldowns
	base := entityTypeBase(clientConfig, d.Id(), d.Get("title").(string))
	err = base.PopulateRawJSON(ctx, body)

	return base, err
}

func entityTypeCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	template, err := entityType(ctx, d, m.(models.ClientConfig))
	tflog.Info(ctx, "ENTITY TYPE: create", map[string]interface{}{"TFID": template.TFID, "err": err})
	if err != nil {
		return diag.FromErr(err)
	}
	b, err := template.Create(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	b, err = b.Read(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	return populateEntityTypeResourceData(ctx, b, d)
}

func populateEntityTypeResourceData(ctx context.Context, b *models.Base, d *schema.ResourceData) (diags diag.Diagnostics) {
	interfaceMap, err := b.RawJson.ToInterfaceMap()
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("title", interfaceMap["title"])
	if err != nil {
		return diag.FromErr(err)
	}
	exists := d.Get("description") != nil
	if exists {
		err = d.Set("description", interfaceMap["description"])
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.Get("vital_metric") != nil {
		data_vital_metrics := []interface{}{}
		if vital_metrics, exists := interfaceMap["vital_metrics"].([]interface{}); exists {
			for _, vital_metric := range vital_metrics {
				_vital_metric := vital_metric.(map[string]interface{})
				matching_entity_fields := map[string]string{}
				split_by_fields := _vital_metric["split_by_fields"].([]interface{})

				if len(_vital_metric["matching_entity_fields"].([]interface{})) != len(split_by_fields) {
					return diag.Errorf("Error: matching_entity_fields should be same length with split_by_fields")
				}
				for idx, vital_metric := range _vital_metric["matching_entity_fields"].([]interface{}) {
					matching_entity_fields[vital_metric.(string)] = split_by_fields[idx].(string)
				}
				_vital_metric["matching_entity_fields"] = matching_entity_fields
				alert_rules := []interface{}{}
				switch _vital_metric["is_key"].(type) {
				case float64:
					_vital_metric["is_key"] = _vital_metric["is_key"].(float64) > 0
				}

				delete(_vital_metric, "split_by_fields")

				if alert_rule, ok := _vital_metric["alert_rule"].(map[string]interface{}); ok && len(alert_rule) > 0 {
					_alert_rule := map[string]interface{}{}

					critical_threshold := alert_rule["critical_threshold"].([]interface{})
					warning_threshold := alert_rule["warning_threshold"].([]interface{})

					idx := 0
					if critical_threshold[0].(string) == "-inf" {
						idx = 1
					}
					_alert_rule["critical_threshold"], err = strconv.Atoi(critical_threshold[idx].(string))
					if err != nil {
						diag.FromErr(err)
					}
					_alert_rule["warning_threshold"], err = strconv.Atoi(warning_threshold[idx].(string))
					if err != nil {
						diag.FromErr(err)
					}
					_alert_rule["suppress_time"] = alert_rule["suppress_time"].(string)
					_alert_rule["cron_schedule"] = alert_rule["cron_schedule"].(string)
					_alert_rule["is_enabled"] = alert_rule["is_enabled"].(bool)
					if entity_filter, ok := alert_rule["entity_filter"]; ok {
						_alert_rule["entity_filter"] = entity_filter.([]interface{})
					}

					alert_rules = append(alert_rules, _alert_rule)
				}
				_vital_metric["alert_rule"] = alert_rules
				data_vital_metrics = append(data_vital_metrics, _vital_metric)
			}
			err = d.Set("vital_metric", data_vital_metrics)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}
	if d.Get("data_drilldown") != nil {
		data_data_drilldowns := []interface{}{}
		if _data_drilldowns, exists := interfaceMap["data_drilldowns"].([]interface{}); exists {
			for _, data_drilldown := range _data_drilldowns {
				_data_drilldown := data_drilldown.(map[string]interface{})
				data_data_drilldown := map[string]interface{}{}
				data_data_drilldown["title"] = _data_drilldown["title"].(string)
				data_data_drilldown["type"] = _data_drilldown["type"].(string)
				data_static_filters := map[string]interface{}{}
				filters := _data_drilldown["static_filter"].(map[string]interface{})
				if _, ok := filters["filters"]; !ok {
					filters["filters"] = []interface{}{filters}
				}

				for _, filter := range filters["filters"].([]interface{}) {
					_filter := filter.(map[string]interface{})
					_values := _filter["values"].([]interface{})
					if len(_values) > 1 {
						return diag.Errorf("expected only 1 value for static filter in %s", filter)
					}
					data_static_filters[_filter["field"].(string)] = _values[0].(string)
				}

				data_data_drilldown["static_filters"] = data_static_filters
				if entity_field_filter, exists := _data_drilldown["entity_field_filter"]; exists {
					_entity_field_filter := entity_field_filter.(map[string]interface{})
					if filters, ok := _entity_field_filter["filters"]; ok {
						data_data_drilldown["entity_field_filter"] = filters
					} else {
						data_data_drilldown["entity_field_filter"] = []interface{}{
							_entity_field_filter,
						}
					}
					for _, data_drilldown := range data_data_drilldown["entity_field_filter"].([]interface{}) {
						delete(data_drilldown.(map[string]interface{}), "type")
					}
				}
				data_data_drilldowns = append(data_data_drilldowns, data_data_drilldown)
			}
			err = d.Set("data_drilldown", data_data_drilldowns)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}
	if d.Get("dashboard_drilldown") != nil {
		data_dashboard_drilldowns := []interface{}{}
		if _dashboard_drilldowns, exists := interfaceMap["dashboard_drilldowns"].([]interface{}); exists {
			for _, dashboard_drilldown := range _dashboard_drilldowns {
				_dashboard_drilldown := dashboard_drilldown.(map[string]interface{})
				data_dashboard_drilldown := map[string]interface{}{}

				data_dashboard_drilldown["title"] = _dashboard_drilldown["title"].(string)
				data_dashboard_drilldown["dashboard_type"] = _dashboard_drilldown["dashboard_type"].(string)

				if data_dashboard_drilldown["dashboard_type"] != "xml_dashboard" {
					data_dashboard_drilldown["base_url"] = _dashboard_drilldown["base_url"].(string)
				} else if dashboard_id, ok := _dashboard_drilldown["id"]; ok {
					data_dashboard_drilldown["dashboard_id"] = dashboard_id.(string)
				}
				params := _dashboard_drilldown["params"].(map[string]interface{})
				data_params := map[string]interface{}{}
				if alias_param_map, ok := params["alias_param_map"]; ok {
					for _, alias_param_tuple := range alias_param_map.([]interface{}) {
						_alias_param_tuple := alias_param_tuple.(map[string]interface{})
						data_params[_alias_param_tuple["alias"].(string)] = _alias_param_tuple["param"].(string)
					}
				}
				data_dashboard_drilldown["params"] = data_params

				data_dashboard_drilldowns = append(data_dashboard_drilldowns, data_dashboard_drilldown)
			}
			err = d.Set("dashboard_drilldown", data_dashboard_drilldowns)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	d.SetId(b.RESTKey)
	return nil
}

func entityTypeUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	clientConfig := m.(models.ClientConfig)
	base := entityTypeBase(clientConfig, d.Id(), d.Get("title").(string))
	tflog.Info(ctx, "ENTITY TYPE: update", map[string]interface{}{"TFID": base.TFID})
	existing, err := base.Find(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	if existing == nil {
		return entityTypeCreate(ctx, d, m)
	}

	template, err := entityType(ctx, d, clientConfig)
	if err != nil {
		return diag.FromErr(err)
	}
	return diag.FromErr(template.Update(ctx))
}

func entityTypeDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	base := entityTypeBase(m.(models.ClientConfig), d.Id(), d.Get("title").(string))
	tflog.Info(ctx, "ENTITY TYPE: delete", map[string]interface{}{"TFID": base.TFID})
	existing, err := base.Find(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	if existing == nil {
		return diag.Errorf("Unable to find entity type model")
	}
	return diag.FromErr(existing.Delete(ctx))
}

func entityTypeImport(ctx context.Context, data *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	b := entityTypeBase(m.(models.ClientConfig), "", data.Id())
	b, err := b.Find(ctx)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, err
	}
	diags := populateEntityTypeResourceData(ctx, b, data)
	for _, d := range diags {
		if d.Severity == diag.Error {
			return nil, fmt.Errorf(d.Summary)
		}
	}

	if data.Id() == "" {
		return nil, nil
	}
	return []*schema.ResourceData{data}, nil
}
