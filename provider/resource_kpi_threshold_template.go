package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

// TODO: uncomment once scrapper will use framework schema reflect approach
/*func kpiThresholdTemplateTFFormat(b *models.Base) (string, error) {
	res := ResourceKPIThresholdTemplate()
	resData := res.Data(nil)
	d := populateKpiThresholdTemplateResourceData(context.Background(), b, resData)
	if len(d) > 0 {
		err := d[0].Validate()
		if err != nil {
			return "", err
		}
		return "", errors.New(d[0].Summary)
	}
	resourcetpl, err := NewResourceTemplate(resData, res.Schema, "title", "itsi_kpi_threshold_template")
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
}*/

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
	ID                                 types.String                            `tfsdk:"id"`
	Title                              types.String                            `tfsdk:"title"`
	Description                        types.String                            `tfsdk:"description"`
	AdaptiveThresholdingTrainingWindow types.String                            `tfsdk:"adaptive_thresholding_training_window"`
	TimeVariateThresholds              types.Bool                              `tfsdk:"time_variate_thresholds"`
	TimeVariateThresholdsSpecification TimeVariateThresholdsSpecificationModel `tfsdk:"time_variate_thresholds_specification"`
	AdaptiveThresholdsIsEnabled        types.Bool                              `tfsdk:"adaptive_thresholds_is_enabled"`
	SecGrp                             types.String                            `tfsdk:"sec_grp"`
}

/*
(:̲̅:̲̅:̲̅[̲̅ ]̲̅:̲̅:̲̅:̲̅) Bandaid for a terraform bug introduced in terraform 1.14 and still present in 1.15.5,
which for some reason every other time reverts the Optional Fields of the KPI threshold template resource to their default values.
That is the reason why we have to set the default values here, and not in the resource schema definition.
TODO: move the default values to the KPI threshold template resource schema once the bug is fixed.
*/
func (m *modelKpiThresholdTemplate) SetDefaults(ctx context.Context) (diags diag.Diagnostics) {
	policies := []PolicyModel{}
	if diags.Append(m.TimeVariateThresholdsSpecification.Policies.ElementsAs(ctx, &policies, false)...); diags.HasError() {
		return
	}
	for i := range policies {
		for _, t := range []*ThresholdSettingModel{&policies[i].AggregateThresholds, &policies[i].EntityThresholds} {
			levels := []KpiThresholdLevelModel{}
			if diags.Append(t.ThresholdLevels.ElementsAs(ctx, &levels, false)...); diags.HasError() {
				return
			}

			for i := range levels {
				if levels[i].DynamicParam.IsUnknown() {
					levels[i].DynamicParam = types.Float64Value(0.0)
				}
			}
			t.ThresholdLevels, diags = types.SetValueFrom(ctx, t.ThresholdLevels.ElementType(ctx), levels)

			if t.BaseSeverityLabel.IsUnknown() {
				t.BaseSeverityLabel = types.StringValue("normal")
			}
			if t.IsMaxStatic.IsUnknown() {
				t.IsMaxStatic = types.BoolValue(false)
			}
			if t.IsMinStatic.IsUnknown() {
				t.IsMinStatic = types.BoolValue(false)
			}
			if t.MetricField.IsUnknown() {
				t.MetricField = types.StringValue("")
			}
			if t.Search.IsUnknown() {
				t.Search = types.StringValue("")
			}
		}
	}
	m.TimeVariateThresholdsSpecification.Policies, diags = types.SetValueFrom(ctx,
		m.TimeVariateThresholdsSpecification.Policies.ElementType(ctx),
		policies)

	return
}

type TimeVariateThresholdsSpecificationModel struct {
	Policies types.Set `tfsdk:"policies"`
}

