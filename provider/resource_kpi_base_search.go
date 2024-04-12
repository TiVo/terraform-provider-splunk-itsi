package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

var _ validator.String = baseSearchValidator{}

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
)

func kpiBaseSearchBase(clientConfig models.ClientConfig, key string, title string) *models.Base {
	base := models.NewBase(clientConfig, key, title, "kpi_base_search")
	return base
}

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
	oldMetricsByTitle := map[string]*Metric{}

	// save metrics from state
	if state.Metrics != nil {
		for _, metric := range state.Metrics {
			oldMetricsByTitle[metric.Title.ValueString()] = metric
		}
	}

	// compare with planned metrics, forget ones with unchanged IDs
	for _, metricState := range plan.Metrics {
		if metricState.ID.IsUnknown() {
			if metricToRemap, ok := oldMetricsByTitle[metricState.Title.ValueString()]; ok {
				metricState.ID = metricToRemap.ID
				delete(oldMetricsByTitle, metricState.Title.ValueString())
			}
		} else {
			delete(oldMetricsByTitle, metricState.Title.ValueString())
		}
	}

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

	Metrics []*Metric `tfsdk:"metrics"`
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

func (r *resourceKpiBaseSearch) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *resourceKpiBaseSearch) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kpi_base_search"
}

func (r *resourceKpiBaseSearch) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Blocks: map[string]schema.Block{
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
							Default:     float64default.StaticFloat64(0),
						},
						"gap_severity": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Severity level assigned for data gaps (info, normal, low, medium, high, critical, or unknown).",
							Validators: []validator.String{
								stringvalidator.OneOf("info", "critical", "high", "medium", "low", "normal", "unknown"),
							},
							Default: stringdefault.StaticString("unknown"),
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
							Optional:    true,
							Computed:    true,
							Description: "Severity color assigned for data gaps.",
							Default:     stringdefault.StaticString("#CCCCCC"),
						},
						"gap_severity_color_light": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Severity light color assigned for data gaps.",
							Default:     stringdefault.StaticString("#EEEEEE"),
						},
						"gap_severity_value": schema.StringAttribute{
							Optional:    true,
							Description: "Severity value assigned for data gaps.",
							Computed:    true,
							Default:     stringdefault.StaticString("-1"),
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
				Description: "General description for this KPI base search.",
			},
			"actions": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Set of strings, delimited by comma. Corresponds custom actions stanzas, defined in alert_actions.conf.",
				Default:     stringdefault.StaticString(""),
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
				Required:    false,
				Optional:    true,
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
				Description: "Used to further split metrics. Hidden in the UI.",
			},
			"search_alert_earliest": schema.StringAttribute{
				Required:    true,
				Description: "Value in minutes. This determines how far back each time window is during KPI search runs.",
			},
			"sec_grp": schema.StringAttribute{
				Required:    true,
				Description: "The team the object belongs to. ",
			},
			"source_itsi_da": schema.StringAttribute{
				Required:    true,
				Description: "Source of DA used for this search. See KPI Threshold Templates.",
			},
		},
	}
}

func (r *resourceKpiBaseSearch) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan KpiBaseSearchState

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	base, diags := kpiBaseSearchJson(ctx, r.client, plan)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	base, err := base.Create(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create entity", err.Error())
		return
	}

	plan.ID = types.StringValue(base.RESTKey)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *resourceKpiBaseSearch) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state KpiBaseSearchState
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

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

	state, diags := kpiBaseSearchModelFromBase(ctx, b)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *resourceKpiBaseSearch) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan KpiBaseSearchState
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	base, diags := kpiBaseSearchJson(ctx, r.client, plan)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	existing, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update KPI Base Search", err.Error())
		return
	}
	if existing == nil {
		resp.Diagnostics.AddError("Unable to update KPI Base Search", "KPI Base Search not found")
		return
	}
	if err := base.Update(ctx); err != nil {
		resp.Diagnostics.AddError("Unable to update KPI Base Search", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *resourceKpiBaseSearch) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state KpiBaseSearchState
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	base := kpiBaseSearchBase(r.client, state.ID.ValueString(), state.Title.ValueString())
	b, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete KPI Base Search", err.Error())
		return
	}
	if b == nil {
		return
	}
	if err := b.Delete(ctx); err != nil {
		resp.Diagnostics.AddError("Unable to delete KPI Base Search", err.Error())
		return
	}
}

func (r *resourceKpiBaseSearch) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

	state, diags := kpiBaseSearchModelFromBase(ctx, b)
	if resp.Diagnostics.Append(diags...); diags.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func kpiBaseSearchModelFromBase(_ context.Context, b *models.Base) (m KpiBaseSearchState, diags diag.Diagnostics) {
	//var d diag.Diagnostics
	if b == nil || b.RawJson == nil {
		diags.AddError("Unable to populate entity model", "base object is nil or empty.")
		return
	}

	interfaceMap, err := b.RawJson.ToInterfaceMap()
	if err != nil {
		diags.AddError("Unable to populate KPI Base search model", err.Error())
		return
	}

	diags = append(diags, marshalBasicTypesByTag("json", interfaceMap, &m)...)

	if v, ok := interfaceMap["metrics"]; ok && v != nil {
		metrics, err := unpackSlice[map[string]interface{}](v.([]interface{}))
		if err != nil {
			diags.AddError("Unable to unpack metrics in the KPI BS model", err.Error())
			return
		}
		metricStates := []*Metric{}
		for _, metric := range metrics {
			metricState := &Metric{}
			diags = append(diags, marshalBasicTypesByTag("json", metric, metricState)...)
			if diags.HasError() {
				return
			}

			metricStates = append(metricStates, metricState)
		}
		m.Metrics = metricStates
	}

	m.ID = types.StringValue(b.RESTKey)

	return
}

func kpiBaseSearchJson(ctx context.Context, clientConfig models.ClientConfig, m KpiBaseSearchState) (config *models.Base, diags diag.Diagnostics) {

	body := map[string]interface{}{}
	diags = append(diags, unmarshalBasicTypesByTag("json", &m, body)...)
	if diags.HasError() {
		return
	}

	metrics := []map[string]interface{}{}
	for _, metricState := range m.Metrics {
		metric := map[string]interface{}{}
		if metricState.ID.IsUnknown() {
			id, _ := uuid.GenerateUUID()
			metricState.ID = types.StringValue(id)
		}
		diags = append(diags, unmarshalBasicTypesByTag("json", metricState, metric)...)
		if diags.HasError() {
			return
		}

		metrics = append(metrics, metric)
	}
	body["metrics"] = metrics

	config = kpiBaseSearchBase(clientConfig, m.ID.ValueString(), m.Title.ValueString())
	if err := config.PopulateRawJSON(ctx, body); err != nil {
		diags.AddError("Unable to populate base object", err.Error())
	}
	return
}
