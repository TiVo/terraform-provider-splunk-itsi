package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	schemav2 "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	validationv2 "github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// type ThresholdSettingModelv2 struct {
// 	BaseSeverityLabel types.String              `json:"baseSeverityLabel" tfsdk:"base_severity_label"`
// 	GaugeMax          types.Float64             `json:"gaugeMax" tfsdk:"gauge_max"`
// 	GaugeMin          types.Float64             `json:"gaugeMin" tfsdk:"gauge_min"`
// 	IsMaxStatic       types.Bool                `json:"isMaxStatic" tfsdk:"is_max_static"`
// 	IsMinStatic       types.Bool                `json:"isMinStatic" tfsdk:"is_min_static"`
// 	MetricField       types.String              `json:"metricField" tfsdk:"metric_field"`
// 	RenderBoundaryMax types.Float64             `json:"renderBoundaryMax" tfsdk:"render_boundary_max"`
// 	RenderBoundaryMin types.Float64             `json:"renderBoundaryMin" tfsdk:"render_boundary_min"`
// 	Search            types.String              `json:"search" tfsdk:"search"`
// 	ThresholdLevels   []*KpiThresholdLevelModel `tfsdk:"threshold_levels"`
// }

type ThresholdSettingModel struct {
	BaseSeverityLabel types.String  `tfsdk:"base_severity_label"`
	GaugeMax          types.Float64 `tfsdk:"gauge_max"`
	GaugeMin          types.Float64 `tfsdk:"gauge_min"`
	IsMaxStatic       types.Bool    `tfsdk:"is_max_static"`
	IsMinStatic       types.Bool    `tfsdk:"is_min_static"`
	MetricField       types.String  `tfsdk:"metric_field"`
	RenderBoundaryMax types.Float64 `tfsdk:"render_boundary_max"`
	RenderBoundaryMin types.Float64 `tfsdk:"render_boundary_min"`
	Search            types.String  `tfsdk:"search"`
	ThresholdLevels   types.Set     `tfsdk:"threshold_levels"`
}

type KpiThresholdLevelModel struct {
	SeverityLabel  types.String  `tfsdk:"severity_label"`
	ThresholdValue types.Float64 `tfsdk:"threshold_value"`
	DynamicParam   types.Float64 `tfsdk:"dynamic_param"`
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
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf("info", "critical", "high", "medium", "low", "normal"),
				},
				Description: "Base severity label assigned for the threshold (info, normal, low, medium, high, critical). ",
			},
			"gauge_max": schema.Float64Attribute{
				Required:    true,
				Description: "Maximum value for the threshold gauge specified by user",
			},
			"gauge_min": schema.Float64Attribute{
				Required:    true,
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
				Required:    true,
				Description: "Thresholding field from the search.",
			},
			"render_boundary_max": schema.Float64Attribute{
				Required:    true,
				Description: "Upper bound value to use to render the graph for the thresholds.",
			},
			"render_boundary_min": schema.Float64Attribute{
				Required:    true,
				Description: "Lower bound value to use to render the graph for the thresholds.",
			},
			"search": schema.StringAttribute{
				Required:    true,
				Description: "Generated search used to compute the thresholds for this KPI.",
			},
		}
}

func getKpiThresholdSettingsSchema() map[string]*schemav2.Schema {
	kpiThresholdLevel := map[string]*schemav2.Schema{
		"severity_label": {
			Type:         schemav2.TypeString,
			Required:     true,
			ValidateFunc: validationv2.StringInSlice([]string{"info", "critical", "high", "medium", "low", "normal"}, false),
			Description:  "Severity label assigned for this threshold level like info, warning, critical, etc",
		},
		"threshold_value": {
			Type:     schemav2.TypeFloat,
			Required: true,
			Description: `Value for the threshold field stats identifying this threshold level.
				This is the key value that defines the levels for values derived from the KPI search metrics.`,
		},
		"dynamic_param": {
			Type:        schemav2.TypeFloat,
			Computed:    true,
			Optional:    true,
			Description: "Value of the dynamic parameter for adaptive thresholds",
		},
	}

	return map[string]*schemav2.Schema{
		"base_severity_label": {
			Type:         schemav2.TypeString,
			Optional:     true,
			Default:      "normal",
			ValidateFunc: validationv2.StringInSlice([]string{"info", "critical", "high", "medium", "low", "normal"}, false),
			Description:  "Base severity label assigned for the threshold (info, normal, low, medium, high, critical). ",
		},
		"gauge_max": {
			Type:        schemav2.TypeFloat,
			Optional:    true,
			Description: "Maximum value for the threshold gauge specified by user",
		},
		"gauge_min": {
			Type:        schemav2.TypeFloat,
			Optional:    true,
			Description: "Minimum value for the threshold gauge specified by user.",
		},
		"is_max_static": {
			Type:        schemav2.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "True when maximum threshold value is a static value, false otherwise. ",
		},
		"is_min_static": {
			Type:        schemav2.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "True when min threshold value is a static value, false otherwise.",
		},
		"metric_field": {
			Type:        schemav2.TypeString,
			Optional:    true,
			Default:     "",
			Description: "Thresholding field from the search.",
		},
		"render_boundary_max": {
			Type:        schemav2.TypeFloat,
			Required:    true,
			Description: "Upper bound value to use to render the graph for the thresholds.",
		},
		"render_boundary_min": {
			Type:        schemav2.TypeFloat,
			Required:    true,
			Description: "Lower bound value to use to render the graph for the thresholds.",
		},
		"search": {
			Type:        schemav2.TypeString,
			Optional:    true,
			Default:     "",
			Description: "Generated search used to compute the thresholds for this KPI.",
		},
		"threshold_levels": {
			Type:     schemav2.TypeSet,
			Optional: true,
			Elem: &schemav2.Resource{
				Schema: kpiThresholdLevel,
			},
		},
	}
}

