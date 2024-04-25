package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

func TestResourceKPIBaseSearchSchema(t *testing.T) {
	testResourceSchema(t, new(resourceKpiBaseSearch))
}

func TestResourceKPIBaseSearchSchemaPlan(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
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

					resource "itsi_kpi_base_search" "test" {
						actions                       = null
						alert_lag                     = "5"
						alert_period                  = "5"
						base_search                   = <<-EOT
							| makeresults count=10
						EOT

						description                   = null
						entity_alias_filtering_fields = null
						entity_breakdown_id_fields    = "index"
						entity_id_fields              = "pqdn"
						is_entity_breakdown           = true
						is_service_entity_filter      = true
						metric_qualifier              = null
						search_alert_earliest         = "5"
						sec_grp                       = "default_itsi_security_group"
						source_itsi_da                = "itsi"
						title                         = "example base search"
						metrics {
							aggregate_statop         = "sum"
							entity_statop            = "sum"
							fill_gaps                = "null_value"
							gap_custom_alert_value   = 0
							gap_severity             = "unknown"
							gap_severity_color       = "#CCCCCC"
							gap_severity_color_light = "#EEEEEE"
							gap_severity_value       = "-1"
							threshold_field          = "count"
							title                    = "metric 1"
							unit                     = ""
						}
						metrics {
							aggregate_statop         = "sum"
							entity_statop            = "sum"
							fill_gaps                = "null_value"
							gap_custom_alert_value   = 0
							gap_severity             = "unknown"
							gap_severity_color       = "#CCCCCC"
							gap_severity_color_light = "#EEEEEE"
							gap_severity_value       = "-1"
							threshold_field          = "percent_increase"
							title                    = "metric 2"
							unit                     = "%"
						}
					}
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
