package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

func TestResourceKpiThresholdTemplateSchema(t *testing.T) {
	testResourceSchema(t, new(resourceKpiThresholdTemplate))
}

func TestResourceKpiThresholdTemplatePlan(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: util.Dedent(`
					provider "itsi" {
						host     = "itsi.example.com"
						user     = "user"
						password = "password"
						port     = 8089
						timeout  = 20
					}

					resource "itsi_kpi_threshold_template" "stdev" {
					  title                                 = "Sample time variate standard deviation threshold template"
					  description                           = "stdev"
					  adaptive_thresholds_is_enabled        = true
					  adaptive_thresholding_training_window = "-7d"
					  time_variate_thresholds               = true
					  sec_grp                               = "default_itsi_security_group"
					  time_variate_thresholds_specification {
					    policies {
					      policy_name = "default_policy"
					      title       = "Default"
					      policy_type = "stdev"
					      aggregate_thresholds {
					        base_severity_label = "normal"
					        metric_field        = ""
					        is_max_static       = false
					        gauge_min           = -4
					        gauge_max           = 4
					        render_boundary_min = -4
					        render_boundary_max = 4
					        is_min_static       = false
					        threshold_levels {
					          severity_label  = "critical"
					          dynamic_param   = 2.75
					          threshold_value = 2.75
					        }
					        threshold_levels {
					          severity_label  = "high"
					          dynamic_param   = 2.25
					          threshold_value = 2.25
					        }
					        threshold_levels {
					          severity_label  = "medium"
					          dynamic_param   = 1.75
					          threshold_value = 1.75
					        }
					        threshold_levels {
					          severity_label  = "low"
					          dynamic_param   = 1.25
					          threshold_value = 1.25
					        }
					      }

					      entity_thresholds {
					        base_severity_label = "normal"
					        gauge_max           = 100
					        gauge_min           = 0
					        is_max_static       = false
					        is_min_static       = false
					        metric_field        = ""
					        render_boundary_max = 100
					        render_boundary_min = 0
					      }
					    }
					    policies {
					      policy_name = "0-1"
					      title       = "test"
					      policy_type = "stdev"
					      aggregate_thresholds {
					        base_severity_label = "normal"
					        metric_field        = ""
					        is_max_static       = false
					        gauge_min           = -4
					        gauge_max           = 4
					        render_boundary_min = -4
					        render_boundary_max = 4
					        is_min_static       = false
					        threshold_levels {
					          severity_label  = "critical"
					          dynamic_param   = 2.75
					          threshold_value = 2.75
					        }
					        threshold_levels {
					          severity_label  = "high"
					          dynamic_param   = 2.25
					          threshold_value = 2.25
					        }
					        threshold_levels {
					          severity_label  = "medium"
					          dynamic_param   = 1.75
					          threshold_value = 1.75
					        }
					        threshold_levels {
					          severity_label  = "low"
					          dynamic_param   = 1.25
					          threshold_value = 1.25
					        }
					      }

					      entity_thresholds {
					        base_severity_label = "normal"
					        gauge_max           = 100
					        gauge_min           = 0
					        is_max_static       = false
					        is_min_static       = false
					        metric_field        = ""
					        render_boundary_max = 100
					        render_boundary_min = 0
					      }
					    }
					  }
					}

				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
