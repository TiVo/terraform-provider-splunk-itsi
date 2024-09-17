package thld

import (
	"context"
	"fmt"
	"strings"

	"github.com/tivo/terraform-provider-splunk-itsi/itsictl/config"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider"
	"github.com/tivo/terraform-provider-splunk-itsi/util"
	"gopkg.in/yaml.v3"
)

type ThresholdResetWorkflow struct {
	thresholdWorkflow
}

func NewThresholdResetWorkflow(cfg config.Config, services []string, kpis []string, dryrun bool) *ThresholdResetWorkflow {
	return &ThresholdResetWorkflow{makeThresholdWorkflow(cfg, services, kpis, dryrun)}
}

type serviceThresholdResetProcessor struct {
	services []*models.ItsiObj

	w *ThresholdResetWorkflow
}

func (p *serviceThresholdResetProcessor) Items() []*models.ItsiObj {
	return p.services
}

func (p *serviceThresholdResetProcessor) Process(ctx context.Context, svc *models.ItsiObj) (err error) {

	svcMap, err := svc.RawJson.ToInterfaceMap()
	if err != nil {
		return err
	}

	svcID, svcTitle := svc.RESTKey, svcMap["title"].(string)

	kpis, err := provider.UnpackSlice[map[string]any](svcMap["kpis"])
	if err != nil {
		return err
	}

	kpisReset := []string{}

	for _, kpi := range kpis {
		kpiID, kpiTitle := kpi["_key"].(string), kpi["title"].(string)

		if strings.HasPrefix(kpiID, "SHKPI") {
			continue
		}

		match := p.w.kpiMatch(kpiID, kpiTitle)
		if match {
			p.w.resetThresholding(kpi)
			kpisReset = append(kpisReset, fmt.Sprintf("%s (%s)", kpiTitle, kpiID))
		}
	}

	if len(kpisReset) > 0 {
		svcMap["kpis"] = kpis
		if err := svc.PopulateRawJSON(ctx, svcMap); err != nil {
			return fmt.Errorf("failed to populate service api model: %w", err)
		}

		if !p.w.dryrun {
			if diags := svc.UpdateAsync(ctx); diags.HasError() {
				return fmt.Errorf("failed to save service: %#v", diags)
			}
		}

		kpisResetYaml, err := yaml.Marshal(kpisReset)
		if err != nil {
			return err
		}

		msg := ""
		if !p.w.dryrun {
			msg = fmt.Sprintf("Service [ %s ] has been saved. ", svcTitle)
		}
		msg += fmt.Sprintf("Thresholds have been reset for %d KPIs.", len(kpisReset))

		p.w.Log.Info(
			msg,
			"service_id", svcID,
			"kpis_reset", string(kpisResetYaml),
		)
	}

	return
}

func (w *ThresholdResetWorkflow) processBatch(ctx context.Context, services []*models.ItsiObj) error {
	return util.ProcessInParallel(ctx, &serviceThresholdResetProcessor{services, w}, w.Cfg.Concurrency)
}

func (w *ThresholdResetWorkflow) Execute(ctx context.Context) error {

	w.Log.Info(
		"Starting threshold reset workflow",
		"service_selectors", w.displaySelectors(w.services),
		"kpi_selectors", w.displaySelectors(w.kpis),
		"concurrency", w.Cfg.Concurrency,
		"dry_run", w.dryrun,
	)

	batch := []*models.ItsiObj{}
	for svc, err := range w.Services(ctx) {
		if err != nil {
			return err
		}

		if batch = append(batch, svc); len(batch)%w.Cfg.Concurrency == 0 {
			if err = w.processBatch(ctx, batch); err != nil {
				return err
			}
			batch = []*models.ItsiObj{}
		}
	}

	return w.processBatch(ctx, batch)
}
