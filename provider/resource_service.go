package provider

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
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
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
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
func (r *resourceService) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	configureResourceMetadata(req, resp, resourceNameService)
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
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"service": schema.StringAttribute{
							Required:    true,
							Description: "_key value of service that this service depends on.",
						},
						"kpis": schema.SetAttribute{
						"kpis": schema.SetAttribute{
							Required:    true,
							Description: "A set of _key ids for each KPI in service identified by serviceid, which this service will depend on.",
							ElementType: types.StringType,
						},
						"overloaded_urgencies": schema.MapAttribute{
							Optional:    true,
							Computed:    true,
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func getKpiHashKey(kpiData KpiState, hash_key *string) {
	baseSearchId := kpiData.BaseSearchID.ValueString()
	baseSearchMetricId := kpiData.BaseSearchMetric.ValueString()

	if baseSearchId == "" || baseSearchMetricId == "" {
		// Failed to identify key, do not modify this plan
		return
	}

	hash := sha1.New()
	hash.Write([]byte(baseSearchId + "_" + baseSearchMetricId))
	*hash_key = hex.EncodeToString(hash.Sum(nil))
	return
}

type KpiMapFields struct {
	ID                  types.String
	Description         types.String
	ThresholdTemplateID types.String
	Urgency             types.Int64
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

		// kpiid is important to save for historical raw data. Historical raw data makes sense,
		// until base search & metris is same
		internalIdentifier := ""
		getKpiHashKey(kpi, &internalIdentifier)
		if internalIdentifier == "" {
			resp.Diagnostics.AddError("KPI state missed required fields",
				fmt.Sprintf("no base search data specified, smt went wrong: %s", kpi))
		}

		kpiOldKeys[internalIdentifier] = &KpiMapFields{
			ID: kpi.ID,
		}
	}
	// redefine urgency in case they specified in config
	for _, kpi := range config.KPIs {
		internalIdentifier := ""
		getKpiHashKey(kpi, &internalIdentifier)
		if k, ok := kpiOldKeys[internalIdentifier]; internalIdentifier != "" && ok {
			k.Urgency = kpi.Urgency
			if kpi.Urgency.IsNull() {
				k.Urgency = types.Int64Value(DEFAULT_URGENCY)
			}
			k.Description = kpi.Description
			k.ThresholdTemplateID = kpi.ThresholdTemplateID
		}

	}

	tfKpis := []KpiState{}
	for _, kpi := range plan.KPIs {
		internalIdentifier := ""
		getKpiHashKey(kpi, &internalIdentifier)

		// map kpis in case kpi hash was successfull on get
		if existingKpi, ok := kpiOldKeys[internalIdentifier]; internalIdentifier != "" && ok {
			kpi.ID = existingKpi.ID
			kpi.Urgency = existingKpi.Urgency
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

	base := serviceBase(r.client, state.ID.ValueString(), state.Title.ValueString())
	b, err := base.Read(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read service", err.Error())
		return
	}
	if b == nil || b.RawJson == nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &ServiceState{})...)
		return
	}

	state, diags = serviceModelFromBase(ctx, b)
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

	base, diags := serviceStateToJson(ctx, r.client, &plan)
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

	state, diags := serviceModelFromBase(ctx, base)
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

	updateTimeout, diags := plan.Timeouts.Create(ctx, tftimeout.Update)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	base, diags := serviceStateToJson(ctx, r.client, &plan)
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
	if err := base.UpdateAsync(ctx); err != nil {
		resp.Diagnostics.AddError("Unable to update service", err.Error())
		return
	}

	plan.ID = types.StringValue(base.RESTKey)

	base, err = base.Read(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update Service", err.Error())
		return
	}

	state, diags := serviceModelFromBase(ctx, base)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	state.Timeouts = plan.Timeouts
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

func (r *resourceService) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ServiceState
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	deleteTimeout, diags := state.Timeouts.Create(ctx, tftimeout.Delete)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

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
	ctx, cancel := context.WithTimeout(ctx, tftimeout.Read)
	defer cancel()

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

	var timeouts timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &timeouts)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Timeouts = timeouts

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
	if len(tags) == 0 {
		m.Tags = types.SetNull(types.StringType)
	} else {
		m.Tags, diags = types.SetValueFrom(ctx, types.StringType, tags)
	}

	kpis, err := unpackSlice[map[string]interface{}](interfaceMap["kpis"])
	if err != nil {
		diags.AddError("Unable to unpack KPIs from service model", err.Error())
		return
	}

	m.KPIs = []KpiState{}
	metricLookup := new(KPIBSMetricLookup)

	for _, kpi := range kpis {
		kpiTF := KpiState{}
		diags = append(diags, marshalBasicTypesByTag("json", kpi, &kpiTF)...)

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
				if urgency, err := util.Atoi(val); err == nil {
					kpiTF.Urgency = types.Int64Value(int64(urgency))
				} else {
					diags.AddError("Unable to parse urgency from service model", err.Error())
					continue
				}
			}

			m.KPIs = append(m.KPIs, kpiTF)
		}
	}
	m.EntityRules = []EntityRuleState{}
	entityRules, err := unpackSlice[map[string]interface{}](interfaceMap["entity_rules"])
	if err != nil {
		diags.AddError("Unable to unpack entity rules from service model", err.Error())
		return
	}

	for _, entityRuleAndSet := range entityRules {
		ruleState := EntityRuleState{}
		ruleSet := []RuleState{}
		ruleItems, err := unpackSlice[map[string]interface{}](entityRuleAndSet["rule_items"])
		if err != nil {
			diags.AddError("Unable to unpack rule_item from service model", err.Error())
			return
		}
		for _, ruleItem := range ruleItems {
			ruleTF := RuleState{}
			diags = append(diags, marshalBasicTypesByTag("json", ruleItem, &ruleTF)...)
			ruleSet = append(ruleSet, ruleTF)
		}
		ruleState.Rule = ruleSet
		m.EntityRules = append(m.EntityRules, ruleState)
	}

	m.ServiceDependsOn = []ServiceDependsOnState{}
	serviceDependsOn, err := unpackSlice[map[string]interface{}](interfaceMap["services_depends_on"])
	if interfaceMap["services_depends_on"] != nil && err != nil {
		diags.AddError("Unable to unpack services_depends_on from service model", err.Error())
		return
	}
	for _, serviceDepend := range serviceDependsOn {
		serviceDependsOn := ServiceDependsOnState{}
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

func serviceStateToJson(ctx context.Context, clientConfig models.ClientConfig, m *ServiceState) (config *models.Base, diags diag.Diagnostics) {
	body := map[string]interface{}{}
	config = serviceBase(clientConfig, m.ID.ValueString(), m.Title.ValueString())

	body["object_type"] = "service"
	body["title"] = m.Title.ValueString()
	body["description"] = m.Description.ValueString()

	body["is_healthscore_calculate_by_entity_enabled"] = util.Btoi(m.IsHealthscoreCalculateByEntityEnabled.ValueBool())
	body["enabled"] = util.Btoi(m.Enabled.ValueBool())

	body["sec_grp"] = m.SecurityGroup.ValueString()
		m.Tags, diags = types.SetValueFrom(ctx, types.StringType, tags)
	}

	kpis, err := unpackSlice[map[string]interface{}](interfaceMap["kpis"])
	if err != nil {
		diags.AddError("Unable to unpack KPIs from service model", err.Error())
		return
	}

	m.KPIs = []KpiState{}
	metricLookup := new(KPIBSMetricLookup)

	for _, kpi := range kpis {
		kpiTF := KpiState{}
		diags = append(diags, marshalBasicTypesByTag("json", kpi, &kpiTF)...)

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
				if urgency, err := util.Atoi(val); err == nil {
					kpiTF.Urgency = types.Int64Value(int64(urgency))
				} else {
					diags.AddError("Unable to parse urgency from service model", err.Error())
					continue
				}
			}

			m.KPIs = append(m.KPIs, kpiTF)
		}
	}
	m.EntityRules = []EntityRuleState{}
	entityRules, err := unpackSlice[map[string]interface{}](interfaceMap["entity_rules"])
	if err != nil {
		diags.AddError("Unable to unpack entity rules from service model", err.Error())
		return
	}

	for _, entityRuleAndSet := range entityRules {
		ruleState := EntityRuleState{}
		ruleSet := []RuleState{}
		ruleItems, err := unpackSlice[map[string]interface{}](entityRuleAndSet["rule_items"])
		if err != nil {
			diags.AddError("Unable to unpack rule_item from service model", err.Error())
			return
		}
		for _, ruleItem := range ruleItems {
			ruleTF := RuleState{}
			diags = append(diags, marshalBasicTypesByTag("json", ruleItem, &ruleTF)...)
			ruleSet = append(ruleSet, ruleTF)
		}
		ruleState.Rule = ruleSet
		m.EntityRules = append(m.EntityRules, ruleState)
	}

	m.ServiceDependsOn = []ServiceDependsOnState{}
	serviceDependsOn, err := unpackSlice[map[string]interface{}](interfaceMap["services_depends_on"])
	if interfaceMap["services_depends_on"] != nil && err != nil {
		diags.AddError("Unable to unpack services_depends_on from service model", err.Error())
		return
	}
	for _, serviceDepend := range serviceDependsOn {
		serviceDependsOn := ServiceDependsOnState{}
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

func serviceStateToJson(ctx context.Context, clientConfig models.ClientConfig, m *ServiceState) (config *models.Base, diags diag.Diagnostics) {
	body := map[string]interface{}{}
	config = serviceBase(clientConfig, m.ID.ValueString(), m.Title.ValueString())

	body["object_type"] = "service"
	body["title"] = m.Title.ValueString()
	body["description"] = m.Description.ValueString()

	body["is_healthscore_calculate_by_entity_enabled"] = util.Btoi(m.IsHealthscoreCalculateByEntityEnabled.ValueBool())
	body["enabled"] = util.Btoi(m.Enabled.ValueBool())

	body["sec_grp"] = m.SecurityGroup.ValueString()

	//[kpiId][thresholdId][policyName_severityLabel_dynamicParam]{thresholdValue Float64}
	thresholdValueCache := map[string]map[string]map[string]float64{}
	if m.ID.ValueString() != "" {
		base, err := config.Find(ctx)
	if m.ID.ValueString() != "" {
		base, err := config.Find(ctx)
		if err != nil {
			diags.AddError("Failed to find service object", err.Error())
			return nil, diags
			diags.AddError("Failed to find service object", err.Error())
			return nil, diags
		}

		serviceInterface, err := base.RawJson.ToInterfaceMap()
		if err != nil {
			diags.AddError("Failed to convert service object", err.Error())
			return nil, diags
			diags.AddError("Failed to convert service object", err.Error())
			return nil, diags
		}

		if kpis, ok := serviceInterface["kpis"].([]interface{}); ok {
			for _, kpi := range kpis {
				k := kpi.(map[string]interface{})
				if _, ok := k["_key"]; !ok {
					diags.AddError("Missed KPI", fmt.Sprintf("no kpiId was found for service: %v ", m.ID.ValueString()))
					diags.AddError("Missed KPI", fmt.Sprintf("no kpiId was found for service: %v ", m.ID.ValueString()))
				}
				if _, ok := k["kpi_threshold_template_id"]; !ok || k["kpi_threshold_template_id"].(string) == "" {
					continue
				}

				if _, ok := k["adaptive_thresholds_is_enabled"]; !ok || !k["adaptive_thresholds_is_enabled"].(bool) {
					continue
				}

				kpiId := k["_key"].(string)
				thresholdId := k["kpi_threshold_template_id"].(string)
				thresholdValueCache[kpiId] = map[string]map[string]float64{}
				thresholdValueCache[kpiId][thresholdId] = map[string]float64{}

				if timeVariateThresholdSpecification, ok := k["time_variate_thresholds_specification"].(map[string]interface{}); ok {
					for policyName, policy := range timeVariateThresholdSpecification["policies"].(map[string]interface{}) {
						_policy := policy.(map[string]interface{})
						if aggregate_thresholds, ok := _policy["aggregate_thresholds"].(map[string]interface{}); ok {
							for _, threshold_level := range aggregate_thresholds["thresholdLevels"].([]interface{}) {
								_threshold_level := threshold_level.(map[string]interface{})
								severityValue := _threshold_level["severityValue"].(float64)
								dynamicParam := _threshold_level["dynamicParam"].(float64)
								thresholdValue := _threshold_level["thresholdValue"].(float64)
								key := policyName + fmt.Sprint(severityValue) + "_" + fmt.Sprint(dynamicParam)
								thresholdValueCache[kpiId][thresholdId][key] = thresholdValue
							}
						}
					}
				}

			}
		}
	}

	itsiKpis := []map[string]interface{}{}
	tfKpis := []KpiState{}
	for _, kpi := range m.KPIs {
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

		restKey := kpi.BaseSearchID.ValueString()
	itsiKpis := []map[string]interface{}{}
	tfKpis := []KpiState{}
	for _, kpi := range m.KPIs {
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

		restKey := kpi.BaseSearchID.ValueString()
		kpiSearchInterface, err := getKpiBSData(ctx, clientConfig, restKey)
		if err != nil {
			diags.AddError("Failed to map KPI BS Data", err.Error())
			return
			diags.AddError("Failed to map KPI BS Data", err.Error())
			return
		}

		itsiKpi := map[string]interface{}{
			"title":                      kpi.Title.ValueString(),
			"urgency":                    kpi.Urgency.ValueInt64(),
			"search_type":                kpi.SearchType.ValueString(),
			"type":                       kpi.Type.ValueString(),
			"description":                kpi.Description.ValueString(),
			"title":                      kpi.Title.ValueString(),
			"urgency":                    kpi.Urgency.ValueInt64(),
			"search_type":                kpi.SearchType.ValueString(),
			"type":                       kpi.Type.ValueString(),
			"description":                kpi.Description.ValueString(),
			"base_search_id":             restKey,
			"base_search":                kpiSearchInterface["base_search"],
			"is_entity_breakdown":        kpiSearchInterface["is_entity_breakdown"],
			"is_service_entity_filter":   kpiSearchInterface["is_service_entity_filter"],
			"entity_breakdown_id_fields": kpiSearchInterface["entity_breakdown_id_fields"],
			"entity_id_fields":           kpiSearchInterface["entity_id_fields"],
			"alert_period":               kpiSearchInterface["alert_period"],
			"alert_lag":                  kpiSearchInterface["alert_lag"],
			"search_alert_earliest":      kpiSearchInterface["search_alert_earliest"],
		}

		for _, metric := range kpiSearchInterface["metrics"].([]interface{}) {
			_metric := metric.(map[string]interface{})
			if _metric["title"].(string) == kpi.BaseSearchMetric.ValueString() {
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
			diags.AddError("Metric Not Found", fmt.Sprintf("%s metric not found", kpi.BaseSearchMetric.ValueString()))
			return
		}

		itsiKpi["_key"] = kpi.ID.ValueString()
		itsiKpi["_key"] = kpi.ID.ValueString()

		if !kpi.ThresholdTemplateID.IsNull() {
			thresholdRestKey := kpi.ThresholdTemplateID.ValueString()
		if !kpi.ThresholdTemplateID.IsNull() {
			thresholdRestKey := kpi.ThresholdTemplateID.ValueString()
			thresholdTemplateBase := kpiThresholdTemplateBase(clientConfig, thresholdRestKey, thresholdRestKey)

			thresholdTemplateBase, err = thresholdTemplateBase.Find(ctx)
			if err != nil {
				diags.AddError("KPI Threshold Template fetching is failed", err.Error())
				return
				diags.AddError("KPI Threshold Template fetching is failed", err.Error())
				return
			}
			if thresholdTemplateBase == nil {
				diags.AddError("thresholdTemplateBase == nil",
					fmt.Sprintf("KPI Threshold Template %s not found", thresholdRestKey))
				return
				diags.AddError("thresholdTemplateBase == nil",
					fmt.Sprintf("KPI Threshold Template %s not found", thresholdRestKey))
				return
			}

			thresholdTemplateInterface, err := thresholdTemplateBase.RawJson.ToInterfaceMap()
			if err != nil {
				diags.AddError("KPI Threshold Template is failed to be populated", err.Error())
				return
				diags.AddError("KPI Threshold Template is failed to be populated", err.Error())
				return
			}

			itsiKpi["kpi_threshold_template_id"] = thresholdRestKey
			for _, thresholdKey := range []string{"time_variate_thresholds", "adaptive_thresholds_is_enabled",
				"adaptive_thresholding_training_window", "aggregate_thresholds", "entity_thresholds",
				"time_variate_thresholds_specification"} {
				if value, ok := thresholdTemplateInterface[thresholdKey]; ok {
					itsiKpi[thresholdKey] = value
				}
			}

			//populate training data from cache
			id := kpi.ID.ValueString()
			id := kpi.ID.ValueString()
			if _, ok := thresholdValueCache[id]; ok {
				if currentThresholdCache, ok := thresholdValueCache[id][thresholdRestKey]; ok {
					timeVariateThresholdsSpecification := itsiKpi["time_variate_thresholds_specification"].(map[string]interface{})
					for policyName, policy := range timeVariateThresholdsSpecification["policies"].(map[string]interface{}) {
						_policy := policy.(map[string]interface{})
						if aggregate_thresholds, ok := _policy["aggregate_thresholds"].(map[string]interface{}); ok {
							for _, threshold_level := range aggregate_thresholds["thresholdLevels"].([]interface{}) {
								_threshold_level := threshold_level.(map[string]interface{})
								severityValue := _threshold_level["severityValue"].(float64)
								dynamicParam := _threshold_level["dynamicParam"].(float64)
								key := policyName + fmt.Sprint(severityValue) + "_" + fmt.Sprint(dynamicParam)
								_threshold_level["thresholdValue"] = currentThresholdCache[key]

								itsiKpi["time_variate_thresholds_specification"] = timeVariateThresholdsSpecification
							}
						}
					}
				}
			}
		}

		itsiKpis = append(itsiKpis, itsiKpi)
		tfKpis = append(tfKpis, kpi)
		tfKpis = append(tfKpis, kpi)
	}

	body["kpis"] = itsiKpis
	m.KPIs = tfKpis
	m.KPIs = tfKpis

	//entity rules
	itsiEntityRules := []map[string]interface{}{}
	for _, entityRuleGroup := range m.EntityRules {
		itsiEntityGroupRules := []map[string]interface{}{}
		if len(entityRuleGroup.Rule) == 0 {
	for _, entityRuleGroup := range m.EntityRules {
		itsiEntityGroupRules := []map[string]interface{}{}
		if len(entityRuleGroup.Rule) == 0 {
			continue
		}

		for _, entityRule := range entityRuleGroup.Rule {
			rule := map[string]interface{}{}
			unmarshalBasicTypesByTag("json", &entityRule, rule)
			itsiEntityGroupRules = append(itsiEntityGroupRules, rule)
		for _, entityRule := range entityRuleGroup.Rule {
			rule := map[string]interface{}{}
			unmarshalBasicTypesByTag("json", &entityRule, rule)
			itsiEntityGroupRules = append(itsiEntityGroupRules, rule)
		}

		itsiEntityRuleGroup := map[string]interface{}{"rule_condition": "AND", "rule_items": itsiEntityGroupRules}
		itsiEntityRules = append(itsiEntityRules, itsiEntityRuleGroup)
	}
	body["entity_rules"] = itsiEntityRules

	//service depends on
	itsiServicesDependsOn := []map[string]interface{}{}
	for _, serviceDependsOn := range m.ServiceDependsOn {
		dependsOnKPIs := []string{}
		diags = append(diags, serviceDependsOn.KPIs.ElementsAs(ctx, &dependsOnKPIs, false)...)

	for _, serviceDependsOn := range m.ServiceDependsOn {
		dependsOnKPIs := []string{}
		diags = append(diags, serviceDependsOn.KPIs.ElementsAs(ctx, &dependsOnKPIs, false)...)

		if len(dependsOnKPIs) == 0 {
			diags.AddWarning("Glitch on service_depends_on",
				"service_depends_on might contain an unexpected empty element")
			diags.AddWarning("Glitch on service_depends_on",
				"service_depends_on might contain an unexpected empty element")
			continue
		}

		dependsOnItem := map[string]interface{}{
			"serviceid":         serviceDependsOn.Service.ValueString(),
			"serviceid":         serviceDependsOn.Service.ValueString(),
			"kpis_depending_on": dependsOnKPIs,
		}

		overloaded_urgencies := map[string]int{}
		diags = append(diags, serviceDependsOn.OverloadedUrgencies.ElementsAs(ctx, &overloaded_urgencies, false)...)

		overloaded_urgencies := map[string]int{}
		diags = append(diags, serviceDependsOn.OverloadedUrgencies.ElementsAs(ctx, &overloaded_urgencies, false)...)

		if len(overloaded_urgencies) > 0 {
			dependsOnItem["overloaded_urgencies"] = overloaded_urgencies
		}

		itsiServicesDependsOn = append(itsiServicesDependsOn, dependsOnItem)
	}
	body["services_depends_on"] = itsiServicesDependsOn

	//tags
	var serviceTags []string
	diags = append(diags, m.Tags.ElementsAs(ctx, &serviceTags, false)...)

	diags = append(diags, m.Tags.ElementsAs(ctx, &serviceTags, false)...)

	if len(serviceTags) > 0 {
		body["service_tags"] = map[string][]string{"tags": serviceTags}
	}

	if err := config.PopulateRawJSON(ctx, body); err != nil {
		diags.AddError("Unable to populate base object", err.Error())
	}

	return
}

/* helper data structure to allow us specify metrics by title rather than ID */

type KPIBSMetricLookup struct {
	titleByKpiBsIDandMetricID map[string]string
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
