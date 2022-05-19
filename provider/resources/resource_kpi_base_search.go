package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

var validateStaStop schema.SchemaValidateFunc = validation.Any(
	validation.StringInSlice([]string{"avg", "count", "dc", "earliest", "latest", "max", "median", "min", "stdev", "sum"}, false),
	validation.StringMatch(regexp.MustCompile(`^perc\d{1,2}$`), ""),
)

func kpiBSTFFormat(b *models.Base) (string, error) {
	res := ResourceKPIBaseSearch()
	resData := res.Data(nil)
	d := populateBaseSearchResourceData(context.Background(), b, resData)
	if len(d) > 0 {
		err := d[0].Validate()
		if err != nil {
			return "", err
		}
		return "", errors.New(d[0].Summary)
	}
	resourcetpl, err := NewResourceTemplate(resData, res.Schema, "title", "itsi_kpi_base_search")
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

func kpiBaseSearchBase(clientConfig models.ClientConfig, key string, title string) *models.Base {
	base := models.NewBase(clientConfig, key, title, "kpi_base_search")
	return base
}

func ResourceKPIBaseSearch() *schema.Resource {
	return &schema.Resource{
		CreateContext: kpiBaseSearchCreate,
		ReadContext:   kpiBaseSearchRead,
		UpdateContext: kpiBaseSearchUpdate,
		DeleteContext: kpiBaseSearchDelete,
		Importer: &schema.ResourceImporter{
			StateContext: kpiBaseSearchImport,
		},
		Schema: map[string]*schema.Schema{
			// "_key": {
			// 	Type:         schema.TypeString,
			// 	Optional:     true,
			// 	InputDefault: "",
			// },
			"title": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of this KPI base search.",
			},
			"description": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "General description for this KPI base search.",
			},
			"actions": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "?",
				Default:     "",
			},
			"alert_lag": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Contains the number of seconds of lag to apply to the alert search, max is 30 minutes (1799 seconds).",
			},
			"alert_period": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "User specified interval to run the KPI search in minutes.",
			},
			"base_search": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "KPI search defined by user for this KPI. All generated searches for the KPI are based on this search.",
			},
			"entity_alias_filtering_fields": {
				Type:        schema.TypeString,
				Required:    false,
				Optional:    true,
				Description: "Fields from this KPI's search events that will be mapped to the alias fields defined in entities for the service containing this KPI. This field enables the KPI search to tie the aliases of entities to the fields from the KPI events in identifying entities at search time.",
			},
			"entity_breakdown_id_fields": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "KPI search events are split by the alias field defined in entities for the service containing this KPI",
			},
			"entity_id_fields": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Fields from this KPI's search events that will be mapped to the alias fields defined in entities for the service containing this KPI. This field enables the KPI search to tie the aliases of entities to the fields from the KPI events in identifying entities at search time.",
			},
			"is_entity_breakdown": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Determines if search breaks down by entities. See KPI definition.",
			},
			"is_service_entity_filter": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "If true a filter is used on the search based on the entities included in the service.",
			},
			"metric_qualifier": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Used to further split metrics. Hidden in the UI.",
			},
			"metrics": {
				Required: true,
				Type:     schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "generated metric _key",
						},
						"aggregate_statop": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Statistical operation (avg, max, median, stdev, and so on) used to combine data for the aggregate alert_value (used for all KPI).",
							ValidateFunc: validateStaStop,
						},
						"entity_statop": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Statistical operation (avg, max, mean, and so on) used to combine data for alert_values on a per entity basis (used if entity_breakdown is true).",
							ValidateFunc: validateStaStop,
						},
						"fill_gaps": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "How to fill missing data",
							ValidateFunc: validation.StringInSlice([]string{"null_value", "last_available_value", "custom_value"}, false),
						},
						"gap_custom_alert_value": {
							Type:     schema.TypeFloat,
							Optional: true,
							Default:  0,
						},
						"gap_severity": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "unknown",
						},
						"gap_severity_color": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "#CCCCCC",
						},
						"gap_severity_color_light": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "#EEEEEE",
						},
						"gap_severity_value": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "-1",
						},
						"threshold_field": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The field on which the statistical operation runs",
						},
						"title": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name of this metric",
						},
						"unit": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "User-defined units for the values in threshold field.",
						},
					},
				},
			},
			"search_alert_earliest": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Value in minutes. This determines how far back each time window is during KPI search runs.",
			},
			"sec_grp": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The team the object belongs to. ",
			},
			"source_itsi_da": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Source of DA used for this search. See KPI Threshold Templates.",
			},
		},
	}
}

