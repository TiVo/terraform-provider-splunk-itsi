package provider

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"maps"
	"time"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/util"
)

const (
	itsiResourceTypeService = "service"
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
	configureResourceClient(ctx, resourceNameService, req, &r.client, resp)
}

type ServiceState struct {
	ID                                    types.String            `json:"_key" tfsdk:"id"`
	Title                                 types.String            `json:"title" tfsdk:"title"`
	Description                           types.String            `json:"description" tfsdk:"description"`
	Enabled                               types.Bool              `json:"enabled" tfsdk:"enabled"`
	IsHealthscoreCalculateByEntityEnabled types.Bool              `json:"is_healthscore_calculate_by_entity_enabled" tfsdk:"is_healthscore_calculate_by_entity_enabled"`
	SecurityGroup                         types.String            `json:"sec_grp" tfsdk:"security_group"`
	Tags                                  types.Set               `tfsdk:"tags"`
	ShkpiID                               types.String            `json:"shkpi_id" tfsdk:"shkpi_id"`
	KPIs                                  []KpiState              `tfsdk:"kpi"`
	EntityRules                           []EntityRuleState       `tfsdk:"entity_rules"`
	ServiceDependsOn                      []ServiceDependsOnState `tfsdk:"service_depends_on"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (s ServiceState) objectype() string {
	return itsiResourceTypeService
}

func (s ServiceState) title() string {
	return s.Title.ValueString()
}

type KpiState struct {
	ID                  types.String     `json:"_key" tfsdk:"id"`
	Title               types.String     `json:"title" tfsdk:"title"`
	Description         types.String     `json:"description" tfsdk:"description"`
	Type                types.String     `json:"type" tfsdk:"type"`
	Urgency             types.Int64      `json:"urgency" tfsdk:"urgency"`
	BaseSearchID        types.String     `json:"base_search_id" tfsdk:"base_search_id"`
	SearchType          types.String     `json:"search_type" tfsdk:"search_type"`
	BaseSearchMetric    types.String     `tfsdk:"base_search_metric"`
	ThresholdTemplateID types.String     `json:"kpi_threshold_template_id" tfsdk:"threshold_template_id"`
	MLThresholding      []MLThresholding `tfsdk:"ml_thresholding"`
}

func (ks KpiState) internalKey() string {
	baseSearchId := ks.BaseSearchID.ValueString()
	baseSearchMetricId := ks.BaseSearchMetric.ValueString()

	if baseSearchId == "" || baseSearchMetricId == "" {
		// Failed to identify key, do not modify this plan
		return ""
	}

	hash := sha1.New()
	hash.Write([]byte(baseSearchId + "_" + baseSearchMetricId))
	return hex.EncodeToString(hash.Sum(nil))
}

type MLThresholding struct {
	Direction      types.String      `tfsdk:"direction"`
	TrainingWindow types.String      `tfsdk:"training_window"`
	StartDate      timetypes.RFC3339 `tfsdk:"start_date"`
}

// ServiceDependsOn represents the schema for service dependencies within a service.
type ServiceDependsOnState struct {
	Service             types.String `json:"service" tfsdk:"service"`
	KPIs                types.Set    `tfsdk:"kpis"`
	OverloadedUrgencies types.Map    `tfsdk:"overloaded_urgencies"`
}

// EntityRule represents the schema for an entity rule within a service.
type EntityRuleState struct {
	Rule []RuleState `tfsdk:"rule"`
}
type RuleState struct {
	Field     types.String `json:"field" tfsdk:"field"`
	FieldType types.String `json:"field_type" tfsdk:"field_type"`
	RuleType  types.String `json:"rule_type" tfsdk:"rule_type"`
	Value     types.String `json:"value" tfsdk:"value"`
}

func (r *resourceService) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	configureResourceMetadata(req, resp, resourceNameService)
}

func (r *resourceService) blockMLThresholding(_ context.Context) schema.Block {
	return schema.ListNestedBlock{
		Description: "Configuration for AI-driven KPI Analysis",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"direction": schema.StringAttribute{
					Description: "Determines if the KPI should stay above a certain level, below a certain level, or constrained to a specific range. Takes values 'both', 'lower' or 'upper'.",
					Required:    true,
					Validators: []validator.String{
						stringvalidator.OneOf("both", "lower", "upper"),
					},
				},
				"training_window": schema.StringAttribute{
					Description: "Time window over which the thresholding recommendation should run. Same window will be used as the training window for adaptive thresholding. Takes values '-7d', '-14d', '-30d', '-60d'.",
					Required:    true,
					Validators: []validator.String{
						stringvalidator.OneOf("-7d", "-14d", "-30d", "-60d"),
					},
				},
				"start_date": schema.StringAttribute{
					Required:   true,
					CustomType: timetypes.RFC3339Type{},
					Description: "Defines the starting date and time from which the ML-Assisted Thresholding algorithm would analyze the historical KPI data. Must be a timestamp in " +
						"[RFC3339](https://datatracker.ietf.org/doc/html/rfc3339#section-5.8) format " +
						"(see [RFC3339 time string](https://tools.ietf.org/html/rfc3339#section-5.8) e.g., " +
						"`YYYY-MM-DDTHH:MM:SSZ`).",
				},
			},
		},
		Validators: []validator.List{
			listvalidator.SizeAtMost(1),
			listvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("threshold_template_id")),
		},
	}
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
	resp.Schema = schema.Schema{
		Description: "Manages a Service within ITSI.",
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
			"kpi": schema.ListNestedBlock{
				Description: "A set of KPI descriptions for this service.",
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"ml_thresholding": r.blockMLThresholding(ctx),
					},
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
							Description: `id (splunk _key) is automatically generated sha1 string, from base_search_id & metric_id seed,
							concatenated with serviceId.`,
						},
						"title": schema.StringAttribute{
							Required:    true,
							Description: "Name of the kpi. Can be any unique value.",
						},
						"description": schema.StringAttribute{
							Description: "User-defined description for the KPI. ",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(""),
						},
						"type": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("kpis_primary"),
							Description: "Could be kpis_primary.",
							Validators: []validator.String{
								stringvalidator.OneOf("kpis_primary"),
							},
						},
						"urgency": schema.Int64Attribute{
							Optional: true,
							Computed: true,
							/**
							 * For the case of the import of configurations, this method overrides the defined urgency level, despite
							 * the specified config value. This behavior is not observed during regular updates, where the specified
							 * config urgency levels are respected.
							 *
							 * Investigation reveals that the issue may be related to the method not recognizing integer values specified
							 * by path in the configuration, as seen in {@link https://github.com/hashicorp/terraform-plugin-framework/blob/main/internal/fwschemadata/data_default.go#L83}.
							 * However, the files generated post-import and the state structures resulting from import/read calls do contain
							 * the correct urgency values. Removing the default setting results in a clean plan. Subsequently,
							 * the default logic has been moved to the plan modifier to address this issue.
							 */
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
							Computed: true,
							Validators: []validator.String{
								stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("ml_thresholding")),
							},
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
						"overloaded_urgencies": schema.MapAttribute{
							Optional:    true,
							Computed:    true,
							Description: "A map of urgency overriddes for the KPIs this service is depending on.",
							ElementType: types.Int64Type,
							Default:     mapdefault.StaticValue(types.MapNull(types.Int64Type)),
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
				Computed:    true,
				Description: "User-defined description for the service.",
				Default:     stringdefault.StaticString(""),
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Boolean value defining whether the service should be enabled.",
			},
			"is_healthscore_calculate_by_entity_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Set the Service Health Score calculation to account for the severity levels of individual entities if at least one KPI is split by entity.",
			},
			"security_group": schema.StringAttribute{
				Optional:    true,
				Description: "The team the object belongs to.",
				Computed:    true,
				Default:     stringdefault.StaticString(itsiDefaultSecurityGroup),
			},
			"tags": schema.SetAttribute{
				Optional:    true,
				Description: "The tags for the service.",
				ElementType: types.StringType,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetNull(types.StringType)),
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

type KpiMapFields struct {
	ID                  types.String
	Description         types.String
	ThresholdTemplateID types.String
	Urgency             types.Int64
	MLThresholding      []MLThresholding
}

const (
	SERVICE_ENABLED_DEFAULT                  = true
	SERVICE_IS_HEALTHSCORE_BY_ENTITY_ENABLED = true
	DEFAULT_URGENCY                          = 5
)

type tfRequest struct {
	Config tfsdk.Config
	State  tfsdk.State
	Plan   tfsdk.Plan
}

type tfResponse struct {
	// Plan is the planned new state for the resource.
	Plan        *tfsdk.Plan
	Diagnostics *diag.Diagnostics
}

func (r *resourceService) remapAttributes(ctx context.Context, req tfRequest, resp *tfResponse) {
	var state, plan, config ServiceState
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if config.Description.IsNull() && plan.Description.IsUnknown() {
		plan.Description = types.StringNull()
	}

	kpiOldKeys := map[string]*KpiMapFields{}
	for _, kpi := range state.KPIs {

		// kpiid must be retained to prevent loss of historical data. Historical KPI data is considered valid
		// as long as base search & metric stay the same
		internalID := kpi.internalKey()
		if internalID == "" {
			resp.Diagnostics.AddError("KPI state missed required fields",
				fmt.Sprintf("no base search data specified, smt went wrong: %v", kpi))
		}

		kpiOldKeys[internalID] = &KpiMapFields{ID: kpi.ID}
	}
	// redefine urgency in case they specified in config
	for _, kpi := range config.KPIs {
		internalID := kpi.internalKey()
		if k, ok := kpiOldKeys[internalID]; internalID != "" && ok {
			k.Urgency = kpi.Urgency
			if kpi.Urgency.IsNull() {
				k.Urgency = types.Int64Value(DEFAULT_URGENCY)
			}
			k.Description = kpi.Description
			k.ThresholdTemplateID = kpi.ThresholdTemplateID
			k.MLThresholding = kpi.MLThresholding
		}

	}

	tfKpis := []KpiState{}
	for _, kpi := range plan.KPIs {
		internalID := kpi.internalKey()

		// map kpis in case kpi hash was successfull on get
		if existingKpi, ok := kpiOldKeys[internalID]; internalID != "" && ok {
			kpi.ID = existingKpi.ID
			kpi.Urgency = existingKpi.Urgency
			kpi.MLThresholding = existingKpi.MLThresholding

			if kpi.Description.IsUnknown() {
				kpi.Description = existingKpi.Description
			}
			if kpi.ThresholdTemplateID.IsUnknown() {
				kpi.ThresholdTemplateID = existingKpi.ThresholdTemplateID
			}

		}

		tfKpis = append(tfKpis, kpi)
	}
	plan.KPIs = tfKpis
	resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
}

func (r *resourceService) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}
	if !req.Config.Raw.IsFullyKnown() {
		return
	}

	tfReq := tfRequest{
		Config: req.Config,
		Plan:   req.Plan,
		State:  req.State,
	}
	tfResp := &tfResponse{
		Diagnostics: &resp.Diagnostics,
		Plan:        &resp.Plan,
	}

	r.remapAttributes(ctx, tfReq, tfResp)

	tflog.Trace(ctx, "Finished modifying plan for service resource")
}

func (r *resourceService) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ServiceState
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	timeouts := state.Timeouts
	readTimeout, diags := timeouts.Read(ctx, tftimeout.Read)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	b, err := ServiceBase(r.client, state.ID.ValueString(), state.Title.ValueString()).Read(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read service", err.Error())
		return
	}
	if b == nil || b.RawJson == nil {
		resp.State.Raw = tftypes.Value{}
		return
	}

	state, diags = newAPIParser(b, newServiceParseWorkflow(r.client)).parse(ctx, b)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	state.Timeouts = timeouts
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *resourceService) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ServiceState
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	timeouts := plan.Timeouts
	createTimeout, diags := timeouts.Create(ctx, tftimeout.Create)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	base, diags := newAPIBuilder(r.client, newServiceBuildWorkflow(r.client)).build(ctx, plan)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	base, err := base.Create(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Service", err.Error())
		return
	}

	plan.ID = types.StringValue(base.RESTKey)

	base, err = base.Read(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update Service", err.Error())
		return
	}

	state, diags := newAPIParser(base, newServiceParseWorkflow(r.client)).parse(ctx, base)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	state.Timeouts = timeouts
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

func (r *resourceService) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ServiceState
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	tfReq := tfRequest{
		Config: req.Config,
		Plan:   req.Plan,
		State:  req.State,
	}
	tfResp := &tfResponse{
		Diagnostics: &resp.Diagnostics,
		Plan:        &req.Plan,
	}

	r.remapAttributes(ctx, tfReq, tfResp)

	updateTimeout, diags := plan.Timeouts.Update(ctx, tftimeout.Update)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	base, diags := newAPIBuilder(r.client, newServiceBuildWorkflow(r.client)).build(ctx, plan)
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
	diags = base.UpdateAsync(ctx)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	base, err = base.Read(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update Service", err.Error())
		return
	}

	state, diags := newAPIParser(base, newServiceParseWorkflow(r.client)).parse(ctx, base)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	state.Timeouts = plan.Timeouts

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *resourceService) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ServiceState
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	deleteTimeout, diags := state.Timeouts.Delete(ctx, tftimeout.Delete)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	base := ServiceBase(r.client, state.ID.ValueString(), state.Title.ValueString())
	b, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete entity", err.Error())
		return
	}
	if b == nil {
		return
	}
	resp.Diagnostics.Append(b.Delete(ctx)...)
}

func (r *resourceService) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx, cancel := context.WithTimeout(ctx, tftimeout.Read)
	defer cancel()

	b, err := ServiceBase(r.client, "", req.ID).Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to find service model", err.Error())
		return
	}
	if b == nil {
		resp.Diagnostics.AddError("Service not found", fmt.Sprintf("Service '%s' not found", req.ID))
		return
	}

	state, diags := newAPIParser(b, newServiceParseWorkflow(r.client)).parse(ctx, b)
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

func ServiceBase(clientConfig models.ClientConfig, key string, title string) *models.ItsiObj {
	base := models.NewItsiObj(clientConfig, key, title, "service")
	return base
}

// [ Helper data structures for managing thresholding configurations ]

type thldValueLookupKeyString string

func thldValueLookupKey(policyName string, severityValue int64, dynamicParam float64) thldValueLookupKeyString {
	return thldValueLookupKeyString(fmt.Sprintf("%s::%d::%v", policyName, severityValue, dynamicParam))
}

// data structure to facilitate storing/looking up threshold values by <policy,severity,dynamicParam>
type thldValueLookup map[thldValueLookupKeyString]float64

func (l thldValueLookup) set(policyName string, severityValue int64, dynamicParam, thldValue float64) {
	l[thldValueLookupKey(policyName, severityValue, dynamicParam)] = thldValue
}

func (l thldValueLookup) get(policyName string, severityValue int64, dynamicParam float64) (thldValue float64, ok bool) {
	thldValue, ok = l[thldValueLookupKey(policyName, severityValue, dynamicParam)]
	return
}

//data structures to facilitate storing KPI thresholding configuration

type thldConfKey struct {
	kpiID     string
	thldTplID string
}
type thldConfValue map[string]any
type thldConfigState map[thldConfKey]thldConfValue

func (tcs thldConfigState) set(k thldConfKey, field string, v any) {
	if k.kpiID == "" {
		panic("thldConfigState: kpiID must not be empty")
	}
	if field == "" {
		panic("thldConfigState: field must not be empty")
	}
	if v == nil {
		return
	}

	if tcs[k] == nil {
		tcs[k] = make(thldConfValue)
	}
	tcs[k][field] = v
}

func (tcs thldConfigState) get(k thldConfKey) thldConfValue {
	if v, ok := tcs[k]; ok {
		return v
	} else {
		return make(thldConfValue)
	}
}

func (tcs thldConfigState) thldValueLookup(k thldConfKey) (l thldValueLookup) {
	kpi := tcs.get(k)
	timeVariateThresholdSpecification, ok := kpi["time_variate_thresholds_specification"].(map[string]any)
	if !ok {
		return nil
	}

	l = make(thldValueLookup)
	for policyName, policy := range timeVariateThresholdSpecification["policies"].(map[string]any) {
		_policy := policy.(map[string]any)
		if aggregate_thresholds, ok := _policy["aggregate_thresholds"].(map[string]any); ok {
			for _, threshold_level := range aggregate_thresholds["thresholdLevels"].([]any) {
				_threshold_level := threshold_level.(map[string]any)
				severityValue := int64(_threshold_level["severityValue"].(float64))
				dynamicParam := _threshold_level["dynamicParam"].(float64)
				thresholdValue := _threshold_level["thresholdValue"].(float64)
				l.set(policyName, severityValue, dynamicParam, thresholdValue)
			}
		}
	}
	return
}

// [ Service Parse Workflow ]___________________________________________________

type serviceParseWorkflow struct {
	clientConfig models.ClientConfig
}

var _ apiparseWorkflow[ServiceState] = &serviceParseWorkflow{}

func newServiceParseWorkflow(c models.ClientConfig) *serviceParseWorkflow {
	return &serviceParseWorkflow{c}
}

//lint:ignore U1000 used by apiparser
func (w *serviceParseWorkflow) parseSteps() []apiparseWorkflowStepFunc[ServiceState] {
	return []apiparseWorkflowStepFunc[ServiceState]{
		w.basics,
		w.tags,
		w.kpis,
		w.entityRules,
		w.serviceDependsOn,
	}
}

func (w *serviceParseWorkflow) basics(ctx context.Context, fields map[string]any, res *ServiceState) (diags diag.Diagnostics) {
	diags.Append(unmarshalBasicTypesByTag("json", fields, res)...)
	return
}

func (w *serviceParseWorkflow) tags(ctx context.Context, fields map[string]any, res *ServiceState) (diags diag.Diagnostics) {
	tags := []any{}
	if serviceTagsInterface, ok := fields["service_tags"].(map[string]any); ok {
		if tagsInterface, ok := serviceTagsInterface["tags"]; ok && tagsInterface != nil {
			if _tags, ok := tagsInterface.([]any); ok {
				tags = _tags
			}
		}
	}
	if len(tags) == 0 {
		res.Tags = types.SetNull(types.StringType)
	} else {
		res.Tags, diags = types.SetValueFrom(ctx, types.StringType, tags)
	}
	return
}

func (w *serviceParseWorkflow) kpis(ctx context.Context, fields map[string]any, res *ServiceState) (diags diag.Diagnostics) {
	kpis, err := UnpackSlice[map[string]any](fields["kpis"])
	if err != nil {
		diags.AddError("Unable to unpack KPIs from service model", err.Error())
		return
	}

	res.KPIs = []KpiState{}
	metricLookup := new(KPIBSMetricLookup)

	for _, kpi := range kpis {
		kpiTF := KpiState{}
		diags.Append(unmarshalBasicTypesByTag("json", kpi, &kpiTF)...)

		if kpiTF.Title.ValueString() == "ServiceHealthScore" {
			res.ShkpiID = kpiTF.ID
		} else if kpiTF.SearchType.ValueString() != "shared_base" {
			diags.AddWarning(
				fmt.Sprintf("[%s] Skipping %s KPI", res.Title.ValueString(), kpiTF.Title.ValueString()),
				fmt.Sprintf("%s KPIs is not supported", kpiTF.SearchType.ValueString()))
		} else {
			if kpiTF.BaseSearchMetric, err = metricLookup.lookupMetricTitleByID(ctx, w.clientConfig, kpi["base_search_id"].(string), kpi["base_search_metric"].(string)); err != nil {
				diags.AddError("Unable to map KPIs BS metric ID to KPIs BS name", err.Error())
				continue
			}
			if val, ok := kpi["urgency"]; ok {
				if urgency, err := util.Atoi(val); err == nil {
					kpiTF.Urgency = types.Int64Value(int64(urgency))
				} else {
					diags.AddError("Unable to parse urgency from service model", err.Error())
					continue
				}
			}

			if util.Atob(kpi["is_recommended_time_policies"]) {
				kpiTF.MLThresholding = []MLThresholding{{
					Direction:      types.StringValue(kpi["threshold_direction"].(string)),
					TrainingWindow: types.StringValue(kpi["recommendation_training_window"].(string)),
					StartDate:      timetypes.NewRFC3339TimeValue(time.Unix(int64(kpi["recommendation_start_date"].(float64)), 0).UTC()),
				}}
				kpiTF.ThresholdTemplateID = types.StringNull()
			} else {
				kpiTF.MLThresholding = []MLThresholding{}
			}

			res.KPIs = append(res.KPIs, kpiTF)
		}
	}
	return
}

func (w *serviceParseWorkflow) entityRules(ctx context.Context, fields map[string]any, res *ServiceState) (diags diag.Diagnostics) {
	res.EntityRules = []EntityRuleState{}
	entityRules, err := UnpackSlice[map[string]any](fields["entity_rules"])
	if err != nil {
		diags.AddError("Unable to unpack entity rules from service model", err.Error())
		return
	}

	for _, entityRuleAndSet := range entityRules {
		ruleState := EntityRuleState{}
		ruleSet := []RuleState{}
		ruleItems, err := UnpackSlice[map[string]any](entityRuleAndSet["rule_items"])
		if err != nil {
			diags.AddError("Unable to unpack rule_item from service model", err.Error())
			return
		}
		for _, ruleItem := range ruleItems {
			ruleTF := RuleState{}
			diags.Append(unmarshalBasicTypesByTag("json", ruleItem, &ruleTF)...)
			ruleSet = append(ruleSet, ruleTF)
		}
		ruleState.Rule = ruleSet
		res.EntityRules = append(res.EntityRules, ruleState)
	}

	return
}

func (w *serviceParseWorkflow) serviceDependsOn(ctx context.Context, fields map[string]any, res *ServiceState) (diags diag.Diagnostics) {
	res.ServiceDependsOn = []ServiceDependsOnState{}
	serviceDependsOn, err := UnpackSlice[map[string]any](fields["services_depends_on"])
	if fields["services_depends_on"] != nil && err != nil {
		diags.AddError("Unable to unpack services_depends_on from service model", err.Error())
		return
	}
	for _, serviceDepend := range serviceDependsOn {
		serviceDependsOn := ServiceDependsOnState{}
		serviceDependsOn.Service = types.StringValue(serviceDepend["serviceid"].(string))
		kpiIds, err := UnpackSlice[string](serviceDepend["kpis_depending_on"])
		if err != nil {
			diags.AddError("Unable to unpack kpis_depending_on from service model", err.Error())
			return
		}
		serviceDependsOn.KPIs, diags = types.SetValueFrom(ctx, types.StringType, kpiIds)
		if overloadedUrgencies, hasOverloadedUrgencies := serviceDepend["overloaded_urgencies"]; hasOverloadedUrgencies {
			serviceDependsOn.OverloadedUrgencies, diags = types.MapValueFrom(ctx, types.Int64Type, overloadedUrgencies.(map[string]any))
		} else {
			serviceDependsOn.OverloadedUrgencies = types.MapNull(types.Int64Type)
		}
		res.ServiceDependsOn = append(res.ServiceDependsOn, serviceDependsOn)
	}
	return
}

// [ Service Build Workflow ]___________________________________________________

type serviceBuildWorkflow struct {
	clientConfig models.ClientConfig
	tcs          thldConfigState
}

var _ apibuildWorkflow[ServiceState] = &serviceBuildWorkflow{}

func newServiceBuildWorkflow(c models.ClientConfig) *serviceBuildWorkflow {
	return &serviceBuildWorkflow{c, nil}
}

//lint:ignore U1000 used by apibuilder
func (w *serviceBuildWorkflow) buildSteps() []apibuildWorkflowStepFunc[ServiceState] {
	return []apibuildWorkflowStepFunc[ServiceState]{
		w.basics,
		w.populateThresholdValueCache,
		w.kpis,
		w.entityRules,
		w.serviceDependsOn,
		w.tags,
	}
}

func (w *serviceBuildWorkflow) basics(ctx context.Context, obj ServiceState) (body map[string]any, diags diag.Diagnostics) {
	return map[string]any{
		"object_type": "service",
		"title":       obj.Title.ValueString(),
		"description": obj.Description.ValueString(),
		"is_healthscore_calculate_by_entity_enabled": util.Btoi(obj.IsHealthscoreCalculateByEntityEnabled.ValueBool()),
		"enabled": util.Btoi(obj.Enabled.ValueBool()),
		"sec_grp": obj.SecurityGroup.ValueString(),
	}, nil
}

func (w *serviceBuildWorkflow) thresholdingConfigFields() []string {
	// list of KPI-level thresholding configuration fields that will :
	// * be populated according to the threshold template, if `kpi_threshold_template_id`` is provided in the `kpi` block,
	// * retained if the threshold template id is not provided (to allow for custom / AI-recommended thresholds)
	//
	return []string{
		"adaptive_thresholding_training_window",
		"adaptive_thresholds_is_enabled",
		"aggregate_outlier_detection_enabled",
		"aggregate_thresholds",
		"entity_thresholds",
		"outlier_detection_algo",
		"outlier_detection_sensitivity",
		"threshold_recommendations",
		"time_variate_thresholds_specification",
		"time_variate_thresholds",
	}
}

