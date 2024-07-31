package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
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

var _ validator.String = baseSearchValidator{}

const (
	itsiResourceKpiBaseSearch = "kpi_base_search"
)

type baseSearchValidator struct{}

// Description describes the validation in plain text formatting.
func (validator baseSearchValidator) Description(_ context.Context) string {
	return "In KPI base search, the search string shouldn't start with the leading search command"
}

// MarkdownDescription describes the validation in Markdown formatting.
func (validator baseSearchValidator) MarkdownDescription(ctx context.Context) string {
	return validator.Description(ctx)
}

// Validate performs the validation.
func (v baseSearchValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()
	value = strings.TrimSpace(value)

	if strings.HasPrefix(value, "search") {
		response.Diagnostics.Append(validatordiag.InvalidAttributeValueMatchDiagnostic(
			request.Path,
			v.Description(ctx),
			value,
		))
		return
	}
}

func NewKpiBaseSearch() resource.Resource {
	return &resourceKpiBaseSearch{}
}

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &resourceKpiBaseSearch{}
	_ resource.ResourceWithImportState = &resourceKpiBaseSearch{}
	_ resource.ResourceWithConfigure   = &resourceKpiBaseSearch{}
	_ resource.ResourceWithModifyPlan  = &resourceKpiBaseSearch{}

	_ tfmodel = &KpiBaseSearchState{}
)

