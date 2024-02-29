package provider

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

func serviceTFFormat(b *models.Base) (string, error) {
	res := ResourceService()
	resData := res.Data(nil)
	d := populateServiceResourceData(context.Background(), b, resData)
	if len(d) > 0 {
		err := d[0].Validate()
		if err != nil {
			return "", err
		}
		return "", errors.New(d[0].Summary)
	}
	resourcetpl, err := NewResourceTemplate(resData, res.Schema, "title", "itsi_service")
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
}

// testOverridable
var GenerateUUID = func(internalIdentifier string) (string, error) {
	return uuid.GenerateUUID()
}

func serviceBase(clientConfig models.ClientConfig, key string, title string) *models.Base {
	base := models.NewBase(clientConfig, key, title, "service")
	return base
}

/* helper data structure to allow us specify metrics by title rather than ID */

type KPIBSMetricLookup struct {
	titleByKpiBsIDandMetricID map[string]string
}

func (ml *KPIBSMetricLookup) lookupKey(kpiBSID, metricID string) string {
	return fmt.Sprintf("%s:%s", kpiBSID, metricID)
}

func (ml *KPIBSMetricLookup) getKpiBSMetricTitleByID(ctx context.Context, cc models.ClientConfig, id string) (titleByID map[string]string, err error) {
	kpiBsData, err := getKpiBSData(ctx, cc, id)
	if err != nil {
		return nil, err
	}
	titleByID = make(map[string]string)

	for _, metric_ := range kpiBsData["metrics"].([]interface{}) {
		metric := metric_.(map[string]interface{})
		titleByID[metric["_key"].(string)] = metric["title"].(string)
	}
	return
}

func (ml *KPIBSMetricLookup) lookupMetricTitleByID(ctx context.Context, cc models.ClientConfig, kpiBSID, metricID string) (title string, err error) {
	if ml.titleByKpiBsIDandMetricID == nil {
		ml.titleByKpiBsIDandMetricID = make(map[string]string)
	}
	title, ok := ml.titleByKpiBsIDandMetricID[ml.lookupKey(kpiBSID, metricID)]
	if ok {
		return
	}

	metricTitleByID, err := ml.getKpiBSMetricTitleByID(ctx, cc, kpiBSID)
	if err != nil {
		return
	}

	for metricID, metricTitle := range metricTitleByID {
		ml.titleByKpiBsIDandMetricID[ml.lookupKey(kpiBSID, metricID)] = metricTitle
	}

	if title, ok = ml.titleByKpiBsIDandMetricID[ml.lookupKey(kpiBSID, metricID)]; !ok {
		err = fmt.Errorf("metric %s not found in KPI Base search %s", metricID, kpiBSID)
	}
	return
}

func getKpiBSData(ctx context.Context, cc models.ClientConfig, id string) (map[string]interface{}, error) {
	kpiSearchBase, err := kpiBaseSearchBase(cc, id, "").Find(ctx)
	if err != nil {
		return nil, err
	}

	if kpiSearchBase == nil {
		return nil, fmt.Errorf("KPI Base search %s not found", id)
	}

	return kpiSearchBase.RawJson.ToInterfaceMap()
}

/*
 *  GENERATED_SEARCH_ATTRIBUTES:
 *  UI generates these searches via get_kpi_searches POST request
 *  with following mappings:
 *  'search' <= 'kpi_base_search',
 *  'kpi_base_search' <= 'kpi_base_search',
 *
 *  'search_aggregate' <= 'single_value_search',
 *  'search_entities' <= 'single_value_search',
 *
 *  'search_time_series' <= 'time_series_search',
 *  'search_time_series_aggregate' <= 'time_series_search',
 *
 *  'search_time_series_entities' <= 'entity_time_series_search,
 *  'search_time_compare' <= 'compare_search',
 *  'search_alert' <= 'alert_search,
 *  'search_alert_entities' (!) Didn't mapped in UI. Default ""
 *
 *  BUT in case all base search field are passed splunk generates it automatically after POST/PUT service
 *
 *  KPI BASE SEARCH is managed through terraform resource, so if kpi base searches' content is changed, splunk responsibility
 *  to update linked fields, there is no need to save linked values in the resource.
 *
 */