func kpiThresholdSettingsToResourceData(sourceThresholdSetting map[string]interface{}, settingType string) (interface{}, error) {
	thresholdSetting := map[string]interface{}{}
	thresholdSetting["base_severity_label"] = sourceThresholdSetting["baseSeverityLabel"]
	thresholdSetting["gauge_max"] = sourceThresholdSetting["gaugeMax"]
	thresholdSetting["gauge_min"] = sourceThresholdSetting["gaugeMin"]
	thresholdSetting["is_max_static"] = sourceThresholdSetting["isMaxStatic"]
	thresholdSetting["is_min_static"] = sourceThresholdSetting["isMinStatic"]
	thresholdSetting["metric_field"] = sourceThresholdSetting["metricField"]
	thresholdSetting["render_boundary_max"] = sourceThresholdSetting["renderBoundaryMax"]
	thresholdSetting["render_boundary_min"] = sourceThresholdSetting["renderBoundaryMin"]
	thresholdSetting["search"] = sourceThresholdSetting["search"]
	thresholdLevels := []interface{}{}
	for _, tData_ := range sourceThresholdSetting["thresholdLevels"].([]interface{}) {
		tData := tData_.(map[string]interface{})
		thresholdLevel := map[string]interface{}{}
		switch tData["dynamicParam"] {
		case "":
			if settingType != "static" {
				return nil, fmt.Errorf("empty dynamic param for adaptive policy %s", settingType)
			}
			thresholdLevel["dynamic_param"] = 0
		default:
			thresholdLevel["dynamic_param"] = tData["dynamicParam"]
		}

		thresholdLevel["severity_label"] = tData["severityLabel"]
		thresholdLevel["threshold_value"] = tData["thresholdValue"]
		thresholdLevels = append(thresholdLevels, thresholdLevel)
	}
	thresholdSetting["threshold_levels"] = thresholdLevels
	return []interface{}{thresholdSetting}, nil
}

func kpiThresholdSettingsToModel(ctx context.Context, attrName string, policyType basetypes.ObjectType, apiThresholdSetting map[string]interface{}, tfthresholdSettingModel *ThresholdSettingModel, settingType string) (diags diag.Diagnostics) {
	tfthresholdSettingModel.BaseSeverityLabel = types.StringValue(apiThresholdSetting["baseSeverityLabel"].(string))

	tfthresholdSettingModel.GaugeMin = types.Float64Value(apiThresholdSetting["gaugeMin"].(float64))
	tfthresholdSettingModel.GaugeMax = types.Float64Value(apiThresholdSetting["gaugeMax"].(float64))

	tfthresholdSettingModel.IsMinStatic = types.BoolValue(apiThresholdSetting["isMinStatic"].(bool))
	tfthresholdSettingModel.IsMaxStatic = types.BoolValue(apiThresholdSetting["isMaxStatic"].(bool))

	tfthresholdSettingModel.MetricField = types.StringValue(apiThresholdSetting["metricField"].(string))

	tfthresholdSettingModel.RenderBoundaryMin = types.Float64Value(apiThresholdSetting["renderBoundaryMin"].(float64))
	tfthresholdSettingModel.RenderBoundaryMax = types.Float64Value(apiThresholdSetting["renderBoundaryMax"].(float64))

	tfthresholdSettingModel.Search = types.StringValue(apiThresholdSetting["search"].(string))

	thresholdLevels := []KpiThresholdLevelModel{}
	for _, tData_ := range apiThresholdSetting["thresholdLevels"].([]interface{}) {
		tData := tData_.(map[string]interface{})
		thresholdLevel := KpiThresholdLevelModel{}
		switch tData["dynamicParam"] {
		case "":
			if settingType != "static" {
				diags.AddError("Failed to populate aggregated threshold", fmt.Sprintf("empty dynamic param for adaptive policy %s", settingType))
				return
			}
			thresholdLevel.DynamicParam = types.Float64Value(0)
		default:
			thresholdLevel.DynamicParam = types.Float64Value(tData["dynamicParam"].(float64))
		}

		thresholdLevel.SeverityLabel = types.StringValue(tData["severityLabel"].(string))
		thresholdLevel.ThresholdValue = types.Float64Value(tData["thresholdValue"].(float64))
		thresholdLevels = append(thresholdLevels, thresholdLevel)
	}
	var diags_ diag.Diagnostics
	tfthresholdSettingModel.ThresholdLevels, diags_ = types.SetValueFrom(ctx, policyType.AttrTypes[attrName].(basetypes.ObjectType).AttrTypes["threshold_levels"].(basetypes.SetType).ElemType, thresholdLevels)
	diags.Append(diags_...)
	return diags
}

