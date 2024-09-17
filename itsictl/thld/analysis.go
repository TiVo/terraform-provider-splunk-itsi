package thld

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"sync"
	"text/template"

	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/splunk"
	"github.com/tivo/terraform-provider-splunk-itsi/util"
)

var _ util.Processor[kpisByTrainingConfig] = &analysisProcessor{}

type analysisProcessor struct {
	batches           []kpisByTrainingConfig
	searchConcurrency int

	workflow *ThresholdRecommendationWorkflow

	svcByKpi map[string]*models.ItsiObj

	results             policiesByService
	trainingConfigByKPI map[kpiID]trainingConfig

	mu sync.Mutex
}

func (p *analysisProcessor) Items() []kpisByTrainingConfig {
	return p.batches
}

func (p *analysisProcessor) splunkreq(batch kpisByTrainingConfig) *provider.SplunkRequest {
	tpl, _ := template.New("_").Parse(`
		| mstats latest(alert_value) AS alert_value latest(alert_level) AS alert_level
		WHERE {{ .BT }}get_itsi_summary_metrics_index{{ .BT }}
		{{ .FILTER }}
		AND is_filled_gap_event!=1 AND is_null_alert_value=0
		{{ .BT }}metrics_service_level_kpi_only{{ .BT }} by itsi_kpi_id, itsi_service_id span=1m
		| where alert_level!=-2
		| table _time, alert_value, alert_level, itsi_kpi_id, itsi_service_id
		| sort 0 itsi_kpi_id | recommendthresholdtemplate threshold_direction={{ .DIRECTION }}
	`)

	searches := []provider.SplunkSearch{}

	for trainingConf, kpis := range batch {
		var buf bytes.Buffer
		if err := tpl.Execute(&buf, struct{ FILTER, DIRECTION, BT string }{FILTER: kpis.filterExpression(), DIRECTION: trainingConf.thresholdDirection, BT: "`"}); err != nil {
			p.workflow.Log.Fatal(
				"unexpected error while rendering a Splunk search for running ML thresholding analysis",
				"filter", kpis.filterExpression(),
				"direction", trainingConf.thresholdDirection)
		}
		latestTime := trainingConf.startTime + trainingConf.size*86400

		searches = append(searches, provider.SplunkSearch{
			Query:               buf.String(),
			AllowNoResults:      false,
			AllowPartialResults: false,
			EarliestTime:        fmt.Sprintf("%d", trainingConf.startTime),
			LatestTime:          fmt.Sprintf("%d", latestTime),
			App:                 "itsi",
			User:                "nobody",
			Timeout:             trainingSearchTimeoutSeconds,
		})
	}

	return provider.NewSplunkRequest(p.workflow.Cfg.ClientConfig(), searches, p.searchConcurrency, []string{}, false, " ")
}

func (p *analysisProcessor) DoMLAnalysis(ctx context.Context, batch kpisByTrainingConfig) error {
	req := p.splunkreq(batch)

	rows, diags := req.Run(ctx)
	if diags.HasError() {
		return fmt.Errorf("splunk search failed")
	}

	results := make(policiesByService)

	for _, row := range rows {

		recommendationFlag, ok := row["Recommendation Flag"].(string)
		if !ok {
			return fmt.Errorf("unexpected results from ML analysis search: recommendation flag is missing")
		}

		kpiID, ok := row["itsi_kpi_id"].(string)
		if !ok {
			p.workflow.Log.Warn("KPI analysis search returned a row without itsi_kpi_id", "row", row)
			continue
		}

		svc := p.svcByKpi[kpiID]

		if svc == nil {
			if kpiID == "None" && recommendationFlag == "INSUFFICIENT_DATA" {
				continue
			}

			return fmt.Errorf("unknown service for %s KPI ID: %#v", kpiID, row)
		}

		if _, ok := results[svc]; !ok {
			results[svc] = make(policiesByKpi)
		}
		if _, ok := results[svc][kpiID]; !ok {
			results[svc][kpiID] = []map[string]splunk.Value{}
		}
		results[svc][kpiID] = append(results[svc][kpiID], row)
	}

	p.mu.Lock()
	maps.Copy(p.results, results)
	p.mu.Unlock()

	return nil
}

func (p *analysisProcessor) Process(ctx context.Context, batch kpisByTrainingConfig) error {
	return p.DoMLAnalysis(ctx, batch)
}
