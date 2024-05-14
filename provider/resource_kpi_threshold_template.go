package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
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
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

func kpiThresholdTemplateBase(clientConfig models.ClientConfig, key string, title string) *models.Base {
	base := models.NewBase(clientConfig, key, title, "kpi_threshold_template")
	return base
}

var (
	_ resource.Resource                = &resourceKpiThresholdTemplate{}
	_ resource.ResourceWithConfigure   = &resourceKpiThresholdTemplate{}
	_ resource.ResourceWithImportState = &resourceKpiThresholdTemplate{}
)

func NewResourceKpiThresholdTemplate() resource.Resource {
	return &resourceKpiThresholdTemplate{}
}

type modelKpiThresholdTemplate struct {
	ID                                 types.String                             `tfsdk:"id" json:"_key"`
	Title                              types.String                             `tfsdk:"title" json:"title"`
	Description                        types.String                             `tfsdk:"description" json:"description"`
	AdaptiveThresholdingTrainingWindow types.String                             `tfsdk:"adaptive_thresholding_training_window" json:"adaptive_thresholding_training_window"`
	TimeVariateThresholds              types.Bool                               `tfsdk:"time_variate_thresholds" json:"time_variate_thresholds"`
	TimeVariateThresholdsSpecification *TimeVariateThresholdsSpecificationModel `tfsdk:"time_variate_thresholds_specification"`
	AdaptiveThresholdsIsEnabled        types.Bool                               `tfsdk:"adaptive_thresholds_is_enabled" json:"adaptive_thresholds_is_enabled"`
	SecGrp                             types.String                             `tfsdk:"sec_grp" json:"sec_grp"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

type TimeVariateThresholdsSpecificationModel struct {
	Policies []PolicyModel `tfsdk:"policies"`
}

type PolicyModel struct {
	PolicyName          types.String          `tfsdk:"policy_name"`
	Title               types.String          `tfsdk:"title"`
	PolicyType          types.String          `tfsdk:"policy_type"`
	TimeBlocks          []TimeBlockModel      `tfsdk:"time_blocks"`
	AggregateThresholds ThresholdSettingModel `tfsdk:"aggregate_thresholds"`
	EntityThresholds    ThresholdSettingModel `tfsdk:"entity_thresholds"`
}

type ThresholdSettingModel struct {
	BaseSeverityLabel types.String             `json:"baseSeverityLabel" tfsdk:"base_severity_label"`
	GaugeMax          types.Float64            `json:"gaugeMax" tfsdk:"gauge_max"`
	GaugeMin          types.Float64            `json:"gaugeMin" tfsdk:"gauge_min"`
	IsMaxStatic       types.Bool               `json:"isMaxStatic" tfsdk:"is_max_static"`
	IsMinStatic       types.Bool               `json:"isMinStatic" tfsdk:"is_min_static"`
	MetricField       types.String             `json:"metricField" tfsdk:"metric_field"`
	RenderBoundaryMax types.Float64            `json:"renderBoundaryMax" tfsdk:"render_boundary_max"`
	RenderBoundaryMin types.Float64            `json:"renderBoundaryMin" tfsdk:"render_boundary_min"`
	ThresholdLevels   []KpiThresholdLevelModel `tfsdk:"threshold_levels"`
}

type KpiThresholdLevelModel struct {
	SeverityLabel  types.String  `json:"severityLabel" tfsdk:"severity_label"`
	ThresholdValue types.Float64 `json:"thresholdValue" tfsdk:"threshold_value"`
	DynamicParam   types.Float64 `json:"dynamicParam" tfsdk:"dynamic_param"`
}

func getKpiThresholdSettingsBlocksAttrs() (map[string]schema.Block, map[string]schema.Attribute) {
	return map[string]schema.Block{
			"threshold_levels": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"severity_label": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.OneOf("info", "critical", "high", "medium", "low", "normal"),
							},
							Description: "Severity label assigned for this threshold level like info, warning, critical, etc",
						},
						"threshold_value": schema.Float64Attribute{
							Required: true,
							Description: `Value for the threshold field stats identifying this threshold level.
							This is the key value that defines the levels for values derived from the KPI search metrics.`,
						},
						"dynamic_param": schema.Float64Attribute{
							Required:    true,
							Description: "Value of the dynamic parameter for adaptive thresholds",
						},
					},
				},
			},
		},
		map[string]schema.Attribute{
			"base_severity_label": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf("info", "critical", "high", "medium", "low", "normal"),
				},
				Description: "Base severity label assigned for the threshold (info, normal, low, medium, high, critical). ",
			},
			"gauge_max": schema.Float64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Maximum value for the threshold gauge specified by user",
			},
			"gauge_min": schema.Float64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Minimum value for the threshold gauge specified by user.",
			},
			"is_max_static": schema.BoolAttribute{
				Required:    true,
				Description: "True when maximum threshold value is a static value, false otherwise. ",
			},
			"is_min_static": schema.BoolAttribute{
				Required:    true,
				Description: "True when min threshold value is a static value, false otherwise.",
			},
			"metric_field": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Thresholding field from the search.",
			},
			"render_boundary_max": schema.Float64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Upper bound value to use to render the graph for the thresholds.",
			},
			"render_boundary_min": schema.Float64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Lower bound value to use to render the graph for the thresholds.",
			},
		}
}

func kpiThresholdSettingsToModel(attrName string, apiThresholdSetting map[string]interface{}, tfthresholdSettingModel *ThresholdSettingModel, settingType string) (diags diag.Diagnostics) {
	diags.Append(marshalBasicTypesByTag("json", apiThresholdSetting, tfthresholdSettingModel)...)

	thresholdLevels := []KpiThresholdLevelModel{}
	for _, tData_ := range apiThresholdSetting["thresholdLevels"].([]interface{}) {
		tData := tData_.(map[string]interface{})
		thresholdLevel := &KpiThresholdLevelModel{}
		switch tData["dynamicParam"] {
		case "":
			if settingType != "static" {
				diags.AddError("Failed to populate aggregated threshold", fmt.Sprintf("empty dynamic param for adaptive policy %s", settingType))
				return
			}
			tData["dynamicParam"] = 0
		}
		diags.Append(marshalBasicTypesByTag("json", tData, thresholdLevel)...)
		thresholdLevels = append(thresholdLevels, *thresholdLevel)
	}
	tfthresholdSettingModel.ThresholdLevels = thresholdLevels
	return
}

func kpiThresholdThresholdSettingsAttributesToPayload(_ context.Context, source ThresholdSettingModel) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics
	thresholdSetting := map[string]interface{}{}
	diags.Append(unmarshalBasicTypesByTag("json", &source, thresholdSetting)...)

	if severity, ok := util.SeverityMap[source.BaseSeverityLabel.ValueString()]; ok {
		thresholdSetting["baseSeverityColor"] = severity.SeverityColor
		thresholdSetting["baseSeverityColorLight"] = severity.SeverityColorLight
		thresholdSetting["baseSeverityLabel"] = severity.SeverityLabel
		thresholdSetting["baseSeverityValue"] = severity.SeverityValue
	} else if !source.BaseSeverityLabel.IsNull() {
		diags.AddError("failed to convert threshold setting model to payload", fmt.Sprintf("schema Validation broken. Unknown severity %s", source.BaseSeverityLabel.ValueString()))
		return nil, diags
	}
	thresholdLevels := []interface{}{}

	for _, tfThresholdLevel := range source.ThresholdLevels {
		thresholdLevel := map[string]interface{}{}
		thresholdLevel["dynamicParam"] = tfThresholdLevel.DynamicParam.ValueFloat64()
		if severity, ok := util.SeverityMap[tfThresholdLevel.SeverityLabel.ValueString()]; ok {
			thresholdLevel["severityColor"] = severity.SeverityColor
			thresholdLevel["severityColorLight"] = severity.SeverityColorLight
			thresholdLevel["severityLabel"] = severity.SeverityLabel
			thresholdLevel["severityValue"] = severity.SeverityValue
		} else {
			diags.AddError("schema Validation broken. Unknown severity %s", tfThresholdLevel.SeverityLabel.ValueString())
			return nil, diags
		}
		thresholdLevel["thresholdValue"] = tfThresholdLevel.ThresholdValue.ValueFloat64()
		thresholdLevels = append(thresholdLevels, thresholdLevel)
	}
	thresholdSetting["thresholdLevels"] = thresholdLevels
	return thresholdSetting, diags
}

type TimeBlockModel struct {
	Interval types.Int64  `tfsdk:"interval"`
	Cron     types.String `tfsdk:"cron"`
}
type resourceKpiThresholdTemplate struct {
	client models.ClientConfig
}

const (
	BASE_SEVERITY_LABEL_DEFAULT = "normal"
)

func (r *resourceKpiThresholdTemplate) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var plan, config modelKpiThresholdTemplate
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if config.Description.IsNull() && plan.Description.IsUnknown() {
		plan.Description = types.StringValue("")
	}

	populateDefaults := func(tsm *ThresholdSettingModel) {
		properties := []*types.Float64{
			&tsm.GaugeMax, &tsm.GaugeMin,
			&tsm.RenderBoundaryMax, &tsm.RenderBoundaryMin,
		}

		for _, p := range properties {
			if p.IsUnknown() {
				*p = types.Float64Null()
			}
		}

		if tsm.MetricField.IsUnknown() {
			tsm.MetricField = types.StringNull()
		}
		if tsm.BaseSeverityLabel.IsUnknown() {
			tsm.BaseSeverityLabel = types.StringValue(BASE_SEVERITY_LABEL_DEFAULT)
		}
	}
	if plan.TimeVariateThresholdsSpecification != nil {
		policies := []PolicyModel{}
		for _, policy := range plan.TimeVariateThresholdsSpecification.Policies {
			populateDefaults(&policy.AggregateThresholds)
			populateDefaults(&policy.EntityThresholds)

			policies = append(policies, policy)
		}
		plan.TimeVariateThresholdsSpecification.Policies = policies
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
	tflog.Trace(ctx, "Finished modifying plan for kpi threshold template resource")

}

// Metadata returns the resource type name.
func (r *resourceKpiThresholdTemplate) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	configureResourceMetadata(req, resp, resourceNameKPIThresholdTemplate)
}

func (r *resourceKpiThresholdTemplate) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureResourceClient(ctx, resourceNameKPIThresholdTemplate, req, &r.client, resp)
}

func (r *resourceKpiThresholdTemplate) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	threshold_settings_blocks, threshold_settings_attributes := getKpiThresholdSettingsBlocksAttrs()

	resp.Schema = schema.Schema{
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
			"time_variate_thresholds_specification": schema.SingleNestedBlock{
				Blocks: map[string]schema.Block{

					"policies": schema.SetNestedBlock{
						Description: "Map object of policies keyed by policy_name. ",
						NestedObject: schema.NestedBlockObject{
							Blocks: map[string]schema.Block{
								"time_blocks": schema.SetNestedBlock{
									//Optional: true,
									NestedObject: schema.NestedBlockObject{
										Attributes: map[string]schema.Attribute{
											"interval": schema.Int64Attribute{
												Required:    true,
												Description: "Corresponds to the cron expression in format: {minute} {hour} {\\*} {\\*} {day}",
											},
											"cron": schema.StringAttribute{
												Required:    true,
												Description: "Corresponds to the cron expression in format: {minute} {hour} {\\*} {\\*} {day}",
											},
										},
									},
								},
								"aggregate_thresholds": schema.SingleNestedBlock{
									Description: "User-defined thresholding levels for \"Aggregate\" threshold type. For more information, see KPI Threshold Setting.",
									Attributes:  threshold_settings_attributes,
									Blocks:      threshold_settings_blocks,
								},
								"entity_thresholds": schema.SingleNestedBlock{
									Description: "User-defined thresholding levels for \"Per Entity\" threshold type. For more information, see KPI Threshold Setting.",
									Attributes:  threshold_settings_attributes,
									Blocks:      threshold_settings_blocks,
								},
							},
							Attributes: map[string]schema.Attribute{
								"policy_name": schema.StringAttribute{
									Required:    true,
									Description: "Internal key value for policy.",
								},
								"title": schema.StringAttribute{
									Required:    true,
									Description: "The policy title, displayed to the user in the UI. Should be unique per policies object.",
								},
								"policy_type": schema.StringAttribute{
									Required: true,
									Description: `The algorithm, specified for the current policy threshold level evaluation.
													Supported values: static, stdev (standard deviation), quantile, range and percentage.`,
									Validators: []validator.String{
										stringvalidator.OneOf("static", "stdev", "quantile", "range", "percentage"),
									},
								},
							},
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
				Required: true,
				//ForceNew:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Name of this KPI threshold template.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "User-defined description for the kpi Threshold Template.",
			},
			"adaptive_thresholds_is_enabled": schema.BoolAttribute{
				Required:    true,
				Description: "Determines if adaptive threshold is enabled for this KPI threshold template.",
			},
			"adaptive_thresholding_training_window": schema.StringAttribute{
				Required:    true,
				Description: "The earliest time for the Adaptive Threshold training algorithm to run over (latest time is always 'now') (e.g. '-7d')",
			},
			"time_variate_thresholds": schema.BoolAttribute{
				Required:    true,
				Description: "If true, thresholds for alerts are pulled from time_variate_thresholds_specification.",
			},
			"sec_grp": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The team the object belongs to. ",
				Default:     stringdefault.StaticString(itsiDefaultSecurityGroup),
			},
		},
	}
}

func (r *resourceKpiThresholdTemplate) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan modelKpiThresholdTemplate
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, tftimeout.Create)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	template, diags := kpiThresholdTemplate(ctx, plan, r.client)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	b, err := template.Create(ctx)
	if err != nil {
		diags.AddError("Failed to create kpi threshold template.", err.Error())
		resp.Diagnostics.Append(diags...)
		return
	}
	resp.Diagnostics.Append(populateKpiThresholdTemplateModel(ctx, b, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *resourceKpiThresholdTemplate) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state modelKpiThresholdTemplate
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := state.Timeouts.Read(ctx, tftimeout.Read)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	base := kpiThresholdTemplateBase(r.client, state.ID.ValueString(), state.Title.ValueString())
	b, err := base.Find(ctx)
	if err != nil || b == nil {
		diags.AddError("Failed to find kpi threshold template.", err.Error())
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(populateKpiThresholdTemplateModel(ctx, b, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *resourceKpiThresholdTemplate) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var plan modelKpiThresholdTemplate
	var state modelKpiThresholdTemplate

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := plan.Timeouts.Create(ctx, tftimeout.Update)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	base := kpiThresholdTemplateBase(r.client, state.ID.ValueString(), plan.Title.ValueString())
	existing, err := base.Find(ctx)
	if err != nil {
		diags.AddError("Failed to find kpi threshold template.", err.Error())
		resp.Diagnostics.Append(diags...)
		return
	}
	if existing == nil {
		_, err := base.Create(ctx)
		if err != nil {
			diags.AddError("Failed to create kpi threshold template.", err.Error())
			resp.Diagnostics.Append(diags...)
			return
		}
	}
	plan.ID = types.StringValue(base.RESTKey)
	if resp.Diagnostics.HasError() {
		return
	}

	base, diags = kpiThresholdTemplate(ctx, plan, r.client)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	err = base.Update(ctx)
	if err != nil {
		diags.AddError("Failed to update kpi threshold template.", err.Error())
		resp.Diagnostics.Append(diags...)
		return
	}
	resp.Diagnostics.Append(populateKpiThresholdTemplateModel(ctx, base, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Set refreshed state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *resourceKpiThresholdTemplate) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state modelKpiThresholdTemplate

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := state.Timeouts.Create(ctx, tftimeout.Delete)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	base := kpiThresholdTemplateBase(r.client, state.ID.ValueString(), state.Title.ValueString())
	existing, err := base.Find(ctx)
	if err != nil {
		diags.AddError("Failed to find kpi threshold template.", err.Error())
		resp.Diagnostics.Append(diags...)
		return
	}
	if existing == nil {
		return
	}

	err = existing.Delete(ctx)
	if err != nil {
		diags.AddError("Failed to delete kpi threshold template.", err.Error())
	}
	resp.Diagnostics.Append(diags...)
}

func kpiThresholdTemplate(ctx context.Context, tfKpiThresholdTemplate modelKpiThresholdTemplate, clientConfig models.ClientConfig) (config *models.Base, diags diag.Diagnostics) {
	body := map[string]interface{}{}
	diags = append(diags, unmarshalBasicTypesByTag("json", &tfKpiThresholdTemplate, body)...)
	body["objectType"] = "kpi_threshold_template"

	policies := map[string]interface{}{}
	if tfKpiThresholdTemplate.TimeVariateThresholdsSpecification != nil {
		for _, tfpolicy := range tfKpiThresholdTemplate.TimeVariateThresholdsSpecification.Policies {
			policy := map[string]interface{}{}
			policy["title"] = tfpolicy.Title.ValueString()
			policy["policy_type"] = tfpolicy.PolicyType.ValueString()
			timeBlocks := [][]interface{}{}

			for _, tfTimeBlock := range tfpolicy.TimeBlocks {
				block := []interface{}{}
				block = append(block, tfTimeBlock.Cron.ValueString())
				block = append(block, tfTimeBlock.Interval.ValueInt64())

				timeBlocks = append(timeBlocks, block)
			}

			policy["time_blocks"] = timeBlocks
			aggregateThresholds, d := kpiThresholdThresholdSettingsAttributesToPayload(ctx, tfpolicy.AggregateThresholds)
			diags.Append(d...)
			if diags.HasError() {
				return
			}
			policy["aggregate_thresholds"] = aggregateThresholds

			entityThresholds, d := kpiThresholdThresholdSettingsAttributesToPayload(ctx, tfpolicy.EntityThresholds)
			diags.Append(d...)
			if diags.HasError() {
				return
			}
			policy["entity_thresholds"] = entityThresholds

			policies[tfpolicy.PolicyName.ValueString()] = policy
		}
		body["time_variate_thresholds_specification"] = map[string]interface{}{
			"policies": policies,
		}

	}

	base := kpiThresholdTemplateBase(clientConfig, tfKpiThresholdTemplate.ID.ValueString(), tfKpiThresholdTemplate.Title.ValueString())
	err := base.PopulateRawJSON(ctx, body)
	if err != nil {
		diags.AddError("Failed to populate kpi threshold template.", err.Error())
		return
	}
	return base, nil
}

func populateKpiThresholdTemplateModel(_ context.Context, b *models.Base, tfModelKpiThresholdTemplate *modelKpiThresholdTemplate) (diags diag.Diagnostics) {
	interfaceMap, err := b.RawJson.ToInterfaceMap()
	if err != nil {
		diags.AddError("Failed to populate interfaceMap.", err.Error())
	}
	diags = append(diags, marshalBasicTypesByTag("json", interfaceMap, tfModelKpiThresholdTemplate)...)

	tfPolicies := []PolicyModel{}

	timeVariateThresholdsSpecificationData := interfaceMap["time_variate_thresholds_specification"].(map[string]interface{})
	for policyName, pData := range timeVariateThresholdsSpecificationData["policies"].(map[string]interface{}) {
		policyData := pData.(map[string]interface{})

		tfPolicy := PolicyModel{
			PolicyName: types.StringValue(policyName),
			Title:      types.StringValue(policyData["title"].(string)),
			PolicyType: types.StringValue(policyData["policy_type"].(string)),
		}

		tfTimeBlocks := []TimeBlockModel{}
		for _, timeBlock := range policyData["time_blocks"].([]interface{}) {
			_timeBlock := timeBlock.([]interface{})
			tfTimeBlock := TimeBlockModel{
				Cron:     types.StringValue(_timeBlock[0].(string)),
				Interval: types.Int64Value(int64(_timeBlock[1].(float64))),
			}
			tfTimeBlocks = append(tfTimeBlocks, tfTimeBlock)
		}
		var diags_ diag.Diagnostics
		tfPolicy.TimeBlocks = tfTimeBlocks
		diags.Append(diags_...)
		tfAggregatedThresholds := ThresholdSettingModel{}
		diags.Append(kpiThresholdSettingsToModel("aggregate_thresholds",
			policyData["aggregate_thresholds"].(map[string]interface{}), &tfAggregatedThresholds, policyData["policy_type"].(string))...)

		tfPolicy.AggregateThresholds = tfAggregatedThresholds

		tfEntityThresholds := ThresholdSettingModel{}
		diags.Append(kpiThresholdSettingsToModel("entity_thresholds", policyData["entity_thresholds"].(map[string]interface{}),
			&tfEntityThresholds, policyData["policy_type"].(string))...)
		tfPolicy.EntityThresholds = tfEntityThresholds
		tfPolicies = append(tfPolicies, tfPolicy)
	}
	tfModelKpiThresholdTemplate.TimeVariateThresholdsSpecification = &TimeVariateThresholdsSpecificationModel{}
	tfModelKpiThresholdTemplate.TimeVariateThresholdsSpecification.Policies = tfPolicies

	tfModelKpiThresholdTemplate.ID = types.StringValue(b.RESTKey)
	return
}

func (r *resourceKpiThresholdTemplate) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx, cancel := context.WithTimeout(ctx, tftimeout.Read)
	defer cancel()

	b := kpiThresholdTemplateBase(r.client, "", req.ID)
	b, err := b.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to find Kpi Threshold Template model", err.Error())
		return
	}
	if b == nil {
		resp.Diagnostics.AddError("Kpi Threshold Template not found", fmt.Sprintf("Kpi Threshold Template '%s' not found", req.ID))
		return
	}

	state := modelKpiThresholdTemplate{}
	if resp.Diagnostics.Append(populateKpiThresholdTemplateModel(ctx, b, &state)...); resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