func kpiThresholdThresholdSettingsAttributesToPayload(ctx context.Context, source ThresholdSettingModel) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics
	thresholdSetting := map[string]interface{}{}
	if severity, ok := util.SeverityMap[source.BaseSeverityLabel.ValueString()]; ok {
		thresholdSetting["baseSeverityColor"] = severity.SeverityColor
		thresholdSetting["baseSeverityColorLight"] = severity.SeverityColorLight
		thresholdSetting["baseSeverityLabel"] = severity.SeverityLabel
		thresholdSetting["baseSeverityValue"] = severity.SeverityValue
	} else {
		diags.AddError("failed to convert threshold setting model to payload", fmt.Sprintf("schema Validation broken. Unknown severity %s", source.BaseSeverityLabel.ValueString()))
		return nil, diags
	}

	thresholdSetting["gaugeMax"] = source.GaugeMax.ValueFloat64()
	thresholdSetting["gaugeMin"] = source.GaugeMin.ValueFloat64()
	thresholdSetting["isMaxStatic"] = source.IsMaxStatic.ValueBool()
	thresholdSetting["isMinStatic"] = source.IsMinStatic.ValueBool()
	thresholdSetting["metricField"] = source.MetricField.ValueString()
	thresholdSetting["renderBoundaryMax"] = source.RenderBoundaryMax.ValueFloat64()
	thresholdSetting["renderBoundaryMin"] = source.RenderBoundaryMin.ValueFloat64()
	thresholdSetting["search"] = source.Search.ValueString()
	thresholdLevels := []interface{}{}
	var thresholdLevelsModels []KpiThresholdLevelModel
	if diags.Append(source.ThresholdLevels.ElementsAs(ctx, &thresholdLevelsModels, false)...); diags.HasError() {
		return nil, diags
	}
	for _, tfThresholdLevel := range thresholdLevelsModels {
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

// TODO: remove once service resource is migrated to the new terraform provider framwork
func kpiThresholdThresholdSettingsToPayload(source map[string]interface{}) (interface{}, error) {
	thresholdSetting := map[string]interface{}{}
	if severity, ok := util.SeverityMap[source["base_severity_label"].(string)]; ok {
		thresholdSetting["baseSeverityColor"] = severity.SeverityColor
		thresholdSetting["baseSeverityColorLight"] = severity.SeverityColorLight
		thresholdSetting["baseSeverityLabel"] = severity.SeverityLabel
		thresholdSetting["baseSeverityValue"] = severity.SeverityValue
	} else {
		return nil, fmt.Errorf("schema Validation broken. Unknown severity %s", source["base_severity_label"])
	}
	thresholdSetting["gaugeMax"] = source["gauge_max"].(float64)
	thresholdSetting["gaugeMin"] = source["gauge_min"].(float64)
	thresholdSetting["isMaxStatic"] = source["is_max_static"].(bool)
	thresholdSetting["isMinStatic"] = source["is_min_static"].(bool)
	thresholdSetting["metricField"] = source["metric_field"].(string)
	thresholdSetting["renderBoundaryMax"] = source["render_boundary_max"].(float64)
	thresholdSetting["renderBoundaryMin"] = source["render_boundary_min"].(float64)
	thresholdSetting["search"] = source["search"].(string)
	thresholdLevels := []interface{}{}
	for _, sourceThresholdLevel_ := range source["threshold_levels"].(*schemav2.Set).List() {
		sourceThresholdLevel := sourceThresholdLevel_.(map[string]interface{})
		thresholdLevel := map[string]interface{}{}
		thresholdLevel["dynamicParam"] = sourceThresholdLevel["dynamic_param"].(float64)
		if severity, ok := util.SeverityMap[sourceThresholdLevel["severity_label"].(string)]; ok {
			thresholdLevel["severityColor"] = severity.SeverityColor
			thresholdLevel["severityColorLight"] = severity.SeverityColorLight
			thresholdLevel["severityLabel"] = severity.SeverityLabel
			thresholdLevel["severityValue"] = severity.SeverityValue
		} else {
			return nil, fmt.Errorf("schema Validation broken. Unknown severity %s", sourceThresholdLevel["severity_label"])
		}
		thresholdLevel["thresholdValue"] = sourceThresholdLevel["threshold_value"].(float64)
		thresholdLevels = append(thresholdLevels, thresholdLevel)
	}
	thresholdSetting["thresholdLevels"] = thresholdLevels
	return thresholdSetting, nil
}
