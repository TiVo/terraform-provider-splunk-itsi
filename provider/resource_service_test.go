package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

func TestResourceServiceSchema(t *testing.T) {
	testResourceSchema(t, new(resourceService))
}

func TestResourceServicePlan(t *testing.T) {
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

					resource "itsi_service" "Test-custom-static-threshold" {
					  enabled = true
					  entity_rules {
					    rule {
					      field      = "host"
					      field_type = "alias"
					      rule_type  = "matches"
					      value      = "example.com"
					    }
					  }
					  kpi {
					    base_search_id     = "625f502d7e6e1a37ea062eff"
					    base_search_metric = "host_count"
					    search_type           = "shared_base"
					    threshold_template_id = "123"
					    title                 = "Test custom static threshold KPI 1"
					    type                  = "kpis_primary"
					    urgency               = 5
					  }
					  security_group = "default_itsi_security_group"
					  title          = "Test custom static threshold"
					}
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