func (w *serviceBuildWorkflow) populateThresholdValueCache(ctx context.Context, obj ServiceState) (body map[string]any, diags diag.Diagnostics) {
	if obj.ID.ValueString() == "" {
		//we are creating a new service, there's no pre-existing state
		return
	}

	b, err := ServiceBase(w.clientConfig, obj.ID.ValueString(), obj.Title.ValueString()).Find(ctx)
	if err != nil {
		diags.AddError("Failed to find the service object", err.Error())
		return
	}

	svc, err := b.RawJson.ToInterfaceMap()
	if err != nil {
		diags.AddError("Failed to parse the service object", err.Error())
		return
	}

	if _, ok := svc["kpis"]; !ok {
		return
	}
	kpis, err := UnpackSlice[map[string]any](svc["kpis"])
	if err != nil {
		diags.AddError("Failed to parse service KPIs", err.Error())
	}

	w.tcs = make(thldConfigState, len(kpis))
	for _, kpi := range kpis {
		var kpiID, thldTplID string
		if v, ok := kpi["_key"]; ok {
			kpiID = v.(string)
		} else {
			diags.AddWarning("Invalid KPI", fmt.Sprintf("Service %s contains a KPI without an ID", obj.ID.ValueString()))
			continue
		}

		if v, ok := kpi["kpi_threshold_template_id"]; ok {
			thldTplID = v.(string)
		}

		for _, field := range w.thresholdingConfigFields() {
			w.tcs.set(thldConfKey{kpiID, thldTplID}, field, kpi[field])
		}
	}

	return
}

