package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &resourceService{}
	_ resource.ResourceWithImportState = &resourceService{}
	_ resource.ResourceWithConfigure   = &resourceService{}
)

type resourceService struct {
	client models.ClientConfig
}

func NewResourceService() resource.Resource {
	return &resourceService{}
}

func (r *resourceService) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(models.ClientConfig)
	if !ok {
		tflog.Error(ctx, "Unable to prepare client")
		resp.Diagnostics.AddError("Unable to prepare client", "invalid provider data")
		return
	}
	r.client = client
}

type ServiceState struct {
	ID                                    types.String             `json:"_key" tfsdk:"id"`
	Title                                 types.String             `json:"title" tfsdk:"title"`
	Description                           types.String             `json:"description" tfsdk:"description"`
	Enabled                               types.Bool               `json:"enabled" tfsdk:"enabled"`
	IsHealthscoreCalculateByEntityEnabled types.Bool               `json:"is_healthscore_calculate_by_entity_enabled" tfsdk:"is_healthscore_calculate_by_entity_enabled"`
	SecurityGroup                         types.String             `json:"sec_grp" tfsdk:"security_group"`
	Tags                                  types.Set                `tfsdk:"tags"`
	ShkpiID                               types.String             `json:"shkpi_id" tfsdk:"shkpi_id"`
	KPIs                                  []*KpiState              `tfsdk:"kpi"`
	EntityRules                           []*EntityRuleState       `tfsdk:"entity_rules"`
	ServiceDependsOn                      []*ServiceDependsOnState `tfsdk:"service_depends_on"`
}

type KpiState struct {
	ID                  types.String `json:"_key" tfsdk:"id"`
	Title               types.String `json:"title" tfsdk:"title"`
	Description         types.String `json:"description" tfsdk:"description"`
	Type                types.String `json:"type" tfsdk:"type"`
	Urgency             types.Int64  `json:"urgency" tfsdk:"urgency"`
	BaseSearchID        types.String `json:"base_search_id" tfsdk:"base_search_id"`
	SearchType          types.String `json:"search_type" tfsdk:"search_type"`
	BaseSearchMetric    types.String `tfsdk:"base_search_metric"`
	ThresholdTemplateID types.String `json:"kpi_threshold_template_id" tfsdk:"threshold_template_id"`
	//CustomThresholds    []*CustomThresholdState `tfsdk:"custom_threshold"`
}

// ServiceDependsOn represents the schema for service dependencies within a service.
type ServiceDependsOnState struct {
	Service             types.String `json:"service" tfsdk:"service"`
	KPIs                types.Set    `tfsdk:"kpis"`
	OverloadedUrgencies types.Map    `tfsdk:"overload_urgencies"`
}

// // CustomThreshold represents the structure for custom threshold settings within a KPI.
// type CustomThresholdState struct {
// 	EntityThresholds    []*ThresholdSettingModelv2 `tfsdk:"entity_thresholds"`
// 	AggregateThresholds []*ThresholdSettingModelv2 `tfsdk:"aggregate_thresholds"`
// }

// EntityRule represents the schema for an entity rule within a service.
type EntityRuleState struct {
	Rule []*RuleState `tfsdk:"rule"`
}
type RuleState struct {
	Field     types.String `json:"field" tfsdk:"field"`
	FieldType types.String `json:"field_type" tfsdk:"field_type"`
	RuleType  types.String `json:"rule_type" tfsdk:"rule_type"`
	Value     types.String `json:"value" tfsdk:"value"`
}

func (r *resourceService) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

/*
 *  GENERATED_SEARCH_ATTRIBUTES:
 *  UI generates these searches via get_kpi_searches POST request
 *  with following mappings:
 *  'search' <= 'kpi_base_search',
 *  'kpi_base_search' <= 'kpi_base_search',
 *
 *  'search_aggregate' <= 'single_value_search',
 *  'search_entities' <= 'single_value_search',
 *
 *  'search_time_series' <= 'time_series_search',
 *  'search_time_series_aggregate' <= 'time_series_search',
 *
 *  'search_time_series_entities' <= 'entity_time_series_search,
 *  'search_time_compare' <= 'compare_search',
 *  'search_alert' <= 'alert_search,
 *  'search_alert_entities' (!) Didn't mapped in UI. Default ""
 *
 *  BUT in case all base search field are passed splunk generates it automatically after POST/PUT service
 *
 *  KPI BASE SEARCH is managed through terraform resource, so if kpi base searches' content is changed, splunk responsibility
 *  to update linked fields, there is no need to save linked values in the resource.
 *
 */

