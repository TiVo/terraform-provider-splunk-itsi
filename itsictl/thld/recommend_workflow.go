package thld

import (
	"context"
	"fmt"
	"iter"
	"math"
	"time"

	"github.com/tivo/terraform-provider-splunk-itsi/itsictl/config"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider"
	"github.com/tivo/terraform-provider-splunk-itsi/util"
	"gopkg.in/yaml.v3"
)

type ThresholdRecommendationWorkflow struct {
	thresholdWorkflow

	useLatestData          bool
	insufficientDataAction string // 'skip' or 'reset'

	latestDataStartDates map[int]int
}

func getLatestStartDatesMap() map[int]int {
	//in case we are analyzing the latest available data,
	//we'll use one the following precomputed starting dates (one per supported training window).
	//this is to reduce the number of splunk searches we need to run (because a different starting date would require a new search)
	startDates := make(map[int]int)
	days := []int{7, 14, 30, 60}
	now := int(time.Now().Unix())
	for _, d := range days {
		startDates[d] = now - (d * 86400)
	}
	return startDates
}

func NewThresholdRecommendationWorkflow(cfg config.Config, services []string, kpis []string, dryrun bool, useLatestData bool, insufficientDataAction string) *ThresholdRecommendationWorkflow {

	return &ThresholdRecommendationWorkflow{
		makeThresholdWorkflow(cfg, services, kpis, dryrun),
		useLatestData,
		insufficientDataAction,
		getLatestStartDatesMap(),
	}
}

func (w *ThresholdRecommendationWorkflow) newAnalysisProcessor(batches []kpisByTrainingConfig, searchConcurrency int) (p *analysisProcessor) {
	svcByKpi := map[string]*models.ItsiObj{}
	trainingConfigByKPI := make(map[kpiID]trainingConfig)
	for _, b := range batches {
		for trainingConfig, kpis := range b {
			for _, kpi := range kpis {
				svcByKpi[kpi.id] = kpi.service
				trainingConfigByKPI[kpi] = trainingConfig
			}
		}
	}

	p = &analysisProcessor{
		workflow: w,

		batches:           batches,
		searchConcurrency: searchConcurrency,

		svcByKpi:            svcByKpi,
		trainingConfigByKPI: trainingConfigByKPI,

		results: make(policiesByService),
	}
	return
}

func (w *ThresholdRecommendationWorkflow) newServiceThresholdUpdateProcessor(trainingConfigByKPI map[kpiID]trainingConfig, policies policiesByService) (p *serviceThresholdUpdateProcessor) {
	return &serviceThresholdUpdateProcessor{
		trainingConfigByKPI: trainingConfigByKPI,
		policies:            policies,
		workflow:            w,
	}
}

func (w *ThresholdRecommendationWorkflow) analysisStartDate(kpi map[string]any) (startDate int, err error) {
	if w.useLatestData {
		days, err := parseTrainingWindowSize(kpi["recommendation_training_window"].(string))
		if err != nil {
			return 0, err
		}

		ok := false
		startDate, ok = w.latestDataStartDates[days]
		if !ok {
			return 0, fmt.Errorf("unsupported training window: %v", days)
		}
	} else {
		startDate = int(kpi["recommendation_start_date"].(float64))
	}
	return
}

func (w *ThresholdRecommendationWorkflow) analysisBatches(ctx context.Context, numberOfBatches int) iter.Seq2[[]kpisByTrainingConfig, error] {

	return func(yield func([]kpisByTrainingConfig, error) bool) {

		batches := make([]kpisByTrainingConfig, 0, numberOfBatches)
		currentBatch := kpisByTrainingConfig{}

		batchServiceCount := 0

		for svc, err := range w.Services(ctx) {
			svcTrainingConfig := kpisByTrainingConfig{}

			if err != nil {
				yield(nil, err)
				return
			}

			svcMap, err := svc.RawJson.ToInterfaceMap()
			if err != nil {
				yield(nil, err)
				return
			}

			kpis, err := provider.UnpackSlice[map[string]any](svcMap["kpis"])
			if err != nil {
				yield(nil, err)
				return
			}

			for _, kpi := range kpis {
				id, kpiTitle := kpi["_key"].(string), kpi["title"].(string)
				match := w.kpiMatch(id, kpiTitle)
				if !match {
					continue
				}

				isRecommdedTimePolicies, _ := kpi["is_recommended_time_policies"].(bool)
				if !isRecommdedTimePolicies {
					continue
				}

				thresholdDirection := kpi["threshold_direction"].(string)
				recommendationTrainingWindow := kpi["recommendation_training_window"].(string)

				startDate, err := w.analysisStartDate(kpi)
				if err != nil {
					yield(nil, err)
					return
				}

				tw, err := makeTrainingWindow(startDate, recommendationTrainingWindow)
				if err != nil {
					yield(nil, err)
					return
				}
				tc := trainingConfig{tw, thresholdDirection}

				svcTrainingConfig[tc] = append(svcTrainingConfig[tc], []kpiID{{svc, id}}...)

			}

			if len(svcTrainingConfig) == 0 {
				continue
			}

			if batchServiceCount < maxServicesPerBatch && currentBatch.hasCapacityFor(svcTrainingConfig) {
				currentBatch.merge(svcTrainingConfig)
				batchServiceCount++
			} else {
				batches = append(batches, currentBatch)
				currentBatch = svcTrainingConfig
				batchServiceCount = 1
			}

			if len(batches) == numberOfBatches {
				if !yield(batches, nil) {
					return
				}
				batches = make([]kpisByTrainingConfig, 0, numberOfBatches)
			}
		}

		if len(currentBatch) > 0 {
			batches = append(batches, currentBatch)
		}
		if len(batches) > 0 {
			if !yield(batches, nil) {
				return
			}
		}

	}
}