func (w *serviceBuildWorkflow) kpis(ctx context.Context, obj ServiceState) (body map[string]any, diags diag.Diagnostics) {
	itsiKpis := []map[string]any{}
	for _, kpi := range obj.KPIs {
		if kpi.ID.IsUnknown() {
			uuid, _ := uuid.GenerateUUID()
			kpi.ID = types.StringValue(uuid)
		}
		if kpi.Description.IsUnknown() {
			kpi.Description = types.StringNull()
		}
		if kpi.ThresholdTemplateID.IsUnknown() {
			kpi.ThresholdTemplateID = types.StringNull()
		}

		kpiBsID := kpi.BaseSearchID.ValueString()
		kpiBS, err := getKpiBSData(ctx, w.clientConfig, kpiBsID)
		if err != nil {
			diags.AddError("Failed to map KPI BS Data", err.Error())
			return
		}

		kpiID, thldTplID := kpi.ID.ValueString(), kpi.ThresholdTemplateID.ValueString()

		itsiKpi := map[string]any{
			"_key":                       kpiID,
			"title":                      kpi.Title.ValueString(),
			"urgency":                    kpi.Urgency.ValueInt64(),
			"search_type":                kpi.SearchType.ValueString(),
			"type":                       kpi.Type.ValueString(),
			"description":                kpi.Description.ValueString(),
			"base_search_id":             kpiBsID,
			"base_search":                kpiBS["base_search"],
			"is_entity_breakdown":        kpiBS["is_entity_breakdown"],
			"is_service_entity_filter":   kpiBS["is_service_entity_filter"],
			"entity_breakdown_id_fields": kpiBS["entity_breakdown_id_fields"],
			"entity_id_fields":           kpiBS["entity_id_fields"],
			"alert_period":               kpiBS["alert_period"],
			"alert_lag":                  kpiBS["alert_lag"],
			"search_alert_earliest":      kpiBS["search_alert_earliest"],
		}
		if len(kpi.MLThresholding) > 0 {
			rt := kpi.MLThresholding[0]
			thresholdDirection := rt.Direction.ValueString()
			itsiKpi["threshold_direction"] = thresholdDirection
			itsiKpi["recommendation_apply_to_value"] = thresholdDirection
			t, d := rt.StartDate.ValueRFC3339Time()
			if diags.Append(d...); diags.HasError() {
				return
			}
			itsiKpi["recommendation_start_date"] = t.Unix()
			itsiKpi["recommendation_training_window"] = rt.TrainingWindow.ValueString()
			itsiKpi["is_recommended_time_policies"] = true
			itsiKpi["was_recommendation_modified"] = false
			itsiKpi["did_load_recommendation"] = true
		}

		for _, metric := range kpiBS["metrics"].([]any) {
			_metric := metric.(map[string]any)
			if _metric["title"].(string) == kpi.BaseSearchMetric.ValueString() {
				itsiKpi["base_search_metric"] = _metric["_key"].(string)
				for _, metricKey := range []string{"aggregate_statop", "entity_statop", "fill_gaps",
					"gap_custom_alert_value", "gap_severity", "gap_severity_color", "gap_severity_color_light",
					"gap_severity_value", "threshold_field", "unit"} {
					itsiKpi[metricKey] = _metric[metricKey]
				}
			}
		}

		if _, ok := itsiKpi["base_search_metric"]; !ok {
			diags.AddError("Metric Not Found", fmt.Sprintf("%s metric not found", kpi.BaseSearchMetric.ValueString()))
			return
		}

		if kpi.ThresholdTemplateID.IsNull() {
			maps.Copy(itsiKpi, w.tcs.get(thldConfKey{kpiID, thldTplID}))
		} else {
			thresholdTemplateBase, err := kpiThresholdTemplateBase(w.clientConfig, thldTplID, thldTplID).Find(ctx)
			if err != nil {
				diags.AddError("KPI Threshold Template fetching is failed", err.Error())
				return
			}
			if thresholdTemplateBase == nil {
				diags.AddError("thresholdTemplateBase == nil",
					fmt.Sprintf("KPI Threshold Template %s not found", thldTplID))
				return
			}

			thresholdTemplateInterface, err := thresholdTemplateBase.RawJson.ToInterfaceMap()
			if err != nil {
				diags.AddError("KPI Threshold Template is failed to be populated", err.Error())
				return
			}

			itsiKpi["kpi_threshold_template_id"] = thldTplID
			for _, thresholdKey := range w.thresholdingConfigFields() {
				if value, ok := thresholdTemplateInterface[thresholdKey]; ok {
					itsiKpi[thresholdKey] = value
				}
			}

			//populate training data from cache

			if thresholdValueLookup := w.tcs.thldValueLookup(thldConfKey{kpiID, thldTplID}); thresholdValueLookup != nil {
				timeVariateThresholdsSpecification := itsiKpi["time_variate_thresholds_specification"].(map[string]any)
				for policyName, policy := range timeVariateThresholdsSpecification["policies"].(map[string]any) {
					_policy := policy.(map[string]any)
					if aggregate_thresholds, ok := _policy["aggregate_thresholds"].(map[string]any); ok {
						for _, threshold_level := range aggregate_thresholds["thresholdLevels"].([]any) {
							_threshold_level := threshold_level.(map[string]any)
							severityValue := int64(_threshold_level["severityValue"].(float64))
							dynamicParam := _threshold_level["dynamicParam"].(float64)
							v, ok := thresholdValueLookup.get(policyName, severityValue, dynamicParam)
							if !ok {
								continue
							}
							_threshold_level["thresholdValue"] = v
							itsiKpi["time_variate_thresholds_specification"] = timeVariateThresholdsSpecification
						}
					}
				}
			}

		}

		itsiKpis = append(itsiKpis, itsiKpi)
	}

	return map[string]any{"kpis": itsiKpis}, diags
}