func (r *resourceService) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	//threshold_settings_blocks, threshold_settings_attributes := getKpiThresholdSettingsBlocksAttrs()
	resp.Schema = schema.Schema{
		Description: "Manages a Service within ITSI.",
		Blocks: map[string]schema.Block{
			"kpi": schema.SetNestedBlock{
				Description: "A set of rules within the rule group, which are combined using AND operator.",
				NestedObject: schema.NestedBlockObject{
					/*Blocks: map[string]schema.Block{
						"custom_threshold": schema.SetNestedBlock{
							//Optional: true,
							NestedObject: schema.NestedBlockObject{
								Blocks: map[string]schema.Block{
									"entity_thresholds": schema.SetNestedBlock{
										//Required: true,
										NestedObject: schema.NestedBlockObject{
											Attributes: threshold_settings_attributes,
											Blocks:     threshold_settings_blocks,
										},
									},
									"aggregate_thresholds": schema.SetNestedBlock{
										//Required: true,
										NestedObject: schema.NestedBlockObject{
											Attributes: threshold_settings_attributes,
											Blocks:     threshold_settings_blocks,
										},
									},
								},
							},
						},
					},*/
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Optional: true,
							Computed: true,
							Description: `id (splunk _key) is automatically generated sha1 string, from base_search_id & metric_id seed,
							concatenated with serviceId.`,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"title": schema.StringAttribute{
							Required:    true,
							Description: "Name of the kpi. Can be any unique value.",
						},
						"description": schema.StringAttribute{
							Optional:    true,
							Description: "User-defined description for the KPI. ",
						},
						"type": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("kpis_primary"),
							Description: "Could be service_health or kpis_primary.",
							Validators: []validator.String{
								stringvalidator.OneOf("kpis_primary", "service_health"),
							},
						},
						"urgency": schema.Int64Attribute{
							Optional: true,
							Computed: true,
							//Default:     int64default.StaticInt64(5),
							Description: "User-assigned importance value for this KPI.",
							Validators: []validator.Int64{
								int64validator.Between(0, 11),
							},
						},
						// BASE_SEARCH_KPI_ATTRIBUTES
						"base_search_id": schema.StringAttribute{
							Required: true,
						},
						"search_type": schema.StringAttribute{
							Optional: true,
							Computed: true,
							Default:  stringdefault.StaticString("shared_base"),
							Validators: []validator.String{
								stringvalidator.OneOf("shared_base"),
							},
						},
						"base_search_metric": schema.StringAttribute{
							Required: true,
						},
						"threshold_template_id": schema.StringAttribute{
							Optional: true,
						},
					},
				},
			},
			"entity_rules": schema.SetNestedBlock{
				Description: "A set of rules within the rule group, which are combined using OR operator.",
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"rule": schema.SetNestedBlock{
							Description: "A set of rules within the rule group, which are combined using AND operator.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"field": schema.StringAttribute{
										Required:    true,
										Description: "The field in the entity definition to compare values to evaluate this rule.",
									},
									"field_type": schema.StringAttribute{
										Required:    true,
										Description: "Takes values alias, info or title specifying in which category of fields the field attribute is located.",
										Validators: []validator.String{
											stringvalidator.OneOf("alias", "entity_type", "info", "title"),
										},
									},
									"rule_type": schema.StringAttribute{
										Required:    true,
										Description: "Takes values not or matches to indicate whether it's an inclusion or exclusion rule.",
										Validators: []validator.String{
											stringvalidator.OneOf("matches", "not"),
										},
									},
									"value": schema.StringAttribute{
										Required:    true,
										Description: "Values to evaluate in the rule. To specify multiple values, separate them with a comma. Values are not case sensitive.",
									},
								},
							},
						},
					},
				},
			},
			"service_depends_on": schema.SetNestedBlock{
				Description: "A set of service descriptions with KPIs in those services that this service depends on.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"service": schema.StringAttribute{
							Required:    true,
							Description: "_key value of service that this service depends on.",
						},
						"kpis": schema.SetAttribute{
							Required:    true,
							Description: "A set of _key ids for each KPI in service identified by serviceid, which this service will depend on.",
							ElementType: types.StringType,
						},
						"overload_urgencies": schema.MapAttribute{
							Optional:    true,
							Description: "A map of urgency overriddes for the KPIs this service is depending on.",
							ElementType: types.Int64Type,
						},
					},
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				Required:    true,
				Description: "Title of the service.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "User-defined description for the service.",
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Description: "Boolean value defining whether the service should be enabled.",
			},
			"is_healthscore_calculate_by_entity_enabled": schema.BoolAttribute{
				Optional:    true,
				Description: "Set the Service Health Score calculation to account for the severity levels of individual entities if at least one KPI is split by entity.",
			},
			"security_group": schema.StringAttribute{
				Optional:    true,
				Description: "The team the object belongs to.",
			},
			"tags": schema.SetAttribute{
				Optional:    true,
				Description: "The tags for the service.",
				ElementType: types.StringType,
			},
			"shkpi_id": schema.StringAttribute{
				Computed:    true,
				Description: "_key value for the Service Health Score KPI.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *resourceService) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ServiceState
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	base := serviceBase(r.client, state.ID.ValueString(), state.Title.ValueString())
	b, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read service", err.Error())
		return
	}
	if b == nil || b.RawJson == nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &ServiceState{})...)
		return
	}

	state, diags := serviceModelFromBase(ctx, b)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *resourceService) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ServiceState
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	base, diags := serviceStateToJson(ctx, r.client, plan)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	base, err := base.Create(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Service", err.Error())
		return
	}

	plan.ID = types.StringValue(base.RESTKey)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

}

