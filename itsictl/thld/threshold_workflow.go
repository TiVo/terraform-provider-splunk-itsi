package thld

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"regexp"
	"slices"

	"github.com/tivo/terraform-provider-splunk-itsi/itsictl/config"
	"github.com/tivo/terraform-provider-splunk-itsi/itsictl/core"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/util"
)

type thresholdWorkflow struct {
	core.Workflow

	/*
		List of service selectors.
		Each selector can be either a service ID or a service name wildcard expression (case insenstive).
	*/
	services []string

	/*
		List of KPI selectors.
		Each selector can be either a KPI ID or a KPI name wildcard expression (case insenstive).
	*/
	kpis []string

	dryrun bool

	kpiWildcards []*regexp.Regexp
}

func makeThresholdWorkflow(cfg config.Config, services []string, kpis []string, dryrun bool) thresholdWorkflow {
	kpiWildcards := make([]*regexp.Regexp, len(kpis))

	for i, kpiStr := range kpis {
		kpiWildcards[i] = util.WildcardToRegexp(kpiStr)
	}

	return thresholdWorkflow{core.MakeWorkflow(cfg), services, kpis, dryrun, kpiWildcards}
}

// a list of selectors to display.
// returns * when the list of selectors is empty.
// This is to help indicate that empty list means all services or KPIs..
func (w *thresholdWorkflow) displaySelectors(selectors []string) []string {
	if len(selectors) > 0 {
		return selectors
	}
	return []string{"*"}
}

func (w *thresholdWorkflow) buildRegexFilter(fieldName, regexPattern string) (string, error) {
	filterMap := map[string]map[string]string{
		fieldName: {
			"$regex": regexPattern,
		},
	}

	filterJSON, err := json.Marshal(filterMap)
	if err != nil {
		return "", err
	}

	return string(filterJSON), nil
}

func (w *thresholdWorkflow) buildKeyFilter(keys []string) (string, error) {
	if len(keys) == 0 {
		return "", fmt.Errorf("keys slice is empty")
	}

	orConditions := make([]map[string]string, len(keys))
	for i, key := range keys {
		orConditions[i] = map[string]string{
			"_key": key,
		}
	}

	filterMap := map[string]interface{}{
		"$or": orConditions,
	}

	filterJSON, err := json.Marshal(filterMap)
	if err != nil {
		return "", err
	}

	return string(filterJSON), nil
}

func (w *thresholdWorkflow) servicesIter(ctx context.Context, p *models.Parameters) iter.Seq2[*models.ItsiObj, error] {
	client := w.Cfg.ClientConfig()
	return models.NewItsiObj(client, "", "", "service").Iter(ctx, p)
}

func (w *thresholdWorkflow) allServices(ctx context.Context) iter.Seq2[*models.ItsiObj, error] {
	return w.servicesIter(ctx, nil)
}

func (w *thresholdWorkflow) servicesByKey(ctx context.Context, keys []string) iter.Seq2[*models.ItsiObj, error] {
	filter, err := w.buildKeyFilter(keys)
	if err != nil {
		w.Log.Fatal("Failed to render a filter expression to filter services by key", "keys", keys)
	}
	return w.servicesIter(ctx, &models.Parameters{Filter: filter})
}

func (w *thresholdWorkflow) servicesByTitle(ctx context.Context, titlePattern string) iter.Seq2[*models.ItsiObj, error] {
	filter, err := w.buildRegexFilter("title", titlePattern)
	if err != nil {
		w.Log.Fatal("Failed to render a filter expression to filter services by title", "titlePattern", titlePattern)
	}
	return w.servicesIter(ctx, &models.Parameters{Filter: filter})
}

/*
Returns an iterator that will stream service objects.
If `services` field is not empty, only services matching provided selectors will be streamed.
Otherwise, all services will be streamed.
*/
func (w *thresholdWorkflow) Services(ctx context.Context) iter.Seq2[*models.ItsiObj, error] {
	if len(w.services) == 0 {
		return w.allServices(ctx)
	}

	serviceWildcards := make([]string, len(w.services))
	for i, serviceStr := range w.services {
		serviceWildcards[i] = util.WildcardToRegexpStr(serviceStr)
	}

	iters := []iter.Seq2[*models.ItsiObj, error]{}

	for c := range slices.Chunk(w.services, 10) {
		iters = append(iters, w.servicesByKey(ctx, c))
	}
	for _, sw := range serviceWildcards {
		iters = append(iters, w.servicesByTitle(ctx, sw))
	}

	return util.Concat2(iters...)
}

func (w *thresholdWorkflow) kpiMatch(kpiID, kpiTitle string) bool {
	if len(w.kpis) == 0 {
		return true
	}

	if slices.Contains(w.kpis, kpiID) {
		return true
	}
	for _, re := range w.kpiWildcards {
		if re.MatchString(kpiTitle) {
			return true
		}
	}

	return false
}

func (w *thresholdWorkflow) entityThresholds() map[string]any {
	const baseSeverity = "normal"
	return map[string]any{
		"baseSeverityLabel":      baseSeverity,
		"baseSeverityValue":      util.SeverityMap[baseSeverity].SeverityValue,
		"baseSeverityColor":      util.SeverityMap[baseSeverity].SeverityColor,
		"baseSeverityColorLight": util.SeverityMap[baseSeverity].SeverityColorLight,
		"metricField":            "",
		"renderBoundaryMin":      0,
		"renderBoundaryMax":      100,
		"isMaxStatic":            false,
		"isMinStatic":            false,
		"gaugeMin":               0,
		"gaugeMax":               100,
		"thresholdLevels":        []string{},
	}
}

func (w *thresholdWorkflow) aggregateThresholds(thresholdLevels []map[string]any) map[string]any {
	const baseSeverity = "normal"
	return map[string]any{
		"baseSeverityLabel":      baseSeverity,
		"baseSeverityValue":      util.SeverityMap[baseSeverity].SeverityValue,
		"baseSeverityColor":      util.SeverityMap[baseSeverity].SeverityColor,
		"baseSeverityColorLight": util.SeverityMap[baseSeverity].SeverityColorLight,
		"metricField":            "",
		"renderBoundaryMin":      0,
		"renderBoundaryMax":      100,
		"isMaxStatic":            false,
		"isMinStatic":            false,
		"gaugeMin":               0,
		"gaugeMax":               100,
		"thresholdLevels":        thresholdLevels,
	}
}

func (w *thresholdWorkflow) defaultPolicies() map[string]any {
	return map[string]any{
		"default_policy": map[string]any{
			"title":                "Default",
			"aggregate_thresholds": w.aggregateThresholds([]map[string]any{}),
			"entity_thresholds":    w.entityThresholds(),
			"policy_type":          "static",
			"time_blocks":          []string{},
		},
	}
}

func (w *thresholdWorkflow) resetThresholding(kpi map[string]any) {
	//reset static aggregate thresholds:
	kpi["aggregate_thresholds"] = w.aggregateThresholds([]map[string]any{})

	//reset static entity thresholds:
	kpi["entity_thresholds"] = w.entityThresholds()

	//reset outlier exclusion config
	kpi["aggregate_outlier_detection_enabled"] = false
	delete(kpi, "outlier_detection_algo")
	delete(kpi, "outlier_detection_sensitivity")

	//reset time variate thresholds
	kpi["time_variate_thresholds"] = false
	delete(kpi, "time_variate_thresholds_specification")

	//reset adaptive thresholds
	kpi["adaptive_thresholds_is_enabled"] = false
	delete(kpi, "adaptive_thresholding_training_window")

	//remove threshold recommendations
	delete(kpi, "threshold_recommendations")
}
