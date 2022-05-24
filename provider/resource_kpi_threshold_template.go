package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

func kpiThresholdTemplateTFFormat(b *models.Base) (string, error) {
	res := ResourceKPIThresholdTemplate()
	resData := res.Data(nil)
	d := populateKpiThresholdTemplateResourceData(context.Background(), b, resData)
	if len(d) > 0 {
		err := d[0].Validate()
		if err != nil {
			return "", err
		}
		return "", errors.New(d[0].Summary)
	}
	resourcetpl, err := NewResourceTemplate(resData, res.Schema, "title", "itsi_kpi_threshold_template")
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

func kpiThresholdTemplateBase(clientConfig models.ClientConfig, key string, title string) *models.Base {
	base := models.NewBase(clientConfig, key, title, "kpi_threshold_template")
	return base
}

func ResourceKPIThresholdTemplate() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a KPI threshold template.",
		CreateContext: kpiThresholdTemplateCreate,
		ReadContext:   kpiThresholdTemplateRead,
		UpdateContext: kpiThresholdTemplateUpdate,
		DeleteContext: kpiThresholdTemplateDelete,
		Importer: &schema.ResourceImporter{
			StateContext: kpiThresholdTemplateImport,
		},
		Schema: map[string]*schema.Schema{
			"title": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"adaptive_thresholds_is_enabled": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"adaptive_thresholding_training_window": {
				Type:     schema.TypeString,
				Required: true,
			},
			"time_variate_thresholds": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"sec_grp": {
				Type:     schema.TypeString,
				Required: true,
			},
			"time_variate_thresholds_specification": {
				Required: true,
				Type:     schema.TypeSet,
				Elem: &schema.Resource{
					Schema: getKpiThresholdPolicySchema(),
				},
			},
		},
	}
}

func kpiThresholdTemplate(d *schema.ResourceData, clientConfig models.ClientConfig) (config *models.Base, err error) {
	body := map[string]interface{}{}
	body["objectType"] = "kpi_threshold_template"
	body["title"] = d.Get("title").(string)
	body["description"] = d.Get("description").(string)
	body["adaptive_thresholds_is_enabled"] = d.Get("adaptive_thresholds_is_enabled").(bool)
	body["adaptive_thresholding_training_window"] = d.Get("adaptive_thresholding_training_window").(string)
	body["time_variate_thresholds"] = d.Get("time_variate_thresholds").(bool)
	body["sec_grp"] = d.Get("sec_grp").(string)

	timeVariateThresholdsSpecification := map[string]interface{}{}
	for _, sourceTimeVariateSpecification_ := range d.Get("time_variate_thresholds_specification").(*schema.Set).List() {
		sourceTimeVariateSpecification := sourceTimeVariateSpecification_.(map[string]interface{})
		policies := map[string]interface{}{}
		for _, sourcePolicy_ := range sourceTimeVariateSpecification["policies"].(*schema.Set).List() {
			sourcePolicy := sourcePolicy_.(map[string]interface{})
			policyName := sourcePolicy["policy_name"].(string)

			policies[policyName], err = kpiThresholdPolicyToPayload(sourcePolicy)
			if err != nil {
				return nil, err
			}
		}
		timeVariateThresholdsSpecification["policies"] = policies
	}
	body["time_variate_thresholds_specification"] = timeVariateThresholdsSpecification
	by, err := json.Marshal(body)
	if err != nil {
		return
	}
	base := kpiThresholdTemplateBase(clientConfig, d.Id(), d.Get("title").(string))
	err = json.Unmarshal(by, &base.RawJson)
	if err != nil {
		return nil, err
	}
	return base, nil
}

func kpiThresholdTemplateCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	template, err := kpiThresholdTemplate(d, m.(models.ClientConfig))
	if err != nil {
		return diag.FromErr(err)
	}
	b, err := template.Create(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	b.Read(ctx)
	return populateKpiThresholdTemplateResourceData(ctx, b, d)
}

func kpiThresholdTemplateRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	base := kpiThresholdTemplateBase(m.(models.ClientConfig), d.Id(), d.Get("title").(string))
	b, err := base.Find(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	if b == nil {
		d.SetId("")
		return nil
	}
	return populateKpiThresholdTemplateResourceData(ctx, b, d)
}

func populateKpiThresholdTemplateResourceData(ctx context.Context, b *models.Base, d *schema.ResourceData) (diags diag.Diagnostics) {
	by, err := b.RawJson.MarshalJSON()
	if err != nil {
		return diag.FromErr(err)
	}
	var interfaceMap map[string]interface{}
	err = json.Unmarshal(by, &interfaceMap)
	if err != nil {
		return diag.FromErr(err)
	}

	for _, f := range []string{"title", "description", "adaptive_thresholds_is_enabled", "adaptive_thresholding_training_window", "time_variate_thresholds", "sec_grp"} {
		err = d.Set(f, interfaceMap[f])
		if err != nil {
			return diag.FromErr(err)
		}
	}

	timeVariateThresholdsSpecification := []map[string]interface{}{}
	policies := []interface{}{}
	timeVariateThresholdsSpecificationData := interfaceMap["time_variate_thresholds_specification"].(map[string]interface{})
	for policyName, pData := range timeVariateThresholdsSpecificationData["policies"].(map[string]interface{}) {
		policyData := pData.(map[string]interface{})
		policy, err := kpiThresholdPolicyToResourceData(policyData, policyName)
		if err != nil {
			return diag.FromErr(err)
		}
		policies = append(policies, policy)
	}
	policiesMap := map[string]interface{}{
		"policies": policies,
	}
	timeVariateThresholdsSpecification = append(timeVariateThresholdsSpecification, policiesMap)
	err = d.Set("time_variate_thresholds_specification", timeVariateThresholdsSpecification)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(b.RESTKey)
	return
}

func kpiThresholdTemplateUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	clientConfig := m.(models.ClientConfig)
	base := kpiThresholdTemplateBase(clientConfig, d.Id(), d.Get("title").(string))
	existing, err := base.Find(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	if existing == nil {
		return kpiThresholdTemplateCreate(ctx, d, m)
	}

	template, err := kpiThresholdTemplate(d, clientConfig)
	if err != nil {
		return diag.FromErr(err)
	}
	return diag.FromErr(template.Update(ctx))
}

func kpiThresholdTemplateDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	base := kpiThresholdTemplateBase(m.(models.ClientConfig), d.Id(), d.Get("title").(string))
	existing, err := base.Find(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	if existing == nil {
		return nil
	}
	return diag.FromErr(existing.Delete(ctx))
}

func kpiThresholdTemplateImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	b := kpiThresholdTemplateBase(m.(models.ClientConfig), "", d.Id())
	b, err := b.Find(ctx)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, err
	}
	diags := populateKpiThresholdTemplateResourceData(ctx, b, d)
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