func (r *resourceService) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ServiceState
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	base, diags := serviceStateToJson(ctx, r.client, plan)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	existing, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update Service", err.Error())
		return
	}
	if existing == nil {
		resp.Diagnostics.AddError("Unable to update Service", "service not found")
		return
	}
	if err := base.Update(ctx); err != nil {
		resp.Diagnostics.AddError("Unable to update service", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *resourceService) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ServiceState
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	base := serviceBase(r.client, state.ID.ValueString(), state.Title.ValueString())
	b, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete entity", err.Error())
		return
	}
	if b == nil {
		return
	}
	if err := b.Delete(ctx); err != nil {
		resp.Diagnostics.AddError("Unable to delete service", err.Error())
		return
	}
}

func (r *resourceService) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	b := serviceBase(r.client, "", req.ID)
	b, err := b.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to find service model", err.Error())
		return
	}
	if b == nil {
		resp.Diagnostics.AddError("Service not found", fmt.Sprintf("Service '%s' not found", req.ID))
		return
	}

	state, diags := serviceModelFromBase(ctx, b)
	if resp.Diagnostics.Append(diags...); diags.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func serviceBase(clientConfig models.ClientConfig, key string, title string) *models.Base {
	base := models.NewBase(clientConfig, key, title, "service")
	return base
}

