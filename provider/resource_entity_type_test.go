package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

func TestResourceEntityTypeSchema(t *testing.T) {
	testResourceSchema(t, new(resourceEntityType))
}

func TestResourceEntityTypePlan(t *testing.T) {
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

					resource "itsi_entity_type" "Kubernetes_Pod" {
					  description = "Kubernetes Pod EXAMPLE 2"
					  title       = "Kubernetes Pod EXAMPLE type"

					  data_drilldown {
					    entity_field_filter {
					      data_field   = "pod-name"
					      entity_field = "pod-name"
					    }

					    static_filters = {
					      metric_name = "kube.pod.*"
					      test        = "1234567890"
					    }
					    title = "Kubernetes Pod metrics"
					    type  = "metrics"
					  }

					  data_drilldown {
					    entity_field_filter {
					      data_field   = "namespace"
					      entity_field = "pod-namespace"
					    }
					    entity_field_filter {
					      data_field   = "pod"
					      entity_field = "pod-name"
					    }
					    static_filters = {
					      sourcetype = "kube:objects:pods"
					    }
					    title = "Kubernetes Pod metadata"
					    type  = "events"
					  }


					  vital_metric {
					    is_key = true
					    matching_entity_fields = {
					      pod-name      = "pod-name"
					      pod-namespace = "pod-namespace"
					    }
					    metric_name = "Average CPU Usage2"
					    search      = "| mstats avg(kube.pod.cpu.usage_rate) as val WHERE 1=1 by pod-name, pod-namespace span=5m"
					  }


					  vital_metric {
					    is_key = false
					    matching_entity_fields = {
					      pod-name      = "pod-name"
					      pod-namespace = "pod-namespace"
					    }
					    metric_name = "Average Memory Usage"
					    search      = "| mstats avg(kube.pod.memory.working_set_bytes) as val WHERE  1=1 by pod-name, pod-namespace span=5m"
					    unit        = "Bytes"

					    alert_rule {

					      critical_threshold = 10
					      warning_threshold  = 5
					      cron_schedule      = "*/10 * * * *"
					      is_enabled         = true
					      suppress_time      = 300

					      entity_filter {
					        field      = "test"
					        field_type = "alias"
					        value      = "test_value"
					      }

					      entity_filter {
					        field      = "test2"
					        field_type = "alias"
					        value      = "test_value2"
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