func metric(source map[string]interface{}) interface{} {
	m := map[string]interface{}{}
	m["_key"] = source["id"].(string)
	m["aggregate_statop"] = source["aggregate_statop"]
	m["entity_statop"] = source["entity_statop"]
	m["fill_gaps"] = source["fill_gaps"]
	m["gap_custom_alert_value"] = source["gap_custom_alert_value"]
	m["gap_severity"] = source["gap_severity"]
	m["gap_severity_color"] = source["gap_severity_color"]
	m["gap_severity_color_light"] = source["gap_severity_color_light"]
	m["gap_severity_value"] = source["gap_severity_value"]
	m["threshold_field"] = source["threshold_field"]
	m["title"] = source["title"]
	m["unit"] = source["unit"]
	return m
}

func kpiBaseSearch(d *schema.ResourceData, clientConfig models.ClientConfig) (config *models.Base, err error) {
	body := map[string]interface{}{}
	body["objectType"] = "kpi_base_search"
	body["title"] = d.Get("title").(string)
	body["description"] = d.Get("description").(string)
	body["actions"] = d.Get("actions").(string)
	body["alert_lag"] = d.Get("alert_lag").(string)
	body["alert_period"] = d.Get("alert_period").(string)
	body["base_search"] = d.Get("base_search").(string)
	body["entity_alias_filtering_fields"] = d.Get("entity_alias_filtering_fields").(string)
	body["entity_breakdown_id_fields"] = d.Get("entity_breakdown_id_fields").(string)
	body["entity_id_fields"] = d.Get("entity_id_fields").(string)
	body["is_entity_breakdown"] = d.Get("is_entity_breakdown").(bool)
	body["is_service_entity_filter"] = d.Get("is_service_entity_filter").(bool)
	body["metric_qualifier"] = d.Get("metric_qualifier").(string)

	if d.HasChange("metrics") {
		metricsOld, metricsNew := d.GetChange("metrics")
		metricsOldbyId := map[string]map[string]interface{}{}

		for _, metrics := range metricsOld.(*schema.Set).List() {
			metricsData := metrics.(map[string]interface{})
			id := metricsData["id"].(string)
			metricsOldbyId[id] = metricsData
		}
		for _, metrics := range metricsNew.(*schema.Set).List() {
			metricsData := metrics.(map[string]interface{})
			id, ok := metricsData["id"]
			if ok && id != "" {
				delete(metricsOldbyId, id.(string))
			}
		}
		metricsOldbyTitle := map[string]map[string]interface{}{}
		for _, metricsData := range metricsOldbyId {
			metricsOldbyTitle[metricsData["title"].(string)] = metricsData
		}
		for _, metrics := range metricsNew.(*schema.Set).List() {
			metricsData := metrics.(map[string]interface{})
			id, ok := metricsData["id"]
			if !ok || id == "" {
				title := metricsData["title"].(string)
				if oldMetric, ok := metricsOldbyTitle[title]; ok {
					metricsData["id"] = oldMetric["id"]
					delete(metricsOldbyTitle, title)
				} else {
					metricsData["id"], _ = uuid.GenerateUUID()
				}
			}
		}
		err := d.Set("metrics", metricsNew)
		if err != nil {
			return nil, err
		}
	}

	metrics := []interface{}{}
	for _, g := range d.Get("metrics").(*schema.Set).List() {
		metrics = append(metrics, metric(g.(map[string]interface{})))
		if err != nil {
			return nil, err
		}
	}
	body["metrics"] = metrics
	body["search_alert_earliest"] = d.Get("search_alert_earliest").(string)
	body["sec_grp"] = d.Get("sec_grp").(string)
	body["source_itsi_da"] = d.Get("source_itsi_da").(string)

	by, err := json.Marshal(body)
	if err != nil {
		return
	}
	base := kpiBaseSearchBase(clientConfig, d.Id(), d.Get("title").(string))
	err = json.Unmarshal(by, &base.RawJson)
	if err != nil {
		return nil, err
	}
	return base, nil
}

func kpiBaseSearchCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	clientConfig := m.(models.ClientConfig)
	template, err := kpiBaseSearch(d, clientConfig)
	if err != nil {
		return diag.FromErr(err)
	}
	b, err := template.Create(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	b.Read(ctx)
	return populateBaseSearchResourceData(ctx, b, d)
}

func kpiBaseSearchRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	base := kpiBaseSearchBase(m.(models.ClientConfig), d.Id(), d.Get("title").(string))
	b, err := base.Find(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	if b == nil {
		d.SetId("")
		return nil
	}
	return populateBaseSearchResourceData(ctx, b, d)
}

func populateBaseSearchResourceData(ctx context.Context, b *models.Base, d *schema.ResourceData) (diags diag.Diagnostics) {
	by, err := b.RawJson.MarshalJSON()
	if err != nil {
		return diag.FromErr(err)
	}
	var interfaceMap map[string]interface{}
	err = json.Unmarshal(by, &interfaceMap)
	if err != nil {
		return diag.FromErr(err)
	}

	for tfField, itsiField := range map[string]string{
		"title":                         "title",
		"description":                   "description",
		"actions":                       "actions",
		"alert_lag":                     "alert_lag",
		"alert_period":                  "alert_period",
		"base_search":                   "base_search",
		"entity_alias_filtering_fields": "entity_alias_filtering_fields",
		"entity_breakdown_id_fields":    "entity_breakdown_id_fields",
		"entity_id_fields":              "entity_id_fields",
		"is_entity_breakdown":           "is_entity_breakdown",
		"is_service_entity_filter":      "is_service_entity_filter",
		"metric_qualifier":              "metric_qualifier",
		"search_alert_earliest":         "search_alert_earliest",
		"sec_grp":                       "sec_grp",
		"source_itsi_da":                "source_itsi_da",
	} {
		if err = d.Set(tfField, interfaceMap[itsiField]); err != nil {
			return diag.FromErr(err)
		}
	}

	//metrics
	metrics := []interface{}{}
	for _, metricsData_ := range interfaceMap["metrics"].([]interface{}) {
		metricsData := metricsData_.(map[string]interface{})
		m := map[string]interface{}{}
		m["id"] = metricsData["_key"]
		m["aggregate_statop"] = metricsData["aggregate_statop"]
		m["entity_statop"] = metricsData["entity_statop"]
		m["fill_gaps"] = metricsData["fill_gaps"]
		gapCustomAlertValue, err := strconv.ParseFloat(fmt.Sprintf("%v", metricsData["gap_custom_alert_value"]), 64)
		if err != nil {
			return diag.FromErr(err)
		}
		m["gap_custom_alert_value"] = gapCustomAlertValue
		m["gap_severity"] = metricsData["gap_severity"]
		m["gap_severity_color"] = metricsData["gap_severity_color"]
		m["gap_severity_color_light"] = metricsData["gap_severity_color_light"]
		m["gap_severity_value"] = metricsData["gap_severity_value"]
		m["threshold_field"] = metricsData["threshold_field"]
		m["title"] = metricsData["title"]
		m["unit"] = metricsData["unit"]
		metrics = append(metrics, m)
	}
	if err = d.Set("metrics", metrics); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(b.RESTKey)
	return nil
}

func kpiBaseSearchUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	clientConfig := m.(models.ClientConfig)
	base := kpiBaseSearchBase(clientConfig, d.Id(), d.Get("title").(string))
	existing, err := base.Find(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	if existing == nil {
		return kpiBaseSearchCreate(ctx, d, m)
	}

	template, err := kpiBaseSearch(d, clientConfig)
	if err != nil {
		return diag.FromErr(err)
	}
	return diag.FromErr(template.Update(ctx))
}

func kpiBaseSearchDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	base := kpiBaseSearchBase(m.(models.ClientConfig), d.Id(), d.Get("title").(string))
	existing, err := base.Find(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	if existing == nil {
		return nil
	}
	return diag.FromErr(existing.Delete(ctx))
}

func kpiBaseSearchImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	b := kpiBaseSearchBase(m.(models.ClientConfig), "", d.Id())
	b, err := b.Find(ctx)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, err
	}
	diags := populateBaseSearchResourceData(ctx, b, d)
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
