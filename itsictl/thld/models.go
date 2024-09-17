package thld

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/splunk"
)

const (
	kpisPerSearchThld            = 10
	maxServicesPerBatch          = 5
	trainingSearchTimeoutSeconds = 300
)

type trainingWindow struct {
	startTime int //unix timestamp in seconds
	size      int //number of days
}

func parseTrainingWindowSize(sizeStr string) (size int, err error) {
	re := regexp.MustCompile(`^-(\d+)d$`)
	sizeMatch := re.FindStringSubmatch(sizeStr)

	if len(sizeMatch) != 2 {
		err = fmt.Errorf("[Error] %s is not a valid training window: %w", sizeStr, err)
		return
	}

	size, err = strconv.Atoi(sizeMatch[1])
	if err != nil {
		err = fmt.Errorf("[Error] %s is not a valid training window: %w", sizeStr, err)
		return
	}
	return
}

func makeTrainingWindow(startTime int, sizeStr string) (tw trainingWindow, err error) {
	size, err := parseTrainingWindowSize(sizeStr)
	if err != nil {
		return
	}
	tw = trainingWindow{startTime, size}
	return
}

type trainingConfig struct {
	trainingWindow
	thresholdDirection string
}

func (tc *trainingConfig) String() string {
	t := time.Unix(int64(tc.startTime), 0)
	strDate := t.Format("2006-01-02 15:04:05")
	return fmt.Sprintf("%s/%dd/%s", strDate, tc.size, tc.thresholdDirection)
}

type kpiID struct {
	service *models.ItsiObj
	id      string
}

type kpiIDList []kpiID

func (l kpiIDList) kpiIDsByService() (res map[*models.ItsiObj][]string) {
	res = make(map[*models.ItsiObj][]string)
	for _, kpiID := range l {
		if _, ok := res[kpiID.service]; !ok {
			res[kpiID.service] = []string{}
		}
		res[kpiID.service] = append(res[kpiID.service], kpiID.id)
	}
	return
}

func (l kpiIDList) filterExpression() string {
	if len(l) == 0 {
		return ""
	}

	kpisByService := l.kpiIDsByService()
	svcStrConditionList := make([]string, 0, len(kpisByService))
	for svc, kpis := range kpisByService {
		kpiStrList := make([]string, len(kpis))
		for i, kpi := range kpis {
			kpiStrList[i] = fmt.Sprintf(`"%s"`, kpi)
		}
		kpiFilter := strings.Join(kpiStrList, ",")

		svcStrConditionList = append(svcStrConditionList, fmt.Sprintf(`(itsi_service_id="%s" AND itsi_kpi_id IN (%s)) `, svc.RESTKey, kpiFilter))
	}

	return fmt.Sprintf("AND ( %s )", strings.Join(svcStrConditionList, " OR "))
}

type kpisByTrainingConfig map[trainingConfig]kpiIDList

// returns a boolean indicating whether two <training config, kpis > maps can be merged
// without any search's list of KPIs exceeding `kpisPerSearchThld` number of KPIs
func (a kpisByTrainingConfig) hasCapacityFor(b kpisByTrainingConfig) bool {
	for k, v := range b {
		if _, ok := a[k]; ok {
			if len(a[k])+len(v) > kpisPerSearchThld {
				return false
			}
		}
	}
	return true
}

func (a kpisByTrainingConfig) merge(b kpisByTrainingConfig) {
	for k, v := range b {
		if s, ok := a[k]; ok {
			a[k] = append(s, v...)
		} else {
			a[k] = v
		}
	}
}

type policiesByKpi map[string][]map[string]splunk.Value
type policiesByService map[*models.ItsiObj]policiesByKpi