func ResourceService() *schema.Resource {

	customStaticThresholdSchema := map[string]*schema.Schema{
		"entity_thresholds": {
			Type:     schema.TypeSet,
			Required: true,
			Elem: &schema.Resource{
				Schema: getKpiThresholdSettingsSchema(),
			},
		},
		"aggregate_thresholds": {
			Type:     schema.TypeSet,
			Required: true,
			Elem: &schema.Resource{
				Schema: getKpiThresholdSettingsSchema(),
			},
		},
	}

	kpisSchema := map[string]*schema.Schema{
		"id": {
			Type:     schema.TypeString,
			Computed: true,
			//Optional: true,
			Description: `id (splunk _key) is automatically generated sha1 string, from base_search_id & metric_id seed,
			 concatenated with serviceId.`,
		},
		"title": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Name of the kpi. Can be any unique value.",
		},
		"description": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "User-defined description for the KPI. ",
		},
		"type": {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      "kpis_primary",
			Description:  "Could be service_health or kpis_primary.",
			ValidateFunc: validation.StringInSlice([]string{"kpis_primary", "service_health"}, false),
		},
		"urgency": {
			Type:         schema.TypeInt,
			Optional:     true,
			Default:      5,
			Description:  "User-assigned importance value for this KPI.",
			ValidateFunc: validation.IntBetween(0, 11),
		},
		// BASE_SEARCH_KPI_ATTRIBUTES
		"base_search_id": {
			Type:     schema.TypeString,
			Required: true,
		},
		"search_type": {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      "shared_base",
			ValidateFunc: validation.StringInSlice([]string{"shared_base"}, false),
		},
		"base_search_metric": {
			Type:     schema.TypeString,
			Required: true,
		},
		"threshold_template_id": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"custom_threshold": {
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Resource{
				Schema: customStaticThresholdSchema,
			},
		},
	}
	entityRulesSchema := map[string]*schema.Schema{
		"rule": {
			Type:        schema.TypeSet,
			Optional:    true,
			Description: "A set of rules within the rule group, which are combined using AND operator.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"field": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The field in the entity definition to compare values to evaluate this rule.",
					},
					"field_type": {
						Type:         schema.TypeString,
						Required:     true,
						Description:  "Takes values alias, info or title specifying in which category of fields the field attribute is located.",
						ValidateFunc: validation.StringInSlice([]string{"alias", "entity_type", "info", "title"}, false),
					},
					"rule_type": {
						Type:         schema.TypeString,
						Required:     true,
						Description:  "Takes values not or matches to indicate whether it's an inclusion or exclusion rule.",
						ValidateFunc: validation.StringInSlice([]string{"matches", "not"}, false),
					},
					"value": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "Values to evaluate in the rule. To specify multiple values, separate them with a comma. Values are not case sensitive.",
					},
				},
			},
		},
	}

	return &schema.Resource{
		Description:   "Manages a Service within ITSI.",
		CreateContext: serviceCreate,
		ReadContext:   serviceRead,
		UpdateContext: serviceUpdate,
		DeleteContext: serviceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: serviceImport,
		},
		Schema: map[string]*schema.Schema{
			"title": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Title of the service.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "User defined description for the service.",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Boolean value defining whether the service should be enabled.",
			},
			"is_healthscore_calculate_by_entity_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				Description: `Set the Service Heath Score calculation to account for the severity levels of individual entities
				               if at least one KPI is split by entity.`,
			},
			"security_group": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "default_itsi_security_group",
				Description: "The team the object belongs to.",
			},
			"entity_rules": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "A set of rule groups that are combined by OR operator.",
				Elem: &schema.Resource{
					Schema: entityRulesSchema,
				},
			},
			"service_depends_on": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "A set of service descriptions with KPIs in those services that this service depends on.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"service": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "_key value of service that this service depends on.",
						},
						"kpis": {
							Type:        schema.TypeSet,
							Required:    true,
							Description: "A set of _key ids for each KPI in service identified by serviceid, which this service will depend on.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"overloaded_urgencies": {
							Type:        schema.TypeMap,
							Optional:    true,
							Description: "A map of urgency overriddes for the KPIs this service is depending on.",
							Elem: &schema.Schema{
								Type: schema.TypeInt,
							},
						},
					},
				},
			},
			"tags": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The tags for the service. Cannot contain the following characters: / \\ \" ' ! @ ? . , ; $ ^ ",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"kpi": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "A set of KPI descriptions for this service.",
				Elem: &schema.Resource{
					Schema: kpisSchema,
				},
			},
			//Computed fields
			"shkpi_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "_key value for the Service Health Score KPI.",
			},
		},
	}
}

func service(ctx context.Context, d *schema.ResourceData, clientConfig models.ClientConfig) (config *models.Base, err error) {
	body := map[string]interface{}{}

	body["object_type"] = "service"
	body["title"] = d.Get("title").(string)
	body["description"] = d.Get("description").(string)
	isHealthscoreCalculateByEntityEnabled := d.Get("is_healthscore_calculate_by_entity_enabled").(bool)
	if isHealthscoreCalculateByEntityEnabled {
		body["is_healthscore_calculate_by_entity_enabled"] = 1
	} else {
		body["is_healthscore_calculate_by_entity_enabled"] = 0
	}
	body["enabled"] = func() int {
		if d.Get("enabled").(bool) {
			return 1
		}
		return 0
	}()
	body["sec_grp"] = d.Get("security_group").(string)

	base := serviceBase(clientConfig, d.Id(), d.Get("title").(string))

	//[kpiId][thresholdId][policyName_severityLabel_dynamicParam]{thresholdValue Float64}
	thresholdValueCache := map[string]map[string]map[string]float64{}
	if d.Id() != "" {
		base, err := base.Find(ctx)
		if err != nil {
			return nil, err
		}

		serviceInterface, err := base.RawJson.ToInterfaceMap()
		if err != nil {
			return nil, err
		}

		if kpis, ok := serviceInterface["kpis"].([]interface{}); ok {
			for _, kpi := range kpis {
				k := kpi.(map[string]interface{})
				if _, ok := k["_key"]; !ok {
					return nil, fmt.Errorf("no kpiId was found for service: %v ", d.Id())
				}
				if _, ok := k["kpi_threshold_template_id"]; !ok || k["kpi_threshold_template_id"].(string) == "" {
					continue
				}

				if _, ok := k["adaptive_thresholds_is_enabled"]; !ok || !k["adaptive_thresholds_is_enabled"].(bool) {
					continue
				}

				kpiId := k["_key"].(string)
				thresholdId := k["kpi_threshold_template_id"].(string)
				thresholdValueCache[kpiId] = map[string]map[string]float64{}
				thresholdValueCache[kpiId][thresholdId] = map[string]float64{}

				if timeVariateThresholdSpecification, ok := k["time_variate_thresholds_specification"].(map[string]interface{}); ok {
					for policyName, policy := range timeVariateThresholdSpecification["policies"].(map[string]interface{}) {
						_policy := policy.(map[string]interface{})
						if aggregate_thresholds, ok := _policy["aggregate_thresholds"].(map[string]interface{}); ok {
							for _, threshold_level := range aggregate_thresholds["thresholdLevels"].([]interface{}) {
								_threshold_level := threshold_level.(map[string]interface{})
								severityValue := _threshold_level["severityValue"].(float64)
								dynamicParam := _threshold_level["dynamicParam"].(float64)
								thresholdValue := _threshold_level["thresholdValue"].(float64)
								key := policyName + fmt.Sprint(severityValue) + "_" + fmt.Sprint(dynamicParam)
								thresholdValueCache[kpiId][thresholdId][key] = thresholdValue
							}
						}
					}
				}

			}
		}
	}

	//compute kpiIds for dataResource
	if d.HasChange("kpi") {
		kpisOld, kpisNew := d.GetChange("kpi")
		kpiOldKeys := map[string]string{}

		for _, kpi := range kpisOld.(*schema.Set).List() {
			kpiData := kpi.(map[string]interface{})

			// kpiid is important to save for historical raw data. Historical raw data makes sense,
			// until base search & metris is same
			internalIdentifier, err := getKpiHashKey(kpiData)
			if err != nil {
				return nil, err
			}

			kpiOldKeys[internalIdentifier] = kpiData["id"].(string)
		}
		for _, kpi := range kpisNew.(*schema.Set).List() {
			kpiData := kpi.(map[string]interface{})

			internalIdentifier, err := getKpiHashKey(kpiData)
			if err != nil {
				return nil, err
			}

			if existingKpiId, ok := kpiOldKeys[internalIdentifier]; ok {
				kpiData["id"] = existingKpiId
			} else {
				kpiData["id"], _ = GenerateUUID(internalIdentifier)
			}
		}
		err := d.Set("kpi", kpisNew)
		if err != nil {
			return nil, err
		}
	}

	itsiKpis := []map[string]interface{}{}
	for _, kpi := range d.Get("kpi").(*schema.Set).List() {
		kpiData := kpi.(map[string]interface{})
		restKey := kpiData["base_search_id"].(string)
		kpiSearchInterface, err := getKpiBSData(ctx, clientConfig, restKey)
		if err != nil {
			return nil, err
		}

		itsiKpi := map[string]interface{}{
			"title":                      kpiData["title"],
			"urgency":                    kpiData["urgency"],
			"search_type":                kpiData["search_type"],
			"type":                       kpiData["type"],
			"base_search_id":             restKey,
			"base_search":                kpiSearchInterface["base_search"],
			"is_entity_breakdown":        kpiSearchInterface["is_entity_breakdown"],
			"is_service_entity_filter":   kpiSearchInterface["is_service_entity_filter"],
			"entity_breakdown_id_fields": kpiSearchInterface["entity_breakdown_id_fields"],
			"entity_id_fields":           kpiSearchInterface["entity_id_fields"],
			"alert_period":               kpiSearchInterface["alert_period"],
			"alert_lag":                  kpiSearchInterface["alert_lag"],
			"search_alert_earliest":      kpiSearchInterface["search_alert_earliest"],
		}

		if description, ok := kpiData["description"]; ok && description != "" {
			itsiKpi["description"] = description
		}

		for _, metric := range kpiSearchInterface["metrics"].([]interface{}) {
			_metric := metric.(map[string]interface{})
			if _metric["title"].(string) == kpiData["base_search_metric"].(string) {
				itsiKpi["base_search_metric"] = _metric["_key"].(string)
				for _, metricKey := range []string{"aggregate_statop", "entity_statop", "fill_gaps",
					"gap_custom_alert_value", "gap_severity", "gap_severity_color", "gap_severity_color_light",
					"gap_severity_value", "threshold_field", "unit"} {
					itsiKpi[metricKey] = _metric[metricKey]
				}
			}
		}

		if _, ok := itsiKpi["base_search_metric"]; !ok {
			return nil, errors.New(kpi.(map[string]interface{})["base_search_metric"].(string) + " metric not found")
		}

		itsiKpi["_key"] = kpiData["id"]

		if thresholdTemplateId, ok := kpiData["threshold_template_id"]; ok && thresholdTemplateId != "" {
			thresholdRestKey := thresholdTemplateId.(string)
			thresholdTemplateBase := kpiThresholdTemplateBase(clientConfig, thresholdRestKey, thresholdRestKey)

			thresholdTemplateBase, err = thresholdTemplateBase.Find(ctx)
			if err != nil {
				return nil, err
			}
			if thresholdTemplateBase == nil {
				return nil, fmt.Errorf("KPI Threshold Template %s not found", thresholdRestKey)
			}

			thresholdTemplateInterface, err := thresholdTemplateBase.RawJson.ToInterfaceMap()
			if err != nil {
				return nil, err
			}

			itsiKpi["kpi_threshold_template_id"] = thresholdRestKey
			for _, thresholdKey := range []string{"time_variate_thresholds", "adaptive_thresholds_is_enabled",
				"adaptive_thresholding_training_window", "aggregate_thresholds", "entity_thresholds",
				"time_variate_thresholds_specification"} {
				if value, ok := thresholdTemplateInterface[thresholdKey]; ok {
					itsiKpi[thresholdKey] = value
				}
			}

			//populate training data from cache
			id := kpiData["id"].(string)
			if _, ok := thresholdValueCache[id]; ok {
				//TODO: move to function & similar parsing in thresholdValueCache population
				if currentThresholdCache, ok := thresholdValueCache[id][thresholdRestKey]; ok {
					timeVariateThresholdsSpecification := itsiKpi["time_variate_thresholds_specification"].(map[string]interface{})
					for policyName, policy := range timeVariateThresholdsSpecification["policies"].(map[string]interface{}) {
						_policy := policy.(map[string]interface{})
						if aggregate_thresholds, ok := _policy["aggregate_thresholds"].(map[string]interface{}); ok {
							for _, threshold_level := range aggregate_thresholds["thresholdLevels"].([]interface{}) {
								_threshold_level := threshold_level.(map[string]interface{})
								severityValue := _threshold_level["severityValue"].(float64)
								dynamicParam := _threshold_level["dynamicParam"].(float64)
								key := policyName + fmt.Sprint(severityValue) + "_" + fmt.Sprint(dynamicParam)
								_threshold_level["thresholdValue"] = currentThresholdCache[key]

								itsiKpi["time_variate_thresholds_specification"] = timeVariateThresholdsSpecification
							}
						}
					}
				}
			}
		} else if customThreshold, ok := kpiData["custom_threshold"]; ok {
			for _, currentCustomThreshold := range customThreshold.(*schema.Set).List() {
				customThresholdData := currentCustomThreshold.(map[string]interface{})

				aggregateThresholds :=
					customThresholdData["aggregate_thresholds"].(*schema.Set).List()[0].(map[string]interface{})
				entityThresholds :=
					customThresholdData["entity_thresholds"].(*schema.Set).List()[0].(map[string]interface{})

				itsiKpi["aggregate_thresholds"], err = kpiThresholdThresholdSettingsToPayload(aggregateThresholds)
				if err != nil {
					return nil, err
				}
				itsiKpi["entity_thresholds"], err = kpiThresholdThresholdSettingsToPayload(entityThresholds)
				if err != nil {
					return nil, err
				}
			}
		}

		itsiKpis = append(itsiKpis, itsiKpi)
	}

	body["kpis"] = itsiKpis

	//entity rules
	itsiEntityRules := []map[string]interface{}{}
	for _, entityRuleGroup := range d.Get("entity_rules").(*schema.Set).List() {
		itsiEntityGroupRules := []map[string]string{}
		if _, ok := entityRuleGroup.(map[string]interface{})["rule"]; !ok {
			continue
		}

		for _, entityRule := range entityRuleGroup.(map[string]interface{})["rule"].(*schema.Set).List() {
			itsiEntityGroupRules = append(itsiEntityGroupRules, map[string]string{
				"field":      entityRule.(map[string]interface{})["field"].(string),
				"field_type": entityRule.(map[string]interface{})["field_type"].(string),
				"rule_type":  entityRule.(map[string]interface{})["rule_type"].(string),
				"value":      entityRule.(map[string]interface{})["value"].(string)})
		}

		itsiEntityRuleGroup := map[string]interface{}{"rule_condition": "AND", "rule_items": itsiEntityGroupRules}
		itsiEntityRules = append(itsiEntityRules, itsiEntityRuleGroup)
	}
	body["entity_rules"] = itsiEntityRules

	//service depends on
	itsiServicesDependsOn := []map[string]interface{}{}
	for _, itsiServiceDependsOn := range d.Get("service_depends_on").(*schema.Set).List() {
		s := itsiServiceDependsOn.(map[string]interface{})
		dependsOnKPIs := s["kpis"].(*schema.Set).List()

		//Bandaid for the terraform SDK glitch
		//when d.Get("service_depends_on") might contain an unexpected empty element
		if len(dependsOnKPIs) == 0 {
			continue
		}

		dependsOnItem := map[string]interface{}{
			"serviceid":         s["service"],
			"kpis_depending_on": dependsOnKPIs,
		}

		overloaded_urgencies, err := unpackMap[int](s["overloaded_urgencies"].(map[string]interface{}))
		if err != nil {
			return nil, err
		}
		if len(overloaded_urgencies) > 0 {
			dependsOnItem["overloaded_urgencies"] = overloaded_urgencies
		}

		itsiServicesDependsOn = append(itsiServicesDependsOn, dependsOnItem)
	}
	body["services_depends_on"] = itsiServicesDependsOn

	//tags
	var serviceTags []string
	for _, tag := range d.Get("tags").(*schema.Set).List() {
		serviceTags = append(serviceTags, tag.(string))
	}
	if len(serviceTags) > 0 {
		body["service_tags"] = map[string][]string{"tags": serviceTags}
	}

	err = base.PopulateRawJSON(ctx, body)
	return base, err
}

func getKpiHashKey(kpiData map[string]interface{}) (string, error) {
	baseSearchId := kpiData["base_search_id"].(string)
	baseSearchMetricId := kpiData["base_search_metric"].(string)

	if baseSearchId == "" || baseSearchMetricId == "" {
		return "", fmt.Errorf("no base search data specified, smt went wrong: %s", kpiData)
	}

	hash := sha1.New()
	hash.Write([]byte(baseSearchId + "_" + baseSearchMetricId))
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func serviceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	template, err := service(ctx, d, m.(models.ClientConfig))
	if err != nil {
		return diag.FromErr(err)
	}
	b, err := template.Create(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	b, err = b.Read(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	return populateServiceResourceData(ctx, b, d)
}

func serviceRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	base := serviceBase(m.(models.ClientConfig), d.Id(), d.Get("title").(string))
	b, err := base.Find(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	if b == nil {
		d.SetId("")
		return nil
	}
	return populateServiceResourceData(ctx, b, d)
}

func populateServiceResourceData(ctx context.Context, b *models.Base, d *schema.ResourceData) (diags diag.Diagnostics) {
	interfaceMap, err := b.RawJson.ToInterfaceMap()
	if err != nil {
		return diag.FromErr(err)
	}

	if err = d.Set("title", interfaceMap["title"]); err != nil {
		return diag.FromErr(err)
	}

	if err = d.Set("description", interfaceMap["description"]); err != nil {
		return diag.FromErr(err)
	}

	if isHealthscoreCalculateByEntityEnabled, ok := interfaceMap["is_healthscore_calculate_by_entity_enabled"]; ok {
		if err = d.Set("is_healthscore_calculate_by_entity_enabled", (int(isHealthscoreCalculateByEntityEnabled.(float64)) == 1)); err != nil {
			return diag.FromErr(err)
		}
	}

	if err = d.Set("enabled", (int(interfaceMap["enabled"].(float64)) != 0)); err != nil {
		return diag.FromErr(err)
	}

	if err = d.Set("security_group", interfaceMap["sec_grp"]); err != nil {
		return diag.FromErr(err)
	}

	//entity_rules
	tfEntityRuleGroups := []interface{}{}
	for _, itsiEntityRuleGroup := range interfaceMap["entity_rules"].([]interface{}) {
		tfRuleItems := []map[string]interface{}{}
		itsiRuleItems, ok := itsiEntityRuleGroup.(map[string]interface{})["rule_items"]
		if !ok {
			continue
		}

		for _, itsiRule := range itsiRuleItems.([]interface{}) {
			r := itsiRule.(map[string]interface{})
			tfRuleItems = append(tfRuleItems, map[string]interface{}{"field": r["field"], "field_type": r["field_type"], "rule_type": r["rule_type"], "value": r["value"]})
		}
		tfEntityRuleGroups = append(tfEntityRuleGroups, map[string]interface{}{"rule": tfRuleItems})
	}
	if err = d.Set("entity_rules", tfEntityRuleGroups); err != nil {
		return diag.FromErr(err)
	}

	//services_depends_on
	tfServicesDependsOn := []interface{}{}
	if _, ok := interfaceMap["services_depends_on"].([]interface{}); ok {
		for _, itsiServiceDependsOn := range interfaceMap["services_depends_on"].([]interface{}) {
			s := itsiServiceDependsOn.(map[string]interface{})
			dependsOnItem := map[string]interface{}{"service": s["serviceid"], "kpis": s["kpis_depending_on"]}
			if overloadedUrgencies, hasOverloadedUrgencies := s["overloaded_urgencies"]; hasOverloadedUrgencies {
				dependsOnItem["overloaded_urgencies"] = overloadedUrgencies
			}
			tfServicesDependsOn = append(tfServicesDependsOn, dependsOnItem)
		}
		if err = d.Set("service_depends_on", tfServicesDependsOn); err != nil {
			return diag.FromErr(err)
		}
	}

	//tags
	if serviceTags, ok := interfaceMap["service_tags"]; ok {
		if tags, ok := serviceTags.(map[string]interface{})["tags"]; ok {
			err = d.Set("tags", tags)
		}
	}
	if err != nil {
		return diag.FromErr(err)
	}

	//Computed fields

	//shkpi_id & kpis
	tfKpis := []interface{}{}
	metricLookup := new(KPIBSMetricLookup)

	if _, ok := interfaceMap["kpis"].([]interface{}); ok {
		for _, kpi := range interfaceMap["kpis"].([]interface{}) {
			k := kpi.(map[string]interface{})
			tfKpi := map[string]interface{}{}
			if id, ok := k["_key"]; ok {
				for _, key := range []string{"title", "base_search_id", "search_type", "type"} {
					tfKpi[key] = k[key]
				}

				if strings.HasPrefix(id.(string), "SHKPI") {
					err := d.Set("shkpi_id", id)
					if err != nil {
						return diag.FromErr(err)
					}
				} else {
					linkedKPIBS := true
					for _, f := range []string{"base_search_id", "base_search_metric"} {
						if v, ok := k[f]; !ok || v == nil {
							linkedKPIBS = false
							diags = append(diags, diag.Diagnostic{
								Severity: diag.Warning,
								Summary:  fmt.Sprintf("Missing base_search_id and base_search_metric fields for Service %s, KPI %s.\nThe itsi_service resource does not support adhoc KPIs.", b.RESTKey, id),
							})
							break
						}
					}
					if !linkedKPIBS {
						// skip populating the adhoc KPI.
						continue
					}
					if tfKpi["base_search_metric"], err = metricLookup.lookupMetricTitleByID(ctx, b.Splunk, k["base_search_id"].(string), k["base_search_metric"].(string)); err != nil {
						diags = append(diags, diag.Diagnostic{
							Severity: diag.Warning,
							Summary:  err.Error(),
						})
						continue
					}
					tfKpi["id"] = id
					if kpiDescription, ok := k["description"]; ok && kpiDescription != "" {
						tfKpi["description"] = kpiDescription
					}
					if kpiThresholdTemplateId, ok := k["kpi_threshold_template_id"]; ok && kpiThresholdTemplateId != "" {
						tfKpi["threshold_template_id"] = kpiThresholdTemplateId
					} else {
						if k["adaptive_thresholds_is_enabled"].(bool) || k["time_variate_thresholds"].(bool) {
							diags = append(diags, diag.Diagnostic{
								Severity: diag.Warning,
								Summary:  fmt.Sprintf("Custom threshold support only static non-time-variate thresholds: serviceId=%s kpiId=%s. Fallback to default", b.RESTKey, id),
							})
							defaultSetting := []map[string]interface{}{
								{
									"base_severity_label": "normal",
									"gauge_max":           1,
									"gauge_min":           0,
									"is_max_static":       false,
									"is_min_static":       false,
									"metric_field":        "",
									"render_boundary_max": 1,
									"render_boundary_min": 0,
									"search":              "",
								},
							}
							tfKpi["custom_threshold"] = []map[string]interface{}{
								{
									"entity_thresholds":    defaultSetting,
									"aggregate_thresholds": defaultSetting,
								},
							}
						} else {
							entityThresholds, err :=
								kpiThresholdSettingsToResourceData(k["entity_thresholds"].(map[string]interface{}), "static")
							if err != nil {
								return diag.FromErr(err)
							}

							aggregateThresholds, err :=
								kpiThresholdSettingsToResourceData(k["aggregate_thresholds"].(map[string]interface{}), "static")
							if err != nil {
								return diag.FromErr(err)
							}
							tfKpi["custom_threshold"] = []map[string]interface{}{
								{
									"entity_thresholds":    entityThresholds,
									"aggregate_thresholds": aggregateThresholds,
								},
							}
						}
					}
					// UI behavior is inconsistent due to urgency field.
					// If urgency was set via the slider - field is numeric
					// otherwise without slider triggering, kpi urgency will equal "5"
					// So provider accepts string as well, but limits schema to integer
					// to keep things consistent to the docs
					switch urgencyType := k["urgency"].(type) {
					// float64, for JSON numbers
					// https://pkg.go.dev/encoding/json#Unmarshal
					case float64:
						tfKpi["urgency"] = k["urgency"]
					case string:
						out, err := strconv.Atoi(k["urgency"].(string))
						if err != nil {
							return diag.FromErr(err)
						}
						tfKpi["urgency"] = out
					default:
						return diag.FromErr(fmt.Errorf("expected a string or an number, got %T", urgencyType))
					}
					tfKpis = append(tfKpis, tfKpi)
				}
			}
		}
	}
	if err = d.Set("kpi", tfKpis); err != nil {
		return diag.FromErr(err)
	}
	//id
	d.SetId(b.RESTKey)
	return
}

func serviceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	clientConfig := m.(models.ClientConfig)
	base := serviceBase(clientConfig, d.Id(), d.Get("title").(string))
	existing, err := base.Find(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	if existing == nil {
		return serviceCreate(ctx, d, m)
	}

	template, err := service(ctx, d, clientConfig)
	if err != nil {
		return diag.FromErr(err)
	}
	return diag.FromErr(template.UpdateAsync(ctx))
}

func serviceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	base := serviceBase(m.(models.ClientConfig), d.Id(), d.Get("title").(string))
	existing, err := base.Find(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	if existing == nil {
		return nil
	}
	return diag.FromErr(existing.Delete(ctx))
}

func serviceImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	b := serviceBase(m.(models.ClientConfig), "", d.Id())
	b, err := b.Find(ctx)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, err
	}
	diags := populateServiceResourceData(ctx, b, d)
	for _, d := range diags {
		if d.Severity == diag.Error {
			return nil, fmt.Errorf(d.Summary)
		}
	}

	if d.Id() == "" {
		return nil, nil
	}
	return []*schema.ResourceData{d}, nil
}
