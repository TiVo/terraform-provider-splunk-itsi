package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDataSourceKPIBaseSearchSchema(t *testing.T) {
	testDataSourceSchema(t, new(dataSourceKpiBaseSearch))
}

func TestAccDataSourceKPIBaseSearchLifecycle(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckResourceDestroy(resourceNameKPIBaseSearch, "TestAcc_DataSourceKPIBaseSearchLifecycle_test_base_search2"),
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.itsi_kpi_base_search.test", "metrics.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("data.itsi_kpi_base_search.test", "metrics.*", map[string]string{
						"title":                    "metric 1",
						"threshold_field":          "count",
						"aggregate_statop":         "sum",
						"entity_statop":            "sum",
						"fill_gaps":                "null_value",
						"unit":                     "",
						"gap_custom_alert_value":   "0",
						"gap_severity":             "unknown",
						"gap_severity_color":       "#CCCCCC",
						"gap_severity_color_light": "#EEEEEE",
						"gap_severity_value":       "-1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.itsi_kpi_base_search.test", "metrics.*", map[string]string{
						"title":                    "metric 2",
						"threshold_field":          "percent_increase",
						"aggregate_statop":         "sum",
						"entity_statop":            "sum",
						"fill_gaps":                "null_value",
						"unit":                     "%",
						"gap_custom_alert_value":   "0",
						"gap_severity":             "unknown",
						"gap_severity_color":       "#CCCCCC",
						"gap_severity_color_light": "#EEEEEE",
						"gap_severity_value":       "-1",
					}),
				),
			},
		},
	})
}