type PolicyModel struct {
	PolicyName          types.String          `tfsdk:"policy_name"`
	Title               types.String          `tfsdk:"title"`
	PolicyType          types.String          `tfsdk:"policy_type"`
	TimeBlocks          types.Set             `tfsdk:"time_blocks"`
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
	if resp.Diagnostics.Append(plan.SetDefaults(ctx)...); resp.Diagnostics.HasError() {
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

	if resp.Diagnostics.Append(plan.SetDefaults(ctx)...); resp.Diagnostics.HasError() {
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
	body["objectType"] = "kpi_threshold_template"
	body["title"] = tfKpiThresholdTemplate.Title.ValueString()
	body["description"] = tfKpiThresholdTemplate.Description.ValueString()
	body["adaptive_thresholds_is_enabled"] = tfKpiThresholdTemplate.AdaptiveThresholdsIsEnabled.ValueBool()
	body["adaptive_thresholding_training_window"] = tfKpiThresholdTemplate.AdaptiveThresholdingTrainingWindow.ValueString()
	body["time_variate_thresholds"] = tfKpiThresholdTemplate.TimeVariateThresholds.ValueBool()
	body["sec_grp"] = tfKpiThresholdTemplate.SecGrp.ValueString()

	policies := map[string]interface{}{}
	var policieModels []PolicyModel
	if diags.Append(tfKpiThresholdTemplate.TimeVariateThresholdsSpecification.Policies.ElementsAs(ctx, &policieModels, false)...); diags.HasError() {
		return nil, diags
	}
	for _, tfpolicy := range policieModels {
		policy := map[string]interface{}{}
		policy["title"] = tfpolicy.Title.ValueString()
		policy["policy_type"] = tfpolicy.PolicyType.ValueString()
		timeBlocks := [][]interface{}{}
		var timeBlockModels []TimeBlockModel
		if diags.Append(tfpolicy.TimeBlocks.ElementsAs(ctx, &timeBlockModels, false)...); diags.HasError() {
			return nil, diags
		}
		for _, tfTimeBlock := range timeBlockModels {
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

	tfModelKpiThresholdTemplate.Title = types.StringValue(interfaceMap["title"].(string))
	tfModelKpiThresholdTemplate.Description = types.StringValue(interfaceMap["description"].(string))
	tfModelKpiThresholdTemplate.AdaptiveThresholdingTrainingWindow = types.StringValue(interfaceMap["adaptive_thresholding_training_window"].(string))
	tfModelKpiThresholdTemplate.AdaptiveThresholdsIsEnabled = types.BoolValue(interfaceMap["adaptive_thresholds_is_enabled"].(bool))
	tfModelKpiThresholdTemplate.TimeVariateThresholds = types.BoolValue(interfaceMap["time_variate_thresholds"].(bool))
	tfModelKpiThresholdTemplate.SecGrp = types.StringValue(interfaceMap["sec_grp"].(string))

	tfPolicies := []PolicyModel{}
	policySetValue, _diags := tfModelKpiThresholdTemplate.TimeVariateThresholdsSpecification.Policies.ToSetValue(ctx)
	diags = append(diags, _diags...)
	policyObjectType := policySetValue.ElementType(ctx).(basetypes.ObjectType)

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
		timeBlocksElementType := policyObjectType.AttrTypes["time_blocks"].(basetypes.SetType).ElemType
		tfPolicy.TimeBlocks, diags_ = types.SetValueFrom(ctx, timeBlocksElementType, tfTimeBlocks)
		diags.Append(diags_...)
		tfAggregatedThresholds := ThresholdSettingModel{}
		diags.Append(kpiThresholdSettingsToModel(ctx, "aggregate_thresholds", policyObjectType,
			policyData["aggregate_thresholds"].(map[string]interface{}), &tfAggregatedThresholds, policyData["policy_type"].(string))...)

		tfPolicy.AggregateThresholds = tfAggregatedThresholds

		tfEntityThresholds := ThresholdSettingModel{}
		diags.Append(kpiThresholdSettingsToModel(ctx, "entity_thresholds", policyObjectType,
			policyData["entity_thresholds"].(map[string]interface{}), &tfEntityThresholds, policyData["policy_type"].(string))...)
		tfPolicy.EntityThresholds = tfEntityThresholds
		tfPolicies = append(tfPolicies, tfPolicy)
	}
	tfModelKpiThresholdTemplate.TimeVariateThresholdsSpecification = TimeVariateThresholdsSpecificationModel{}
	var diags_ diag.Diagnostics
	tfModelKpiThresholdTemplate.TimeVariateThresholdsSpecification.Policies, diags_ = types.SetValueFrom(ctx, policySetValue.ElementType(ctx), tfPolicies)
	diags.Append(diags_...)

	tfModelKpiThresholdTemplate.ID = types.StringValue(b.RESTKey)
	return
}

func (r *resourceKpiThresholdTemplate) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	diags := resp.Diagnostics
	b := kpiThresholdTemplateBase(r.client, "", req.ID)
	b, err := b.Find(ctx)
	if err != nil || b == nil {
		diags.AddError("Failed to find kpi threshold template.", err.Error())
		resp.Diagnostics.Append(diags...)
		return
	}

	req.ID = b.RESTKey

	resp.State.SetAttribute(ctx, path.Root("time_variate_thresholds_specification"), TimeVariateThresholdsSpecificationModel{})
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
