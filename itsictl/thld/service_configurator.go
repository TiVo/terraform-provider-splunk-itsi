package thld

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider"
	"github.com/tivo/terraform-provider-splunk-itsi/util"
	"gopkg.in/yaml.v3"
)

type serviceThresholdingConfigurator struct {
	service *models.ItsiObj

	serviceID    string
	serviceTitle string

	svcMap map[string]any

	policies            policiesByKpi
	kpis                []map[string]any
	kpisToConfigure     []map[string]any
	trainingConfigByKPI map[kpiID]trainingConfig

	workflow *ThresholdRecommendationWorkflow

	changeSummary map[string]string
	failedKpis    util.Set[string]
}

func (c *serviceThresholdingConfigurator) parseThresholds(thldsStr string) map[string][]float64 {
	thldsStr = strings.Replace(thldsStr, "'", `"`, -1) //this is why we can't have nice things

	thresholds := map[string]any{}
	normalizedThresholds := map[string][]float64{}

	err := json.Unmarshal([]byte(thldsStr), &thresholds)
	if err != nil {
		c.workflow.Log.Fatal("Failed to parse thresholds JSON", "thresholds", thldsStr)
	}

	for k, v := range thresholds {
		if singleThreshold, ok := v.(float64); ok {
			normalizedThresholds[k] = []float64{singleThreshold}
		} else {
			v_, err := provider.UnpackSlice[float64](v)
			if err != nil {
				c.workflow.Log.Fatal("Failed to parse thresholds JSON", "thresholds", thldsStr)
			}
			normalizedThresholds[k] = v_
		}

	}

	return normalizedThresholds
}

func (c *serviceThresholdingConfigurator) recordKPIUpdateSummary(kpiKey, kpiTitle, summary string) {
	c.changeSummary[fmt.Sprintf("%s (%s)", kpiTitle, kpiKey)] = summary
}

func (c *serviceThresholdingConfigurator) kpiConfError(kpiID, kpiTitle string, err error) error {
	return fmt.Errorf("[%s/%s] (%s/%s) %w", c.serviceTitle, kpiTitle, c.serviceID, kpiID, err)
}

