package thld

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider"
	"github.com/tivo/terraform-provider-splunk-itsi/util"
	"gopkg.in/yaml.v3"
)

var _ util.Processor[*models.ItsiObj] = &serviceThresholdUpdateProcessor{}

type serviceThresholdUpdateProcessor struct {
	policies            policiesByService
	trainingConfigByKPI map[kpiID]trainingConfig

	workflow *ThresholdRecommendationWorkflow
}

func (p *serviceThresholdUpdateProcessor) Items() []*models.ItsiObj {
	services := util.NewSet[*models.ItsiObj]()
	for kpiID := range p.trainingConfigByKPI {
		services.Add(kpiID.service)
	}
	return services.ToSlice()
}

func (p *serviceThresholdUpdateProcessor) kpisToConfigure(svc *models.ItsiObj) util.Set[string] {
	kpiIDs := util.NewSet[string]()
	for kpiID := range p.trainingConfigByKPI {
		if kpiID.service == svc {
			kpiIDs.Add(kpiID.id)
		}
	}
	return kpiIDs
}

func (p *serviceThresholdUpdateProcessor) newThresholdConfigurator(svc *models.ItsiObj) (*serviceThresholdingConfigurator, error) {

	svcMap, err := svc.RawJson.ToInterfaceMap()
	if err != nil {
		return nil, err
	}

	svcID, svcTitle := svc.RESTKey, svcMap["title"].(string)
	policies, ok := p.policies[svc]
	if !ok {
		//No policies are were generated for the service.
		//This probably means that there was incufficient data to traing any ML-configured KPI in the service
		policies = make(policiesByKpi)
	}

	kpis, err := provider.UnpackSlice[map[string]any](svcMap["kpis"])
	if err != nil {
		return nil, err
	}

	kpiIDsToConfigure := p.kpisToConfigure(svc)
	kpisToConfigure := []map[string]any{}

	for _, kpi := range kpis {
		if kpiIDsToConfigure.Contains(kpi["_key"].(string)) {
			kpisToConfigure = append(kpisToConfigure, kpi)
		}
	}

	return &serviceThresholdingConfigurator{
		service: svc,

		serviceID:    svcID,
		serviceTitle: svcTitle,

		svcMap: svcMap,

		policies:            policies,
		kpis:                kpis,
		kpisToConfigure:     kpisToConfigure,
		trainingConfigByKPI: p.trainingConfigByKPI,

		changeSummary: map[string]string{},
		failedKpis:    util.NewSet[string](),

		workflow: p.workflow,
	}, nil
}

func (p *serviceThresholdUpdateProcessor) Process(ctx context.Context, svc *models.ItsiObj) error {
	configurator, err := p.newThresholdConfigurator(svc)
	if err != nil {
		return err
	}

	if err = configurator.Configure(); err != nil {
		return err
	}

	if !p.workflow.dryrun {
		if diags := svc.UpdateAsync(ctx); diags.HasError() {
			return fmt.Errorf("failed to save service: %#v", diags)
		}
	}

	p.printServiceUpdateSummary(configurator)

	return nil
}

func (p *serviceThresholdUpdateProcessor) printServiceUpdateSummary(configurator *serviceThresholdingConfigurator) {
	summaryYaml, err := yaml.Marshal(configurator.changeSummary)
	if err != nil {
		p.workflow.Log.Fatal("Failed to marshal service update summary YAML", "changeSummary", configurator.changeSummary, "error", err)
	}

	logLevel := log.InfoLevel
	successfulKpis := len(configurator.kpisToConfigure) - len(configurator.failedKpis)
	msg := ""
	if !p.workflow.dryrun {
		msg = fmt.Sprintf("Service [ %s ] has been saved. ", configurator.serviceTitle)
	}
	msg += fmt.Sprintf("Thresholds have been configured successfully for %d KPIs.", successfulKpis)

	if len(configurator.failedKpis) > 0 {
		logLevel = log.WarnLevel
		msg += fmt.Sprintf(" %d KPIs have not been configured due to insufficient data or KPI value being constant over the training period.", len(configurator.failedKpis))
	}

	kpisFailed := "kpis_skipped"
	if configurator.workflow.insufficientDataAction == "reset" {
		kpisFailed = "kpis_reset"
	}

	p.workflow.Log.Log(logLevel,
		msg,
		"service_id", configurator.serviceID,
		"kpis_processed", len(configurator.kpisToConfigure),
		"kpis_configured", successfulKpis,
		kpisFailed, len(configurator.failedKpis),
		"kpis_update_summary", string(summaryYaml),
	)
}
