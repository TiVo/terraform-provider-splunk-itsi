package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/util"
)

const (
	itsiResourceTypeEntityType     = "entity_type"
	entityTypeDefaultDashboardType = "navigation_link"

	entityTypeDashboardTypeXMLDashboard = "xml_dashboard"
	entityTypeDashboardTypeUDFDashboard = "udf_dashboard"
	entityTypeDashboardTypeNavigation   = "navigation_link"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &resourceEntityType{}
	_ tfmodel           = &entityTypeModel{}
)

// =================== [ Entity Type ] ===================

type entityTypeModel struct {
	ID                 types.String                        `tfsdk:"id"`
	Title              types.String                        `tfsdk:"title"`
	Description        types.String                        `tfsdk:"description"`
	DashboardDrilldown []entityTypeDashboardDrilldownModel `tfsdk:"dashboard_drilldown"`
	DataDrilldown      []entityTypeDataDrilldownModel      `tfsdk:"data_drilldown"`
	VitalMetric        []entityTypeVitalMetricModel        `tfsdk:"vital_metric"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (et entityTypeModel) objectype() string {
	return itsiResourceTypeEntityType
}

func (et entityTypeModel) title() string {
	return et.Title.ValueString()
}

// =================== [ Entity Type / Dashboard Drilldown ] ===================

type entityTypeDashboardDrilldownModel struct {
	Title         types.String `tfsdk:"title"`
	BaseURL       types.String `tfsdk:"base_url"`
	DashboardID   types.String `tfsdk:"dashboard_id"`
	DashboardType types.String `tfsdk:"dashboard_type"`
	Params        types.Map    `tfsdk:"params"`
}

func (m *entityTypeDashboardDrilldownModel) getParams(ctx context.Context) (params map[string]string, diags diag.Diagnostics) {
	params = make(map[string]string)
	if !m.Params.IsNull() {
		diags = m.Params.ElementsAs(ctx, &params, false)
	}
	return
}

// =================== [ Entity Type / Data Drilldown ] ===================

type entityTypeDataDrilldownModel struct {
	Title             types.String                                    `tfsdk:"title"`
	Type              types.String                                    `tfsdk:"type"`
	StaticFilters     types.Map                                       `tfsdk:"static_filters"`
	EntityFieldFilter []entityTypeDataDrilldownEntityFieldFilterModel `tfsdk:"entity_field_filter"`
}

func (m *entityTypeDataDrilldownModel) getStaticFilters(ctx context.Context) (filters map[string]string, diags diag.Diagnostics) {
	filters = make(map[string]string)
	if !m.StaticFilters.IsNull() {
		diags = m.StaticFilters.ElementsAs(ctx, &filters, false)
	}
	return
}

type entityTypeDataDrilldownEntityFieldFilterModel struct {
	DataField   types.String `tfsdk:"data_field"`
	EntityField types.String `tfsdk:"entity_field"`
}

// =================== [ Entity Type / Vital Metric ] ===================

type entityTypeVitalMetricModel struct {
	MetricName           types.String                          `tfsdk:"metric_name"`
	Search               types.String                          `tfsdk:"search"`
	MatchingEntityFields types.Map                             `tfsdk:"matching_entity_fields"`
	IsKey                types.Bool                            `tfsdk:"is_key"`
	Unit                 types.String                          `tfsdk:"unit"`
	AlertRule            []entityTypeVitalMetricAlertRuleModel `tfsdk:"alert_rule"`
}

func (m *entityTypeVitalMetricModel) getMatchingEntityFields(ctx context.Context) (fields map[string]string, diags diag.Diagnostics) {
	fields = make(map[string]string)
	if !m.MatchingEntityFields.IsNull() {
		diags = m.MatchingEntityFields.ElementsAs(ctx, &fields, false)
	}
	return
}

type entityTypeVitalMetricAlertRuleModel struct {
	CriticalThreshold types.Int64                                       `tfsdk:"critical_threshold"`
	WarningThreshold  types.Int64                                       `tfsdk:"warning_threshold"`
	CronSchedule      types.String                                      `tfsdk:"cron_schedule"`
	EntityFilter      []entityTypeVitalMetricAlertRuleEntityFilterModel `tfsdk:"entity_filter"`
	IsEnabled         types.Bool                                        `tfsdk:"is_enabled"`
	SuppressTime      types.String                                      `tfsdk:"suppress_time"`
}

type entityTypeVitalMetricAlertRuleEntityFilterModel struct {
	Field     types.String `tfsdk:"field"`
	FieldType types.String `tfsdk:"field_type"`
	Value     types.String `tfsdk:"value"`
}

type resourceEntityType struct {
	client models.ClientConfig
}

func NewResourceEntityType() resource.Resource {
	return &resourceEntityType{}
}

func (r *resourceEntityType) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureResourceClient(ctx, resourceNameEntityType, req, &r.client, resp)
}

func (r *resourceEntityType) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	configureResourceMetadata(req, resp, resourceNameEntityType)
}

func (r *resourceEntityType) dashboardDrilldownSchema() schema.SetNestedBlock {
	return schema.SetNestedBlock{
		MarkdownDescription: util.Dedent(`
			An array of dashboard drilldown objects.
			Each dashboard drilldown defines an internal or external resource you specify with a URL and parameters
			that map to one of an entity fields. The parameters are passed to the resource when you open the URL.
		`),
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				//NOTE:
				// As of terraform plugin framework 1.7.0 & terraform 1.7.5,
				// setting this object's fields to Computed + Optional triggers bugs
				// in terraform plugin framework, leading to weird and confusing plans,
				// where the planned value may be wrong/different from what's specified in the terraform config.
				// This is why we are currently setting the "dashboard_id", "dashboard_type" and "base_url" attributes as Required.
				"title": schema.StringAttribute{
					MarkdownDescription: "The name of the dashboard.",
					Required:            true,
				},
				"base_url": schema.StringAttribute{
					MarkdownDescription: util.Dedent(`
						An internal or external URL that points to the dashboard.
						This setting exists because for internal purposes, navigation suggestions are treated as dashboards.
						This setting is only required if is_splunk_dashboard is false.
					`),
					Required: true,
					// Optional: true,
					// Computed: true,
					// Default:  stringdefault.StaticString(""),
				},
				"dashboard_id": schema.StringAttribute{
					MarkdownDescription: "A unique identifier for the xml dashboard.",
					Required:            true,
					// Optional:            true,
					// Computed:            true,
					// Default:             stringdefault.StaticString(""),
				},
				"dashboard_type": schema.StringAttribute{
					MarkdownDescription: util.Dedent(`
						The type of dashboard being added.
						The following options are available:
						* xml_dashboard - a Splunk XML dashboard.
						* udf_dashboard - a Splunk UDF (Unified Dashboard Framework) dashboard.
						* navigation_link - a navigation URL. Should be used when base_url is specified.
					`),
					Required: true,
					// Optional: true,
					// Computed: true,
					// Default:  stringdefault.StaticString(entityTypeDefaultDashboardType),
					Validators: []validator.String{
						stringvalidator.OneOf(
							entityTypeDashboardTypeXMLDashboard,
							entityTypeDashboardTypeUDFDashboard,
							entityTypeDashboardTypeNavigation),
					},
				},
				"params": schema.MapAttribute{
					MarkdownDescription: "A set of parameters for the entity dashboard drilldown that provide a mapping of a URL parameter and its alias.",
					Optional:            true,
					Computed:            true,
					ElementType:         types.StringType,
					Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
				},
			},
		},
	}
}

func (r *resourceEntityType) dataDrilldownSchema() schema.SetNestedBlock {
	return schema.SetNestedBlock{
		MarkdownDescription: util.Dedent(`
			An array of data drilldown objects.
			Each data drilldown defines filters for raw data associated with entities that belong to the entity type.
		`),
		NestedObject: schema.NestedBlockObject{
			Blocks: map[string]schema.Block{
				"entity_field_filter": schema.SetNestedBlock{
					MarkdownDescription: util.Dedent(`
						Further filter down to the raw data associated with the entity
						based on a set of selected entity alias or informational fields.
					`),
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"data_field": schema.StringAttribute{
								MarkdownDescription: "Data field.",
								Required:            true,
							},
							"entity_field": schema.StringAttribute{
								MarkdownDescription: "Entity field.",
								Required:            true,
							},
						},
					},
					Validators: []validator.Set{
						setvalidator.SizeAtLeast(1),
					},
				},
			},
			Attributes: map[string]schema.Attribute{
				"title": schema.StringAttribute{
					MarkdownDescription: "The name of the drilldown.",
					Required:            true,
				},
				"type": schema.StringAttribute{
					MarkdownDescription: "Type of raw data to associate with. Must be either metrics or events.",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.OneOf("events", "metrics"),
					},
				},
				"static_filters": schema.MapAttribute{
					MarkdownDescription: "Filter down to a subset of raw data associated with the entity using static information like sourcetype.",
					Optional:            true,
					Computed:            true,
					ElementType:         types.StringType,
					Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
				},
			},
		},
	}
}

func (r *resourceEntityType) vitalMetricSchema() schema.SetNestedBlock {
	return schema.SetNestedBlock{
		MarkdownDescription: util.Dedent(`
			An set of vital metric objects. Vital metrics are statistical calculations based on
			SPL searches that represent the overall health of entities of that type.
		`),
		NestedObject: schema.NestedBlockObject{
			Blocks: map[string]schema.Block{
				"alert_rule": schema.SetNestedBlock{
					NestedObject: schema.NestedBlockObject{
						Blocks: map[string]schema.Block{
							"entity_filter": schema.SetNestedBlock{
								NestedObject: schema.NestedBlockObject{
									Attributes: map[string]schema.Attribute{
										"field": schema.StringAttribute{
											Required: true,
										},
										"field_type": schema.StringAttribute{
											MarkdownDescription: "Takes values alias or info specifying in which category of fields the field attribute is located.",
											Required:            true,
											Validators: []validator.String{
												stringvalidator.OneOf("alias", "info"),
											},
										},
										"value": schema.StringAttribute{
											Required: true,
										},
									},
								},
							},
						},
						Attributes: map[string]schema.Attribute{
							"critical_threshold": schema.Int64Attribute{
								Required: true,
							},
							"warning_threshold": schema.Int64Attribute{
								Required: true,
							},
							"cron_schedule": schema.StringAttribute{
								MarkdownDescription: "Frequency of the alert search",
								Required:            true,
							},
							"is_enabled": schema.BoolAttribute{
								MarkdownDescription: "Indicates if the alert rule is enabled.",
								Optional:            true,
								Computed:            true,
								Default:             booldefault.StaticBool(false),
							},
							"suppress_time": schema.StringAttribute{
								MarkdownDescription: "Frequency of the alert search",
								Optional:            true,
								Computed:            true,
								Default:             stringdefault.StaticString("0"),
							},
						},
					},
					Validators: []validator.Set{
						setvalidator.SizeAtMost(1),
					},
				},
			},
			Attributes: map[string]schema.Attribute{
				"metric_name": schema.StringAttribute{
					MarkdownDescription: util.Dedent(`
						The title of the vital metric. When creating vital metrics,
						it's a best practice to include the aggregation method and the name of the metric being calculated.
						For example, Average CPU usage.
					`),
					Required: true,
				},
				"search": schema.StringAttribute{
					MarkdownDescription: util.Dedent(`
						The search that computes the vital metric. The search must specify the following fields:
						- val for the value of the metric.
						- _time because the UI attempts to render changes over time. You can achieve this by adding span={time} to your search.
						- Fields as described in the split_by_fields configuration of this vital metric.
						For example, your search should be split by host,region if the split_by_fields configuration is [ "host", "region" ].
					`),
					Required: true,
				},
				"matching_entity_fields": schema.MapAttribute{
					MarkdownDescription: util.Dedent(`
						Specifies the aliases of an entity to use to match with the fields specified by the fields that the search configuration is split on.
						Make sure the value matches the split by fields in the actual search.
						For example:
							- search = "..... by InstanceId, region"
							- matching_entity_fields = {instance_id = "InstanceId", zone = "region"}.
					`),
					ElementType: types.StringType,
					Required:    true,
				},
				"is_key": schema.BoolAttribute{
					MarkdownDescription: util.Dedent(`
						Indicates if the vital metric specified is a key metric.
						A key metric calculates the distribution of entities associated with the entity type to indicate the overall health of the entity type.
						The key metric is rendered as a histogram in the Infrastructure Overview. Only one vital metric can have is_key set to true.
					`),
					// As of terraform 1.7.4 and terraform-plugin-framework 1.6.1, making this optional & computed triggers a bug leading to flip flopping of the value on every apply.
					// https://github.com/hashicorp/terraform-plugin-framework/issues/867
					// https://github.com/hashicorp/terraform-plugin-framework/issues/783
					Required: true,
				},
				"unit": schema.StringAttribute{
					MarkdownDescription: "The unit of the vital metric. For example, KB/s.",
					Optional:            true,
					Computed:            true,
					Default:             stringdefault.StaticString(""),
				},
			},
		},
	}
}

func (r *resourceEntityType) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: util.Dedent(`
			An entity_type defines how to classify a type of data source.
			For example, you can create a Linux, Windows, Unix/Linux add-on, VMware, or Kubernetes entity type.
			An entity type can include zero or more data drilldowns and zero or more dashboard drilldowns.
			You can use a single data drilldown or dashboard drilldown for multiple entity types.
		`),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the entity type.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "The name of the entity type.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the entity type.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
		},
		Blocks: map[string]schema.Block{
			"dashboard_drilldown": r.dashboardDrilldownSchema(),
			"data_drilldown":      r.dataDrilldownSchema(),
			"vital_metric":        r.vitalMetricSchema(),
			"timeouts":            timeouts.BlockAll(ctx),
		},
	}
}

// =================== [ Entity Type API / Builder] ===================

type entityTypeBuildWorkflow struct{}

var _ apibuildWorkflow[entityTypeModel] = &entityTypeBuildWorkflow{}

//lint:ignore U1000 used by apibuilder
func (w *entityTypeBuildWorkflow) buildSteps() []apibuildWorkflowStepFunc[entityTypeModel] {
	return []apibuildWorkflowStepFunc[entityTypeModel]{
		w.basics,
		w.dashboardDrilldowns,
		w.dataDrilldowns,
		w.vitalMetrics,
	}
}

func (w *entityTypeBuildWorkflow) basics(ctx context.Context, obj entityTypeModel) (map[string]any, diag.Diagnostics) {
	return map[string]any{
		"object_type": itsiResourceTypeEntityType,
		"sec_grp":     itsiDefaultSecurityGroup,
		"title":       obj.Title.ValueString(),
		"description": obj.Description.ValueString(),
	}, nil
}

func (w *entityTypeBuildWorkflow) dashboardDrilldowns(ctx context.Context, obj entityTypeModel) (map[string]any, diag.Diagnostics) {
	drilldowns := obj.DashboardDrilldown
	drilldownsAPI := make([]map[string]any, len(drilldowns))
	for i, drilldown := range drilldowns {
		params, _ := drilldown.getParams(ctx)

		dashboardDrilldownTitle := drilldown.Title.ValueString()
		dashboardType := drilldown.DashboardType.ValueString()
		var dashboardID, dashboardBaseURL string

		if util.NewSet(entityTypeDashboardTypeXMLDashboard, entityTypeDashboardTypeUDFDashboard).Contains(dashboardType) {
			dashboardID = drilldown.DashboardID.ValueString()
		} else {
			dashboardBaseURL = drilldown.BaseURL.ValueString()
			dashboardID = dashboardDrilldownTitle
		}

		bodyParams := []any{}
		for alias, param := range params {
			bodyParams = append(bodyParams, map[string]any{
				"alias": alias,
				"param": param,
			})
		}

		drilldownsAPI[i] = map[string]any{
			"title":          drilldown.Title.ValueString(),
			"base_url":       dashboardBaseURL,
			"dashboard_id":   dashboardID,
			"dashboard_type": dashboardType,
			"params": map[string]any{
				"static_params":   map[string]any{},
				"alias_param_map": bodyParams,
			},
		}
	}

	return map[string]any{
		"dashboard_drilldowns": drilldownsAPI,
	}, nil
}

func (w *entityTypeBuildWorkflow) dataDrilldowns(ctx context.Context, obj entityTypeModel) (res map[string]any, diags diag.Diagnostics) {
	drilldowns := obj.DataDrilldown

	drilldownsAPI := make([]map[string]any, len(drilldowns))
	for i, drilldown := range drilldowns {
		staticFilters, d := drilldown.getStaticFilters(ctx)
		diags = append(diags, d...)
		if diags.HasError() {
			return
		}
		entityFieldFilters := drilldown.EntityFieldFilter

		staticFiltersItems := []any{}
		for k, v := range staticFilters {
			staticFiltersItems = append(staticFiltersItems, map[string]any{
				"type":   "include",
				"field":  k,
				"values": []string{v},
			})
		}
		var staticFiltersAPI any
		if len(staticFiltersItems) == 1 {
			staticFiltersAPI = staticFiltersItems[0]
		} else {
			staticFiltersAPI = map[string]any{
				"type":    "and",
				"filters": staticFiltersItems,
			}
		}

		entityFieldFilterItems := make([]map[string]any, len(entityFieldFilters))
		for i, filter := range entityFieldFilters {
			entityFieldFilterItems[i] = map[string]any{
				"type":         "entity",
				"data_field":   filter.DataField.ValueString(),
				"entity_field": filter.EntityField.ValueString(),
			}
		}
		var entityFieldFiltersAPI any
		if len(entityFieldFilterItems) == 1 {
			entityFieldFiltersAPI = entityFieldFilterItems[0]
		} else {
			entityFieldFiltersAPI = map[string]any{
				"type":    "and",
				"filters": entityFieldFilterItems,
			}
		}

		drilldownsAPI[i] = map[string]any{
			"title":               drilldown.Title.ValueString(),
			"type":                drilldown.Type.ValueString(),
			"static_filter":       staticFiltersAPI,
			"entity_field_filter": entityFieldFiltersAPI,
		}

	}

	return map[string]any{
		"data_drilldowns": drilldownsAPI,
	}, nil
}

func (w *entityTypeBuildWorkflow) vitalMetrics(ctx context.Context, obj entityTypeModel) (res map[string]any, diags diag.Diagnostics) {
	vitalMetrics := obj.VitalMetric

	vitalMetricsAPI := make([]map[string]any, len(vitalMetrics))

	for i, vm := range vitalMetrics {
		vmAPI := make(map[string]any)
		metricName := vm.MetricName.ValueString()
		vmAPI["metric_name"] = metricName
		vmAPI["search"] = vm.Search.ValueString()
		vmAPI["is_key"] = vm.IsKey.ValueBool()
		vmAPI["unit"] = vm.Unit.ValueString()
		vmAPI["matching_entity_fields"] = []string{}
		vmAPI["split_by_fields"] = []string{}
		matchingEntityFields, d := vm.getMatchingEntityFields(ctx)
		diags = append(diags, d...)
		if diags.HasError() {
			return
		}

		for alias, splitByField := range matchingEntityFields {
			vmAPI["matching_entity_fields"] = append(vmAPI["matching_entity_fields"].([]string), alias)
			vmAPI["split_by_fields"] = append(vmAPI["split_by_fields"].([]string), splitByField)
		}

		for j, rule := range vm.AlertRule {
			if j > 0 {
				diags.AddError("Only one alert rule is supported", fmt.Sprintf("more than one alert rule is passed in metric %s", metricName))
				return
			}
			alertRuleAPI := map[string]any{}

			alertRuleAPI["suppress_time"] = rule.SuppressTime.ValueString()
			alertRuleAPI["cron_schedule"] = rule.CronSchedule.ValueString()
			alertRuleAPI["is_enabled"] = rule.IsEnabled.ValueBool()

			criticalThreshold := int(rule.CriticalThreshold.ValueInt64())
			warningThreshold := int(rule.WarningThreshold.ValueInt64())

			if warningThreshold < criticalThreshold {
				alertRuleAPI["critical_threshold"] = []string{strconv.Itoa(criticalThreshold), "+inf"}
				alertRuleAPI["warning_threshold"] = []string{strconv.Itoa(warningThreshold), strconv.Itoa(criticalThreshold)}
				alertRuleAPI["info_threshold"] = []string{"-inf", strconv.Itoa(warningThreshold)}
			} else {
				alertRuleAPI["critical_threshold"] = []string{"-inf", strconv.Itoa(criticalThreshold)}
				alertRuleAPI["warning_threshold"] = []string{strconv.Itoa(criticalThreshold), strconv.Itoa(warningThreshold)}
				alertRuleAPI["info_threshold"] = []string{strconv.Itoa(warningThreshold), "+inf"}
			}

			alertRuleEntityFiltersAPI := make([]map[string]string, len(rule.EntityFilter))
			for k, filter := range rule.EntityFilter {
				alertRuleEntityFiltersAPI[k] = map[string]string{
					"field":      filter.Field.ValueString(),
					"field_type": filter.FieldType.ValueString(),
					"value":      filter.Value.ValueString(),
				}
			}
			alertRuleAPI["entity_filter"] = alertRuleEntityFiltersAPI
			vmAPI["alert_rule"] = alertRuleAPI
		}

		vitalMetricsAPI[i] = vmAPI
	}

	return map[string]any{
		"vital_metrics": vitalMetricsAPI,
	}, diags
}

// =================== [ Entity Type API / Parser ] ===================

type entityTypeParseWorkflow struct{}

var _ apiparseWorkflow[entityTypeModel] = &entityTypeParseWorkflow{}

//lint:ignore U1000 used by apiparser
func (w *entityTypeParseWorkflow) parseSteps() []apiparseWorkflowStepFunc[entityTypeModel] {
	return []apiparseWorkflowStepFunc[entityTypeModel]{
		w.basics,
		w.dashboardDrilldowns,
		w.dataDrilldowns,
		w.vitalMetrics,
	}
}

func (w *entityTypeParseWorkflow) basics(ctx context.Context, fields map[string]any, res *entityTypeModel) (diags diag.Diagnostics) {
	stringMap, err := unpackMap[string](mapSubset(fields, []string{"title", "description"}))
	if err != nil {
		diags.AddError("Unable to populate entity type model", err.Error())
		return
	}
	res.Title = types.StringValue(stringMap["title"])
	res.Description = types.StringValue(stringMap["description"])
	return
}

func (w *entityTypeParseWorkflow) dashboardDrilldowns(ctx context.Context, fields map[string]any, res *entityTypeModel) (diags diag.Diagnostics) {
	var apiDrilldownList any
	var ok bool
	if apiDrilldownList, ok = fields["dashboard_drilldowns"]; !ok {
		return
	}

	dashboardDrilldowns := []entityTypeDashboardDrilldownModel{}

	apiDrilldowns, err := UnpackSlice[map[string]any](apiDrilldownList)
	if err != nil {
		diags.AddError("Unable to populate entity type model", err.Error())
		return
	}

	for _, apiDrilldown := range apiDrilldowns {
		title := types.StringValue(apiDrilldown["title"].(string))
		id := types.StringValue(apiDrilldown["id"].(string))
		baseURL := types.StringValue(apiDrilldown["base_url"].(string))

		apiParams := apiDrilldown["params"].(map[string]any)

		drilldownParams := map[string]string{}
		if aliasParamMap, ok := apiParams["alias_param_map"]; ok {
			aliasParamTuple, err := UnpackSlice[map[string]any](aliasParamMap)
			if err != nil {
				diags.AddError("Unable to populate entity type model", err.Error())
				return
			}
			for _, _aliasParamTuple := range aliasParamTuple {
				drilldownParams[_aliasParamTuple["alias"].(string)] = _aliasParamTuple["param"].(string)
			}
		}

		drilldownParamsMap, d := types.MapValueFrom(ctx, types.StringType, drilldownParams)
		if diags.Append(d...); diags.HasError() {
			return
		}

		dashboardDrilldowns = append(dashboardDrilldowns, entityTypeDashboardDrilldownModel{
			Title:         title,
			BaseURL:       baseURL,
			DashboardID:   id,
			DashboardType: types.StringValue(apiDrilldown["dashboard_type"].(string)),
			Params:        drilldownParamsMap,
		})
	}

	res.DashboardDrilldown = dashboardDrilldowns
	return
}

func (w *entityTypeParseWorkflow) dataDrilldowns(ctx context.Context, fields map[string]any, res *entityTypeModel) (diags diag.Diagnostics) {
	var apiDrilldownList any
	var ok bool
	if apiDrilldownList, ok = fields["data_drilldowns"]; !ok {
		return
	}

	dataDrilldowns := []entityTypeDataDrilldownModel{}
	apiDrilldowns, err := UnpackSlice[map[string]any](apiDrilldownList)
	if err != nil {
		diags.AddError("Unable to populate entity type model", err.Error())
		return
	}

	for _, apiDrilldown := range apiDrilldowns {
		title := types.StringValue(apiDrilldown["title"].(string))
		drilldownType := types.StringValue(apiDrilldown["type"].(string))

		apiStaticFilters := apiDrilldown["static_filter"].(map[string]any)
		if _, ok := apiStaticFilters["filters"]; !ok {
			apiStaticFilters["filters"] = []any{apiStaticFilters}
		}

		apiStaticFiltersList, err := UnpackSlice[map[string]any](apiStaticFilters["filters"])
		if err != nil {
			diags.AddError("Unable to populate entity type model", err.Error())
			return
		}

		staticFilters := map[string]string{}

		for _, filter := range apiStaticFiltersList {
			values, err := UnpackSlice[string](filter["values"])
			if err != nil {
				diags.AddError("Unable to populate entity type model", err.Error())
				return
			}
			if len(values) > 1 {
				diags.AddError("Unable to populate entity type model", "static filter values should be a single value")
			}
			staticFilters[filter["field"].(string)] = values[0]
		}
		staticFiltersMap, d := types.MapValueFrom(ctx, types.StringType, staticFilters)
		if diags.Append(d...); diags.HasError() {
			return
		}

		tfEntityFilters := []entityTypeDataDrilldownEntityFieldFilterModel{}
		if entityFieldFilter, ok := apiDrilldown["entity_field_filter"]; ok {
			_entityFieldFilter := entityFieldFilter.(map[string]any)
			var apiEntityFilters []map[string]any
			if filters, ok := _entityFieldFilter["filters"]; ok {
				apiEntityFilters, err = UnpackSlice[map[string]any](filters)
				if err != nil {
					diags.AddError("Unable to populate entity type model", err.Error())
					return
				}
			} else {
				apiEntityFilters = []map[string]any{_entityFieldFilter}
			}
			for _, apiEntityFilter := range apiEntityFilters {
				tfEntityFilters = append(tfEntityFilters, entityTypeDataDrilldownEntityFieldFilterModel{
					DataField:   types.StringValue(apiEntityFilter["data_field"].(string)),
					EntityField: types.StringValue(apiEntityFilter["entity_field"].(string)),
				})
			}
		}

		dataDrilldowns = append(dataDrilldowns, entityTypeDataDrilldownModel{
			Title:             title,
			Type:              drilldownType,
			StaticFilters:     staticFiltersMap,
			EntityFieldFilter: tfEntityFilters,
		})
	}

	res.DataDrilldown = dataDrilldowns
	return
}

func (w *entityTypeParseWorkflow) vitalMetrics(ctx context.Context, fields map[string]any, res *entityTypeModel) (diags diag.Diagnostics) {
	var apiVitalMetricsList any
	var ok bool
	if apiVitalMetricsList, ok = fields["vital_metrics"]; !ok {
		return
	}

	vitalMetrics := []entityTypeVitalMetricModel{}
	apiVitalMetrics, err := UnpackSlice[map[string]any](apiVitalMetricsList)
	if err != nil {
		diags.AddError("Unable to populate entity type model", err.Error())
		return
	}

	for _, apiVitalMetric := range apiVitalMetrics {

		tfVMName := types.StringValue(apiVitalMetric["metric_name"].(string))
		tfVMSearch := types.StringValue(apiVitalMetric["search"].(string))
		tfVMIsKey := types.BoolValue(util.Atob(apiVitalMetric["is_key"]))
		tfVMUnit := types.StringValue(apiVitalMetric["unit"].(string))

		matchingEntityFields := map[string]string{}

		apiMatchingEntityFields := apiVitalMetric["matching_entity_fields"].([]any)
		apiSplitByFields := apiVitalMetric["split_by_fields"].([]any)
		if len(apiMatchingEntityFields) != len(apiSplitByFields) {
			diags.AddError("Unable to populate entity type model", "matching_entity_fields and split_by_fields should be of the same length")
			return
		}

		for i, alias := range apiMatchingEntityFields {
			matchingEntityFields[alias.(string)] = apiSplitByFields[i].(string)
		}

		matchingEntityFieldsMap, d := types.MapValueFrom(ctx, types.StringType, matchingEntityFields)
		if diags.Append(d...); diags.HasError() {
			return
		}

		tfAlertRule := []entityTypeVitalMetricAlertRuleModel{}

		apiAlertRule, ok := apiVitalMetric["alert_rule"].(map[string]any)
		if ok && len(apiAlertRule) > 0 {

			apiAlertRuleEntityFilters := apiAlertRule["entity_filter"].([]any)
			tfAlertRuleEntityFilters := []entityTypeVitalMetricAlertRuleEntityFilterModel{}
			for _, apiAlertRuleEntityFilter := range apiAlertRuleEntityFilters {
				apiAlertRuleEntityFilterMap := apiAlertRuleEntityFilter.(map[string]any)
				tfAlertRuleEntityFilters = append(tfAlertRuleEntityFilters, entityTypeVitalMetricAlertRuleEntityFilterModel{
					Field:     types.StringValue(apiAlertRuleEntityFilterMap["field"].(string)),
					FieldType: types.StringValue(apiAlertRuleEntityFilterMap["field_type"].(string)),
					Value:     types.StringValue(apiAlertRuleEntityFilterMap["value"].(string)),
				})
			}

			criticalThresholdStr := apiAlertRule["critical_threshold"].([]any)
			warningThresholdStr := apiAlertRule["warning_threshold"].([]any)

			idx := 0
			if criticalThresholdStr[0].(string) == "-inf" {
				idx = 1
			}

			criticalThreshold, err := strconv.Atoi(criticalThresholdStr[idx].(string))
			if err != nil {
				diags.AddError("Unable to populate entity type model", err.Error())
				return
			}
			warningThreshold, err := strconv.Atoi(warningThresholdStr[idx].(string))
			if err != nil {
				diags.AddError("Unable to populate entity type model", err.Error())
				return
			}

			tfAlertRule = append(tfAlertRule, entityTypeVitalMetricAlertRuleModel{
				CriticalThreshold: types.Int64Value(int64(criticalThreshold)),
				WarningThreshold:  types.Int64Value(int64(warningThreshold)),
				CronSchedule:      types.StringValue(apiAlertRule["cron_schedule"].(string)),
				EntityFilter:      tfAlertRuleEntityFilters,
				IsEnabled:         types.BoolValue(apiAlertRule["is_enabled"].(bool)),
				SuppressTime:      types.StringValue(apiAlertRule["suppress_time"].(string)),
			})

		}

		vitalMetrics = append(vitalMetrics, entityTypeVitalMetricModel{
			MetricName:           tfVMName,
			Search:               tfVMSearch,
			IsKey:                tfVMIsKey,
			Unit:                 tfVMUnit,
			MatchingEntityFields: matchingEntityFieldsMap,
			AlertRule:            tfAlertRule,
		})
	}

	res.VitalMetric = vitalMetrics
	return
}

// =================== [ Entity Type Resource CRUD ] ===================

func (r *resourceEntityType) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state entityTypeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	timeouts := state.Timeouts
	readTimeout, diags := timeouts.Read(ctx, tftimeout.Read)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	base := entityTypeBase(r.client, state.ID.ValueString(), state.Title.ValueString())
	b, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read entity type", err.Error())
		return
	}
	if b == nil || b.RawJson == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state, diags = newAPIParser(b, new(entityTypeParseWorkflow)).parse(ctx, b)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	state.Timeouts = timeouts
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *resourceEntityType) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan entityTypeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, tftimeout.Create)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	base, diags := newAPIBuilder(r.client, new(entityTypeBuildWorkflow)).build(ctx, plan)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	base, err := base.Create(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create entity type", err.Error())
		return
	}

	plan.ID = types.StringValue(base.RESTKey)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *resourceEntityType) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var plan entityTypeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, tftimeout.Update)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	base, diags := newAPIBuilder(r.client, new(entityTypeBuildWorkflow)).build(ctx, plan)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	existing, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update entity type", err.Error())
		return
	}
	if existing == nil {
		resp.Diagnostics.AddError("Unable to update entity type", "entity type not found")
		return
	}
	if err := base.Update(ctx); err != nil {
		resp.Diagnostics.AddError("Unable to update entity type", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *resourceEntityType) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state entityTypeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	deleteTimeout, diags := state.Timeouts.Delete(ctx, tftimeout.Delete)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	base := entityTypeBase(r.client, state.ID.ValueString(), state.Title.ValueString())
	b, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete entity type", err.Error())
		return
	}
	if b == nil {
		return
	}

	resp.Diagnostics.Append(b.Delete(ctx)...)
}

func (r *resourceEntityType) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx, cancel := context.WithTimeout(ctx, tftimeout.Read)
	defer cancel()

	b := entityTypeBase(r.client, "", req.ID)
	b, err := b.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to find entity type model", err.Error())
		return
	}
	if b == nil {
		resp.Diagnostics.AddError("Entity type not found", fmt.Sprintf("Entity type '%s' not found", req.ID))
		return
	}

	state, diags := newAPIParser(b, new(entityTypeParseWorkflow)).parse(ctx, b)
	if resp.Diagnostics.Append(diags...); diags.HasError() {
		return
	}

	var timeouts timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &timeouts)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Timeouts = timeouts

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
