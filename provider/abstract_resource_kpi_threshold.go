package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ThresholdSettingModel struct {
	BaseSeverityLabel types.String             `json:"baseSeverityLabel" tfsdk:"base_severity_label"`
	GaugeMax          types.Float64            `json:"gaugeMax" tfsdk:"gauge_max"`
	GaugeMin          types.Float64            `json:"gaugeMin" tfsdk:"gauge_min"`
	IsMaxStatic       types.Bool               `json:"isMaxStatic" tfsdk:"is_max_static"`
	IsMinStatic       types.Bool               `json:"isMinStatic" tfsdk:"is_min_static"`
	MetricField       types.String             `json:"metricField" tfsdk:"metric_field"`
	RenderBoundaryMax types.Float64            `json:"renderBoundaryMax" tfsdk:"render_boundary_max"`
	RenderBoundaryMin types.Float64            `json:"renderBoundaryMin" tfsdk:"render_boundary_min"`
	Search            types.String             `json:"search" tfsdk:"search"`
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
				Validators: []validator.String{
					stringvalidator.OneOf("info", "critical", "high", "medium", "low", "normal"),
				},
				Description: "Base severity label assigned for the threshold (info, normal, low, medium, high, critical). ",
			},
			"gauge_max": schema.Float64Attribute{
				Optional:    true,
				Description: "Maximum value for the threshold gauge specified by user",
			},
			"gauge_min": schema.Float64Attribute{
				Optional:    true,
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
				Description: "Thresholding field from the search.",
			},
			"render_boundary_max": schema.Float64Attribute{
				Optional:    true,
				Description: "Upper bound value to use to render the graph for the thresholds.",
			},
			"render_boundary_min": schema.Float64Attribute{
				Optional:    true,
				Description: "Lower bound value to use to render the graph for the thresholds.",
			},
			"search": schema.StringAttribute{
				Optional:    true,
				Description: "Generated search used to compute the thresholds for this KPI.",
			},
		}
}

func kpiThresholdSettingsToModel(attrName string, apiThresholdSetting map[string]interface{}, tfthresholdSettingModel *ThresholdSettingModel, settingType string) (diags diag.Diagnostics) {
	marshalBasicTypesByTag("json", apiThresholdSetting, tfthresholdSettingModel)

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
		marshalBasicTypesByTag("json", tData, thresholdLevel)
		thresholdLevels = append(thresholdLevels, *thresholdLevel)
	}
	var diags_ diag.Diagnostics
	tfthresholdSettingModel.ThresholdLevels = thresholdLevels
	diags.Append(diags_...)
	return diags
}

func kpiThresholdThresholdSettingsAttributesToPayload(ctx context.Context, source ThresholdSettingModel) (interface{}, diag.Diagnostics) {
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