func (c *serviceThresholdingConfigurator) configureKPI(kpi map[string]any) (err error) {
	newKpiConfig := map[string]any{}
	details := []string{}

	kpiKey, kpiTitle := kpi["_key"].(string), kpi["title"].(string)
	kpiPolicies := c.policies[kpiKey]

	timeVariateThresholdsEnabled := false
	adaptiveThresholdsEnabled := false

	isConstantKPI := false
	isInsufficientData := false

	var sensitivity float64

	timeVariateThresholdPolicies := c.workflow.defaultPolicies()

	if len(kpiPolicies) > 0 {
		policiesYaml, err := yaml.Marshal(kpiPolicies)
		if err != nil {
			c.workflow.Log.Fatal("Failed to marshal kpiPolicies", "kpiPolicies", kpiPolicies)
		}

		c.workflow.Log.Debug(
			fmt.Sprintf("Updating thresholding configuration for the [%s / %s] KPI", c.serviceTitle, kpiTitle),
			"service_id", c.serviceID,
			"kpi_id", kpiKey,
			"policies", string(policiesYaml),
		)
	} else {
		isInsufficientData = true
	}

	for _, kpiPolicy := range kpiPolicies {
		recommendationFlag, ok := kpiPolicy["Recommendation Flag"]
		if !ok {
			return c.kpiConfError(kpiKey, kpiTitle, fmt.Errorf("unexpected KPI policy (%#v): Recommendation Flag was not found", kpiPolicy))
		}

		algo, ok := kpiPolicy["Algorithm"].(string)
		if ok {
			algo = strings.ToLower(algo)
		} else {
			return c.kpiConfError(kpiKey, kpiTitle, fmt.Errorf("unexpected KPI policy (%#v): Alogrithm is not provided", kpiPolicy))
		}

		//ATM, we only support stdev and static policies
		switch algo {
		case "stdev":
			adaptiveThresholdsEnabled = true
		case "static":
		case "none":
			if recommendationFlag == "CONSTANT_KPI" && len(kpiPolicies) == 1 {
				isConstantKPI = true
				continue
			}
			fallthrough
		default:
			return c.kpiConfError(kpiKey, kpiTitle, fmt.Errorf("%s (%s) recommendation is not supported yet", algo, recommendationFlag))
		}

		cron, ok := kpiPolicy["Cron Expression"].(string)
		if !ok {
			return c.kpiConfError(kpiKey, kpiTitle, fmt.Errorf("cron expression is missing"))
		}

		//When `cron` field is None, we presume the policy is not "time variate", and should be used
		//to populate KPI's global aggregate threshold levels
		isTimeVariatePolicy := !(cron == "None")
		timeVariateThresholdsEnabled = timeVariateThresholdsEnabled || isTimeVariatePolicy

		var mean, std float64
		if mean, err = strconv.ParseFloat(kpiPolicy["Mean"].(string), 64); err != nil {
			return c.kpiConfError(kpiKey, kpiTitle, fmt.Errorf("could not parse Mean"))
		}
		if std, err = strconv.ParseFloat(kpiPolicy["Std"].(string), 64); err != nil {
			return c.kpiConfError(kpiKey, kpiTitle, fmt.Errorf("could not parse Std"))
		}

		if adaptiveThresholdsEnabled {
			if sensitivity, err = strconv.ParseFloat(kpiPolicy["Sensitivity"].(string), 64); err != nil {
				return c.kpiConfError(kpiKey, kpiTitle, fmt.Errorf("could not parse Sensitivity"))
			}
		}

		thresholds := c.parseThresholds(kpiPolicy["Thresholds"].(string))

		aggregateThresholdLevels := []map[string]any{}

		for severity, dynamicParams := range thresholds {
			for _, dynamicParam := range dynamicParams {
				thldValue := 0.0
				if algo == "stdev" {
					thldValue = mean + std*dynamicParam
				} else {
					//static
					thldValue = dynamicParam
				}

				itsiThldLevel := map[string]any{
					"severityLabel":      severity,
					"severityValue":      util.SeverityMap[severity].SeverityValue,
					"severityColor":      util.SeverityMap[severity].SeverityColor,
					"severityColorLight": util.SeverityMap[severity].SeverityColorLight,
					"thresholdValue":     thldValue,
					"dynamicParam":       dynamicParam,
				}
				aggregateThresholdLevels = append(aggregateThresholdLevels, itsiThldLevel)
			}
		}

		if isTimeVariatePolicy {
			var duration float64
			if duration, err = strconv.ParseFloat(kpiPolicy["Duration"].(string), 64); err != nil {
				return c.kpiConfError(kpiKey, kpiTitle, fmt.Errorf("could not parse Duration"))
			}

			title := fmt.Sprintf("[%s] %v", cron, duration) //TODO: improve policy title generation
			key := util.Sha256(title)
			itsiPolicy := map[string]any{
				"title":                title,
				"aggregate_thresholds": c.workflow.aggregateThresholds(aggregateThresholdLevels),
				"entity_thresholds":    c.workflow.entityThresholds(),
				"policy_type":          "stdev",
				"time_blocks":          []any{[]any{cron, duration}},
			}
			timeVariateThresholdPolicies[key] = itsiPolicy
		} else if len(kpiPolicies) == 1 {
			newKpiConfig["aggregate_thresholds"] = c.workflow.aggregateThresholds(aggregateThresholdLevels)
			break
		} else {
			return c.kpiConfError(kpiKey, kpiTitle, fmt.Errorf("unexpected KPI policy recommendations"))
		}

	}

	newKpiConfig["time_variate_thresholds"] = timeVariateThresholdsEnabled
	newKpiConfig["adaptive_thresholds_is_enabled"] = adaptiveThresholdsEnabled

	if adaptiveThresholdsEnabled {
		newKpiConfig["adaptive_thresholding_training_window"] = fmt.Sprintf("-%dd", c.trainingConfigByKPI[kpiID{c.service, kpiKey}].size)
		newKpiConfig["aggregate_outlier_detection_enabled"] = true
		newKpiConfig["outlier_detection_algo"] = "iqr"
		newKpiConfig["outlier_detection_sensitivity"] = sensitivity
	}
	if timeVariateThresholdsEnabled {
		newKpiConfig["time_variate_thresholds_specification"] = map[string]any{"policies": timeVariateThresholdPolicies}
	}

	ok := !(isConstantKPI || isInsufficientData)

	// Update thresholding configuration
	if ok {
		// set thresholds based on the ML-recommended policies
		c.workflow.resetThresholding(kpi)
		maps.Copy(kpi, newKpiConfig)

	} else {
		// ML analysis did not produce policies due to insufficient data
		// or KPI value being constant.
		// We will skip the KPI or reset its thresholding config
		// depending on the `insufficientDataAction` parameter
		c.failedKpis.Add(kpiKey)
		if c.workflow.insufficientDataAction == "reset" {
			c.workflow.resetThresholding(kpi)
		}
	}

	// Record update summary
	if ok {

		if timeVariateThresholdsEnabled {
			details = append(details, "time variate")
		}
		if adaptiveThresholdsEnabled {
			details = append(details, "adaptive (stdev)")
		} else {
			if isConstantKPI {
				details = append(details, "None (constant KPI)")
			} else {
				details = append(details, "static")
			}
		}

	} else {
		if isConstantKPI {
			details = append(details, "None (constant KPI)")
		} else {
			details = append(details, "None (insufficient data / no policies generated)")
		}
	}

	c.recordKPIUpdateSummary(kpiKey, kpiTitle, strings.Join(details, ", "))
	return nil
}

func (c *serviceThresholdingConfigurator) Configure() error {
	var err error
	for _, kpi := range c.kpisToConfigure {
		if err = c.configureKPI(kpi); err != nil {
			return err
		}
	}

	c.svcMap["kpis"] = c.kpis
	if err := c.service.PopulateRawJSON(context.TODO(), c.svcMap); err != nil {
		return fmt.Errorf("[%s] (%s): failed to populate service api model", c.serviceTitle, c.serviceID)
	}

	return nil
}