func serviceModelFromBase(ctx context.Context, b *models.Base) (m ServiceState, diags diag.Diagnostics) {
	if b == nil || b.RawJson == nil {
		diags.AddError("Unable to populate service model", "base object is nil or empty.")
		return
	}

	interfaceMap, err := b.RawJson.ToInterfaceMap()
	if err != nil {
		diags.AddError("Unable to populate service model", err.Error())
		return
	}
	diags = append(diags, marshalBasicTypesByTag("json", interfaceMap, &m)...)

	tags := []interface{}{}
	if serviceTagsInterface, ok := interfaceMap["service_tags"].(map[string]interface{}); ok {
		if tagsInterface, ok := serviceTagsInterface["tags"]; ok && tagsInterface != nil {
			if _tags, ok := tagsInterface.([]interface{}); ok {
				//		tagsStr, err := unpackSlice[string](tags)
				tags = _tags
			}
		}
	}
	m.Tags, diags = types.SetValueFrom(ctx, types.StringType, tags)

	kpis, err := unpackSlice[map[string]interface{}](interfaceMap["kpis"])
	if err != nil {
		diags.AddError("Unable to unpack KPIs from service model", err.Error())
		return
	}

	m.KPIs = []*KpiState{}
	metricLookup := new(KPIBSMetricLookup)

	for _, kpi := range kpis {
		kpiTF := &KpiState{}
		diags = append(diags, marshalBasicTypesByTag("json", kpi, kpiTF)...)

		if kpiTF.Title.ValueString() == "ServiceHealthScore" {
			m.ShkpiID = kpiTF.ID
		} else if kpiTF.SearchType.ValueString() != "shared_base" {
			diags.AddWarning(
				fmt.Sprintf("[%s] Skipping %s KPI", m.Title.ValueString(), kpiTF.Title.ValueString()),
				fmt.Sprintf("%s KPIs is not supported", kpiTF.SearchType.ValueString()))
		} else {
			if kpiTF.BaseSearchMetric, err = metricLookup.lookupMetricTitleByID(ctx, b.Splunk, kpi["base_search_id"].(string), kpi["base_search_metric"].(string)); err != nil {
				diags.AddError("Unable to map KPIs BS metric ID to KPIs BS name", err.Error())
				continue
			}
			if val, ok := kpi["urgency"]; ok {
				if urgency, ok := val.(float64); ok {
					kpiTF.Urgency = types.Int64Value(int64(urgency))
				} else if urgency, err := strconv.Atoi(val.(string)); err == nil {
					kpiTF.Urgency = types.Int64Value(int64(urgency))
				}
			}

			/*kpiTF.CustomThresholds = []*CustomThresholdState{}
			if kpiTF.ThresholdTemplateID.ValueString() == "" {
				kpiTF.ThresholdTemplateID = types.StringNull()
				if kpi["adaptive_thresholds_is_enabled"].(bool) || kpi["time_variate_thresholds"].(bool) {
					diags.AddWarning("Unsupported thresholding",
						fmt.Sprintf("Custom threshold support only static non-time-variate thresholds: serviceId=%s kpiId=%s. Fallback to default", b.RESTKey, kpiTF.ID))
				} else {
					customThreshold := &CustomThresholdState{}

					unpackThresholds := func(key string) (thresholds *ThresholdSettingModelv2) {
						thresholds = &ThresholdSettingModelv2{}
						diags = append(diags, marshalBasicTypesByTag("json", kpi[key].(map[string]interface{}), thresholds)...)
						thresholds.ThresholdLevels = []*KpiThresholdLevelModel{}

						if itsiThresholds, ok := kpi[key].(map[string]interface{}); ok {
							levels, err := unpackSlice[map[string]interface{}](itsiThresholds["thresholdLevels"])
							if err != nil {
								diags.AddError("Unable to unpack custom KPIs levels from service model", err.Error())
								return
							}

							for _, level := range levels {
								levelTF := &KpiThresholdLevelModel{}
								diags = append(diags, marshalBasicTypesByTag("json", level, levelTF)...)

								thresholds.ThresholdLevels = append(thresholds.ThresholdLevels, levelTF)
							}
						}
						return
					}

					customThreshold.AggregateThresholds = []*ThresholdSettingModelv2{unpackThresholds("aggregate_thresholds")}
					customThreshold.EntityThresholds = []*ThresholdSettingModelv2{unpackThresholds("entity_thresholds")}
					kpiTF.CustomThresholds = append(kpiTF.CustomThresholds, customThreshold)
				}
			}*/

			m.KPIs = append(m.KPIs, kpiTF)
		}
	}
	m.EntityRules = []*EntityRuleState{}
	entityRules, err := unpackSlice[map[string]interface{}](interfaceMap["entity_rules"])
	if err != nil {
		diags.AddError("Unable to unpack entity rules from service model", err.Error())
		return
	}

	for _, entityRuleAndSet := range entityRules {
		ruleState := &EntityRuleState{}
		ruleSet := []*RuleState{}
		ruleItems, err := unpackSlice[map[string]interface{}](entityRuleAndSet["rule_items"])
		if err != nil {
			diags.AddError("Unable to unpack rule_item from service model", err.Error())
			return
		}
		for _, ruleItem := range ruleItems {
			ruleTF := &RuleState{}
			diags = append(diags, marshalBasicTypesByTag("json", ruleItem, ruleTF)...)
			ruleSet = append(ruleSet, ruleTF)
		}
		ruleState.Rule = ruleSet
		m.EntityRules = append(m.EntityRules, ruleState)
	}

	m.ServiceDependsOn = []*ServiceDependsOnState{}
	serviceDependsOn, err := unpackSlice[map[string]interface{}](interfaceMap["services_depends_on"])

	for _, serviceDepend := range serviceDependsOn {
		serviceDependsOn := &ServiceDependsOnState{}
		serviceDependsOn.Service = types.StringValue(serviceDepend["serviceid"].(string))
		kpiIds, err := unpackSlice[string](serviceDepend["kpis_depending_on"])
		if err != nil {
			diags.AddError("Unable to unpack kpis_depending_on from service model", err.Error())
			return
		}
		serviceDependsOn.KPIs, diags = types.SetValueFrom(ctx, types.StringType, kpiIds)
		if overloadedUrgencies, hasOverloadedUrgencies := serviceDepend["overloaded_urgencies"]; hasOverloadedUrgencies {
			serviceDependsOn.OverloadedUrgencies, diags = types.MapValueFrom(ctx, types.Int64Type, overloadedUrgencies.(map[string]interface{}))
		} else {
			serviceDependsOn.OverloadedUrgencies = types.MapNull(types.Int64Type)
		}
		m.ServiceDependsOn = append(m.ServiceDependsOn, serviceDependsOn)
	}

	m.ID = types.StringValue(b.RESTKey)

	return
}
func serviceStateToJson(ctx context.Context, clientConfig models.ClientConfig, m ServiceState) (config *models.Base, diags diag.Diagnostics) {
	title := m.Title.ValueString()
	body := map[string]interface{}{}

	body["title"] = title
	//body["description"] = m.Description.ValueString()

	config = serviceBase(clientConfig, m.ID.ValueString(), title)
	if err := config.PopulateRawJSON(ctx, body); err != nil {
		diags.AddError("Unable to populate base object", err.Error())
	}
	return
}

