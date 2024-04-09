package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
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

type TimeBlockModel struct {
	Interval types.Int64  `tfsdk:"interval"`
	Cron     types.String `tfsdk:"cron"`
}
type resourceKpiThresholdTemplate struct {
	client models.ClientConfig
}

func (r *resourceKpiThresholdTemplate) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data modelKpiThresholdTemplate
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	/*for _, policy := range data.TimeVariateThresholdsSpecification.Policies {
		for _, level := range append(policy.AggregateThresholds.ThresholdLevels, policy.EntityThresholds.ThresholdLevels...) {
			var isStaticPolicy = policy.PolicyType.ValueString() != "static"
			if level.DynamicParam.IsNull() && isStaticPolicy {
				resp.Diagnostics.AddError(
					"Missing Attribute Configuration",
					"Expected dynamic_param in case of the non-static thresholds (ex: stdev).",
				)
				return
			} else if !level.DynamicParam.IsNull() && !isStaticPolicy {
				resp.Diagnostics.AddError(
					"Wrong Attribute Configuration",
					"Not expect dynamic_param in case of the static thresholds.",
				)
				return
			}
		}
	}*/
}

// Metadata returns the resource type name.
func (r *resourceKpiThresholdTemplate) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "itsi_kpi_threshold_template"
}

func (r *resourceKpiThresholdTemplate) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *resourceKpiThresholdTemplate) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	threshold_settings_blocks, threshold_settings_attributes := getKpiThresholdSettingsBlocksAttrs()

	resp.Schema = schema.Schema{
		Blocks: map[string]schema.Block{
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
				Description: "User defined description of the entity.",
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
				Required:    true,
				Description: "The team the object belongs to. ",
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
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
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

func populateKpiThresholdTemplateModel(ctx context.Context, b *models.Base, tfModelKpiThresholdTemplate *modelKpiThresholdTemplate) (diags diag.Diagnostics) {
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
	resp.Diagnostics.Append(populateKpiThresholdTemplateModel(ctx, b, &state)...)
	// if resp.Diagnostics.Append(diags...); diags.HasError() {
	// 	return
	// }

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