func (w *serviceBuildWorkflow) entityRules(_ context.Context, obj ServiceState) (_ map[string]any, diags diag.Diagnostics) {
	itsiEntityRules := []map[string]any{}
	for _, entityRuleGroup := range obj.EntityRules {
		itsiEntityGroupRules := []map[string]any{}
		if len(entityRuleGroup.Rule) == 0 {
			continue
		}

		for _, entityRule := range entityRuleGroup.Rule {
			rule := map[string]any{}
			diags.Append(marshalBasicTypesByTag("json", &entityRule, rule)...)
			itsiEntityGroupRules = append(itsiEntityGroupRules, rule)
		}

		itsiEntityRuleGroup := map[string]any{"rule_condition": "AND", "rule_items": itsiEntityGroupRules}
		itsiEntityRules = append(itsiEntityRules, itsiEntityRuleGroup)
	}
	return map[string]any{"entity_rules": itsiEntityRules}, diags
}

func (w *serviceBuildWorkflow) serviceDependsOn(ctx context.Context, obj ServiceState) (_ map[string]any, diags diag.Diagnostics) {
	itsiServicesDependsOn := []map[string]any{}
	for _, serviceDependsOn := range obj.ServiceDependsOn {
		dependsOnKPIs := []string{}
		diags.Append(serviceDependsOn.KPIs.ElementsAs(ctx, &dependsOnKPIs, false)...)

		if len(dependsOnKPIs) == 0 {
			diags.AddWarning("Glitch on service_depends_on",
				"service_depends_on might contain an unexpected empty element")
			continue
		}

		dependsOnItem := map[string]any{
			"serviceid":         serviceDependsOn.Service.ValueString(),
			"kpis_depending_on": dependsOnKPIs,
		}

		overloaded_urgencies := map[string]int{}
		diags.Append(serviceDependsOn.OverloadedUrgencies.ElementsAs(ctx, &overloaded_urgencies, false)...)

		if len(overloaded_urgencies) > 0 {
			dependsOnItem["overloaded_urgencies"] = overloaded_urgencies
		}

		itsiServicesDependsOn = append(itsiServicesDependsOn, dependsOnItem)
	}
	return map[string]any{"services_depends_on": itsiServicesDependsOn}, diags
}

func (w *serviceBuildWorkflow) tags(ctx context.Context, obj ServiceState) (body map[string]any, diags diag.Diagnostics) {
	var serviceTags []string
	diags.Append(obj.Tags.ElementsAs(ctx, &serviceTags, false)...)

	if len(serviceTags) > 0 {
		body = map[string]any{
			"service_tags": map[string][]string{"tags": serviceTags},
		}
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

	for _, metric_ := range kpiBsData["metrics"].([]any) {
		metric := metric_.(map[string]any)
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

func getKpiBSData(ctx context.Context, cc models.ClientConfig, id string) (map[string]any, error) {
	kpiSearchBase, err := kpiBaseSearchBase(cc, id, "").Find(ctx)
	if err != nil {
		return nil, err
	}

	if kpiSearchBase == nil {
		return nil, fmt.Errorf("KPI Base search %s not found", id)
	}

	return kpiSearchBase.RawJson.ToInterfaceMap()
}