// parallelismParameters calculates the number of parallel batches and parallel searches
// within each batch for the threshold recommendation workflow. Given the total allowed concurrency
// (w.Cfg.Concurrency) and a preferred ratio of parallel searches to parallel batches, it finds
// integer values for 'parallelBatches' and 'parallelSearches' such that:
//   - parallelSearches * parallelBatches == w.Cfg.Concurrency
//   - The ratio parallelSearches / parallelBatches is as close as possible to 'preferredRatio'.
func (w *ThresholdRecommendationWorkflow) analysisParallelismParameters(preferredRatio float64) (parallelSearches, parallelBatches int) {
	minRatioDiff := math.MaxFloat64
	for i := 1; i <= w.Cfg.Concurrency; i++ {
		if w.Cfg.Concurrency%i == 0 {
			j := w.Cfg.Concurrency / i
			currentRatio := float64(i) / float64(j)
			ratioDiff := math.Abs(currentRatio - preferredRatio)
			if ratioDiff < minRatioDiff {
				minRatioDiff = ratioDiff
				parallelSearches = i
				parallelBatches = j
			}
		}
	}

	return
}

func (w *ThresholdRecommendationWorkflow) Execute(ctx context.Context) error {
	parallelSearches, parallelBatches := w.analysisParallelismParameters(0.5)

	w.Log.Info(
		"Starting threshold recommendation workflow",
		"service_selectors", w.displaySelectors(w.services),
		"kpi_selectors", w.displaySelectors(w.kpis),
		"use_latest_data", w.useLatestData,
		"insufficient_data_action", w.insufficientDataAction,
		"parallel_batches", parallelBatches,
		"parallel_searches", parallelSearches,
		"concurrency", w.Cfg.Concurrency,
		"dry_run", w.dryrun,
	)

	for batches, err := range w.analysisBatches(ctx, parallelBatches) {
		if err != nil {
			return err
		}

		// run ML analysis
		w.printBatches(batches)
		analysisProcessor := w.newAnalysisProcessor(batches, parallelSearches)
		if err = util.ProcessInParallel(ctx, analysisProcessor, parallelBatches); err != nil {
			return err
		}

		// update thresholding policies based on the analysis results
		if err = util.ProcessInParallel(
			ctx,
			w.newServiceThresholdUpdateProcessor(analysisProcessor.trainingConfigByKPI, analysisProcessor.results),
			w.Cfg.Concurrency,
		); err != nil {
			return err
		}
	}

	return nil
}

func (w *ThresholdRecommendationWorkflow) printBatches(batches []kpisByTrainingConfig) {
	details := []any{}
	nKpis := 0
	nSplunkSearches := 0

	services := util.NewSet[*models.ItsiObj]()

	for _, b := range batches {
		batchDetails := map[string][]string{}

		for c, kpis := range b {
			nSplunkSearches++

			for _, kpi := range kpis {

				nKpis++
				services.Add(kpi.service)

				if _, ok := batchDetails[c.String()]; !ok {
					batchDetails[c.String()] = []string{}
				}

				batchDetails[c.String()] = append(batchDetails[c.String()], fmt.Sprintf("%s/%s", kpi.service.RESTKey, kpi.id))
			}
		}
		details = append(details, batchDetails)
	}

	batchesYaml, err := yaml.Marshal(details)
	if err != nil {
		w.Log.Fatal("Failed to marshal batches YAML", "batches", details, "error", err)
	}
	w.Log.Info(fmt.Sprintf("Running %d Splunk searches to analyze %d KPIs across %d services", nSplunkSearches, nKpis, len(services)), "batches", string(batchesYaml))
}