type KpiBaseSearchState struct {
	ID                         types.String `tfsdk:"id" json:"_key"`
	Title                      types.String `tfsdk:"title" json:"title"`
	Description                types.String `tfsdk:"description" json:"description"`
	Actions                    types.String `tfsdk:"actions" json:"actions"`
	AlertLag                   types.String `tfsdk:"alert_lag" json:"alert_lag"`
	AlertPeriod                types.String `tfsdk:"alert_period" json:"alert_period"`
	BaseSearch                 types.String `tfsdk:"base_search" json:"base_search"`
	EntityAliasFilteringFields types.String `tfsdk:"entity_alias_filtering_fields" json:"entity_alias_filtering_fields"`
	EntityBreakdownIDFields    types.String `tfsdk:"entity_breakdown_id_fields" json:"entity_breakdown_id_fields"`
	EntityIDFields             types.String `tfsdk:"entity_id_fields" json:"entity_id_fields"`
	IsEntityBreakdown          types.Bool   `tfsdk:"is_entity_breakdown" json:"is_entity_breakdown"`
	IsServiceEntityFilter      types.Bool   `tfsdk:"is_service_entity_filter" json:"is_service_entity_filter"`
	MetricQualifier            types.String `tfsdk:"metric_qualifier" json:"metric_qualifier"`
	SearchAlertEarliest        types.String `tfsdk:"search_alert_earliest" json:"search_alert_earliest"`
	SecGrp                     types.String `tfsdk:"sec_grp" json:"sec_grp"`
	SourceItsiDa               types.String `tfsdk:"source_itsi_da" json:"source_itsi_da"`

	Metrics []Metric `tfsdk:"metrics"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

type Metric struct {
	ID                    types.String  `tfsdk:"id" json:"_key"`
	AggregateStatOp       types.String  `tfsdk:"aggregate_statop" json:"aggregate_statop"`
	EntityStatOp          types.String  `tfsdk:"entity_statop" json:"entity_statop"`
	FillGaps              types.String  `tfsdk:"fill_gaps" json:"fill_gaps"`
	GapCustomAlertValue   types.Float64 `tfsdk:"gap_custom_alert_value" json:"gap_custom_alert_value"`
	GapSeverity           types.String  `tfsdk:"gap_severity" json:"gap_severity"`
	GapSeverityValue      types.String  `tfsdk:"gap_severity_value" json:"gap_severity_value"`
	ThresholdField        types.String  `tfsdk:"threshold_field" json:"threshold_field"`
	Title                 types.String  `tfsdk:"title" json:"title"`
	Unit                  types.String  `tfsdk:"unit" json:"unit"`
	GapSeverityColor      types.String  `tfsdk:"gap_severity_color" json:"gap_severity_color"`
	GapSeverityColorLight types.String  `tfsdk:"gap_severity_color_light" json:"gap_severity_color_light"`
}

type resourceKpiBaseSearch struct {
	client models.ClientConfig
}

func (kbs KpiBaseSearchState) objectype() string {
	return itsiResourceKpiBaseSearch
}

func (kbs KpiBaseSearchState) title() string {
	return kbs.Title.String()
}

func kpiBaseSearchBase(clientConfig models.ClientConfig, key string, title string) *models.Base {
	base := models.NewBase(clientConfig, key, title, "kpi_base_search")
	return base
}

const (
	GAP_CUSTOM_ALERT_VALUE_DEFAULT   = "0"
	GAP_SEVERITY_DEFAULT             = "unknown"
	GAP_SEVERITY_COLOR_DEFAULT       = "#CCCCCC"
	GAP_SEVERITY_COLOR_LIGHT_DEFAULT = "#EEEEEE"
	GAP_SEVERITY_VALUE_DEFAULT       = "-1"
)

func (r *resourceKpiBaseSearch) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() {
		return
	}

	var diags diag.Diagnostics
	var state, plan KpiBaseSearchState
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	base := models.NewBase(r.client, "", "", "service")
	params := models.Parameters{
		Fields: []string{
			"_key",
			"title",
			/* Optional: Uncomment in case verbose error message required
			"kpis._key", "kpis.title", "kpis.base_search_id", "kpis.alert_period",
			"kpis.unit", "kpis.aggregate_statop", "kpis.entity_statop", "kpis.threshold_field",
			"kpis.entity_breakdown_id_fields", "kpis.is_entity_breakdown",*/
		},
		Filter: "",
	}

	// aborts plan in case filter matches > 0 service objects
	abortLinkedKpis := func(filter string) {
		params.Filter = filter
		items, err := base.Dump(ctx, &params)
		if err != nil {
			resp.Diagnostics.AddError("Failed to check linked KPIs", err.Error())
		}

		if len(items) > 0 {
			for _, item := range items {
				resp.Diagnostics.AddError(fmt.Sprintf("%s KPI BS is linked to the service", state.Title.ValueString()),
					fmt.Sprintf("_key=%s title=%s\n", item.RESTKey, item.TFID))
			}
			resp.Diagnostics.AddWarning("Filter that found linked KPIs", filter)
		}
	}

	// on destroy
	if req.Plan.Raw.IsNull() {
		abortLinkedKpis(fmt.Sprintf("{\"kpis.base_search_id\":%s}", state.ID))
		return
	}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(diags...)
	properties := []*types.String{
		&plan.Actions, &plan.Description,
		&plan.EntityAliasFilteringFields, &plan.MetricQualifier,
	}

	for _, p := range properties {
		if p.IsUnknown() {
			*p = types.StringNull()
		}
	}

	oldMetricsByTitle := map[string]Metric{}

	// save metrics from state
	if state.Metrics != nil {
		for _, metric := range state.Metrics {
			oldMetricsByTitle[metric.Title.ValueString()] = metric
		}
	}

	// compare with planned metrics, forget ones with unchanged IDs
	planMetrics := []Metric{}
	for _, metricState := range plan.Metrics {
		if metricState.ID.IsUnknown() {
			if metricToRemap, ok := oldMetricsByTitle[metricState.Title.ValueString()]; ok {
				metricState.ID = metricToRemap.ID
				delete(oldMetricsByTitle, metricState.Title.ValueString())
			}
		} else {
			delete(oldMetricsByTitle, metricState.Title.ValueString())
		}

		properties := []struct {
			prop *types.String
			def  string
		}{
			{&metricState.GapSeverity, GAP_SEVERITY_DEFAULT},
			{&metricState.GapSeverityValue, GAP_SEVERITY_VALUE_DEFAULT},
			{&metricState.GapSeverityColor, GAP_SEVERITY_COLOR_DEFAULT},
			{&metricState.GapSeverityColorLight, GAP_SEVERITY_COLOR_LIGHT_DEFAULT},
		}

		for _, p := range properties {
			if p.prop.IsUnknown() {
				*p.prop = types.StringValue(p.def)
			}
		}
		planMetrics = append(planMetrics, metricState)
	}
	plan.Metrics = planMetrics

	if len(oldMetricsByTitle) > 0 {
		filter := []string{}
		for _, metricToCheckLinking := range oldMetricsByTitle {
			filter = append(filter, fmt.Sprintf("{\"kpis.base_search_metric\": %s}", metricToCheckLinking.ID))
		}

		abortLinkedKpis(fmt.Sprintf("{\"$and\": [{\"kpis.base_search_id\":%s}, {\"$or\":[%s]}]}",
			state.ID, strings.Join(filter, ",")))
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
	tflog.Trace(ctx, "Finished modifying plan for collecton data resource")

}

// =================== [ KPI Base Search API / Builder] ===================

type kpiBaseSearchBuildWorkflow struct{}

//lint:ignore U1000 used by apibuilder
func (w *kpiBaseSearchBuildWorkflow) buildSteps() []apibuildWorkflowStepFunc[KpiBaseSearchState] {
	return []apibuildWorkflowStepFunc[KpiBaseSearchState]{
		w.basics,
		w.metrics,
	}
}

func (w *kpiBaseSearchBuildWorkflow) basics(ctx context.Context, obj KpiBaseSearchState) (map[string]any, diag.Diagnostics) {

	body := map[string]interface{}{}
	diags := unmarshalBasicTypesByTag("json", &obj, body)

	body["objectType"] = itsiResourceKpiBaseSearch
	return body, diags
}

func (w *kpiBaseSearchBuildWorkflow) metrics(ctx context.Context, obj KpiBaseSearchState) (map[string]any, diag.Diagnostics) {

	body := map[string]interface{}{}
	metrics := []map[string]interface{}{}

	for _, metricState := range obj.Metrics {
		metric := map[string]interface{}{}
		if metricState.ID.IsUnknown() {
			id, _ := uuid.GenerateUUID()
			metricState.ID = types.StringValue(id)
		}
		diags := unmarshalBasicTypesByTag("json", &metricState, metric)
		if diags.HasError() {
			return nil, diags
		}

		metrics = append(metrics, metric)
	}
	body["metrics"] = metrics
	return body, nil
}

// =================== [ KPI Base Search API / Parser ] ===================

type kpiBaseSearchParseWorkflow struct{}

var _ apiparseWorkflow[KpiBaseSearchState] = &kpiBaseSearchParseWorkflow{}

//lint:ignore U1000 used by apiparser
func (w *kpiBaseSearchParseWorkflow) parseSteps() []apiparseWorkflowStepFunc[KpiBaseSearchState] {
	return []apiparseWorkflowStepFunc[KpiBaseSearchState]{
		w.basics,
		w.metrics,
	}
}

func (w *kpiBaseSearchParseWorkflow) basics(ctx context.Context, fields map[string]any, res *KpiBaseSearchState) (diags diag.Diagnostics) {
	return marshalBasicTypesByTag("json", fields, res)
}

func (w *kpiBaseSearchParseWorkflow) metrics(ctx context.Context, fields map[string]any, res *KpiBaseSearchState) (diags diag.Diagnostics) {
	if v, ok := fields["metrics"]; ok && v != nil {
		metrics, err := unpackSlice[map[string]interface{}](v.([]interface{}))
		if err != nil {
			diags.AddError("Unable to unpack metrics in the KPI BS model", err.Error())
			return
		}
		metricStates := []Metric{}
		for _, metric := range metrics {
			metricState := Metric{}
			diags = append(diags, marshalBasicTypesByTag("json", metric, &metricState)...)
			if diags.HasError() {
				return
			}

			metricStates = append(metricStates, metricState)
		}
		res.Metrics = metricStates
	}
	return
}

func (r *resourceKpiBaseSearch) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureResourceClient(ctx, resourceNameKPIBaseSearch, req, &r.client, resp)
}

func (r *resourceKpiBaseSearch) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	configureResourceMetadata(req, resp, resourceNameKPIBaseSearch)
}

func (r *resourceKpiBaseSearch) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
			"metrics": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Generated metric _key",
						},
						"aggregate_statop": schema.StringAttribute{
							Required:    true,
							Description: "Statistical operation (avg, max, median, stdev, and so on) used to combine data for the aggregate alert_value (used for all KPI).",
							Validators: []validator.String{
								stringvalidator.RegexMatches(regexp.MustCompile("(avg|count|dc|earliest|latest|max|median|min|stdev|sum|perc*)"), ""),
							},
						},
						"entity_statop": schema.StringAttribute{
							Required:    true,
							Description: "Statistical operation (avg, max, mean, and so on) used to combine data for alert_values on a per entity basis (used if entity_breakdown is true).",
							Validators: []validator.String{
								stringvalidator.RegexMatches(regexp.MustCompile("(avg|count|dc|earliest|latest|max|median|min|stdev|sum|perc*)"), ""),
							},
						},
						"fill_gaps": schema.StringAttribute{
							Required:    true,
							Description: "How to fill missing data",
							Validators: []validator.String{
								stringvalidator.OneOf("null_value", "last_available_value", "custom_value"),
							},
						},
						"gap_custom_alert_value": schema.Float64Attribute{
							Optional:    true,
							Computed:    true,
							Description: "Custom value to fill data gaps.",
							//Default:     float64default.StaticFloat64(0),
						},
						"gap_severity": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Severity level assigned for data gaps (info, normal, low, medium, high, critical, or unknown).",
							Validators: []validator.String{
								stringvalidator.OneOf("info", "critical", "high", "medium", "low", "normal", "unknown"),
							},
							//Default: stringdefault.StaticString("unknown"),
						},
						"unit": schema.StringAttribute{
							Required:    true,
							Description: "User-defined units for the values in threshold field.",
						},
						"threshold_field": schema.StringAttribute{
							Required:    true,
							Description: "The field on which the statistical operation runs",
						},
						"title": schema.StringAttribute{
							Required:    true,
							Description: "Name of this metric",
						},
						"gap_severity_color": schema.StringAttribute{
							Computed:    true,
							Optional:    true,
							Description: "Severity color assigned for data gaps.",
							//Default:     stringdefault.StaticString("#CCCCCC"),
						},
						"gap_severity_color_light": schema.StringAttribute{
							Computed:    true,
							Optional:    true,
							Description: "Severity light color assigned for data gaps.",
							//Default:     stringdefault.StaticString("#EEEEEE"),
						},
						"gap_severity_value": schema.StringAttribute{
							Optional:    true,
							Description: "Severity value assigned for data gaps.",
							Computed:    true,
							//Default:     stringdefault.StaticString("-1"),
						},
					},
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				Required:    true,
				Description: "Name of this KPI base search.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "General description for this KPI base search.",
			},
			"actions": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Set of strings, delimited by comma. Corresponds custom actions stanzas, defined in alert_actions.conf.",
				//Default:     stringdefault.StaticString(""),
			},
			"alert_lag": schema.StringAttribute{
				Required:    true,
				Description: "Contains the number of seconds of lag to apply to the alert search, max is 30 minutes (1799 seconds).",
			},
			"alert_period": schema.StringAttribute{
				Required:    true,
				Description: "User specified interval to run the KPI search in minutes.",
			},
			"base_search": schema.StringAttribute{
				Required:    true,
				Description: "KPI search defined by user for this KPI. All generated searches for the KPI are based on this search.",
				Validators:  []validator.String{baseSearchValidator{}},
			},
			"entity_alias_filtering_fields": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Fields from this KPI's search events that will be mapped to the alias fields defined in entities for the service containing this KPI. This field enables the KPI search to tie the aliases of entities to the fields from the KPI events in identifying entities at search time.",
			},
			"entity_breakdown_id_fields": schema.StringAttribute{
				Required:    true,
				Description: "KPI search events are split by the alias field defined in entities for the service containing this KPI",
			},
			"entity_id_fields": schema.StringAttribute{
				Required:    true,
				Description: "Fields from this KPI's search events that will be mapped to the alias fields defined in entities for the service containing this KPI. This field enables the KPI search to tie the aliases of entities to the fields from the KPI events in identifying entities at search time.",
			},
			"is_entity_breakdown": schema.BoolAttribute{
				Required:    true,
				Description: "Determines if search breaks down by entities. See KPI definition.",
			},
			"is_service_entity_filter": schema.BoolAttribute{
				Required:    true,
				Description: "If true a filter is used on the search based on the entities included in the service.",
			},
			"metric_qualifier": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Used to further split metrics. Hidden in the UI.",
			},
			"search_alert_earliest": schema.StringAttribute{
				Required:    true,
				Description: "Value in minutes. This determines how far back each time window is during KPI search runs.",
			},
			"sec_grp": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The team the object belongs to. ",
				Default:     stringdefault.StaticString(itsiDefaultSecurityGroup),
			},
			"source_itsi_da": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
				Description: "Source of DA used for this search. See KPI Threshold Templates.",
				Default:     stringdefault.StaticString("itsi"),
			},
		},
	}
}

func (r *resourceKpiBaseSearch) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan KpiBaseSearchState

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	timeouts := plan.Timeouts
	createTimeout, diags := timeouts.Create(ctx, tftimeout.Create)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	base, diags := newAPIBuilder(r.client, new(kpiBaseSearchBuildWorkflow)).build(ctx, plan)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	base, err := base.Create(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Kpi Base Search", err.Error())
		return
	}

	// populate computed fields
	state, diags := newAPIParser(base, new(kpiBaseSearchParseWorkflow)).parse(ctx, base)
	if resp.Diagnostics.Append(diags...); diags.HasError() {
		return
	}
	state.Timeouts = timeouts
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *resourceKpiBaseSearch) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state KpiBaseSearchState
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	timeouts := state.Timeouts
	readTimeout, diags := timeouts.Read(ctx, tftimeout.Read)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	base := kpiBaseSearchBase(r.client, state.ID.ValueString(), state.Title.ValueString())
	b, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read KPI Base Search", err.Error())
		return
	}
	if b == nil || b.RawJson == nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &KpiBaseSearchState{})...)
		return
	}

	state, diags = newAPIParser(b, new(kpiBaseSearchParseWorkflow)).parse(ctx, b)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	state.Timeouts = timeouts
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *resourceKpiBaseSearch) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan KpiBaseSearchState
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	base, diags := newAPIBuilder(r.client, new(kpiBaseSearchBuildWorkflow)).build(ctx, plan)

	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	timeouts := plan.Timeouts
	updateTimeout, diags := timeouts.Create(ctx, tftimeout.Update)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	existing, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update KPI Base Search", err.Error())
		return
	}
	if existing == nil {
		resp.Diagnostics.AddError("Unable to update KPI Base Search", "KPI Base Search not found")
		return
	}

	diags = base.UpdateAsync(ctx)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	// populate computed fields
	state, diags := newAPIParser(base, new(kpiBaseSearchParseWorkflow)).parse(ctx, base)
	if resp.Diagnostics.Append(diags...); diags.HasError() {
		resp.Diagnostics.AddError("Unable to parse computed fields from Kpi Base Search", err.Error())
		return
	}
	state.Timeouts = timeouts
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *resourceKpiBaseSearch) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state KpiBaseSearchState
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	deleteTimeout, diags := state.Timeouts.Create(ctx, tftimeout.Delete)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	base := kpiBaseSearchBase(r.client, state.ID.ValueString(), state.Title.ValueString())
	b, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete KPI Base Search", err.Error())
		return
	}
	if b == nil {
		return
	}
	resp.Diagnostics.Append(b.Delete(ctx)...)
}

func (r *resourceKpiBaseSearch) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx, cancel := context.WithTimeout(ctx, tftimeout.Read)
	defer cancel()

	b := kpiBaseSearchBase(r.client, "", req.ID)
	b, err := b.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to find KPI Base Search model", err.Error())
		return
	}
	if b == nil {
		resp.Diagnostics.AddError("KPI Base Search not found", fmt.Sprintf("KPI Base Search '%s' not found", req.ID))
		return
	}

	state, diags := newAPIParser(b, new(kpiBaseSearchParseWorkflow)).parse(ctx, b)
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