/* helper data structure to allow us specify metrics by title rather than ID */

type KPIBSMetricLookup struct {
	titleByKpiBsIDandMetricID map[string]string
}

func (ml *KPIBSMetricLookup) lookupKey(kpiBSID, metricID string) string {
	return fmt.Sprintf("%s:%s", kpiBSID, metricID)
}

func (ml *KPIBSMetricLookup) getKpiBSMetricTitleByID(ctx context.Context, cc models.ClientConfig, id string) (titleByID map[string]string, err error) {
	kpiBsData, err := getKpiBSData(ctx, cc, id)
	if err != nil {
		return nil, err
	}
	titleByID = make(map[string]string)

	for _, metric_ := range kpiBsData["metrics"].([]interface{}) {
		metric := metric_.(map[string]interface{})
		titleByID[metric["_key"].(string)] = metric["title"].(string)
	}
	return
}

func (ml *KPIBSMetricLookup) lookupMetricTitleByID(ctx context.Context, cc models.ClientConfig, kpiBSID, metricID string) (titleTF types.String, err error) {
	if ml.titleByKpiBsIDandMetricID == nil {
		ml.titleByKpiBsIDandMetricID = make(map[string]string)
	}
	title, ok := ml.titleByKpiBsIDandMetricID[ml.lookupKey(kpiBSID, metricID)]
	titleTF = types.StringValue(title)

	if ok {
		return
	}

	metricTitleByID, err := ml.getKpiBSMetricTitleByID(ctx, cc, kpiBSID)
	if err != nil {
		return
	}

	for metricID, metricTitle := range metricTitleByID {
		ml.titleByKpiBsIDandMetricID[ml.lookupKey(kpiBSID, metricID)] = metricTitle
	}

	if title, ok = ml.titleByKpiBsIDandMetricID[ml.lookupKey(kpiBSID, metricID)]; !ok {
		err = fmt.Errorf("metric %s not found in KPI Base search %s", metricID, kpiBSID)
	}
	titleTF = types.StringValue(title)

	return
}

func getKpiBSData(ctx context.Context, cc models.ClientConfig, id string) (map[string]interface{}, error) {
	kpiSearchBase, err := kpiBaseSearchBase(cc, id, "").Find(ctx)
	if err != nil {
		return nil, err
	}

	if kpiSearchBase == nil {
		return nil, fmt.Errorf("KPI Base search %s not found", id)
	}

	return kpiSearchBase.RawJson.ToInterfaceMap()
}
