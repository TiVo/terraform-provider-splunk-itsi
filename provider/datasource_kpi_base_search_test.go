package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tivo/terraform-provider-splunk-itsi/util"
)

func TestDataSourceKPIBaseSearchSchema(t *testing.T) {
	testDataSourceSchema(t, new(dataSourceKpiBaseSearch))
}

// TestDataSourceKPIBaseSearchValidation verifies schema-level validation errors
// without requiring a live server.
func TestDataSourceKPIBaseSearchValidation(t *testing.T) {
	providerBlock := `
		provider "itsi" {
			host     = "itsi.example.com"
			user     = "user"
			password = "password"
			port     = 8089
			timeout  = 20
		}`

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: util.Dedent(providerBlock + `
					data "itsi_kpi_base_search" "test" {}
				`),
				ExpectError: regexp.MustCompile(`No attribute specified when one \(and only one\) of \[id\] is required`),
			},
			{
				Config: util.Dedent(providerBlock + `
					data "itsi_kpi_base_search" "test" {
						id    = "some_id"
						title = "some_title"
					}
				`),
				ExpectError: regexp.MustCompile(`2 attributes specified when one \(and only one\) of \[id\] is required`),
			},
		},
	})
}

func TestAccDataSourceKPIBaseSearchLifecycle(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckResourceDestroy(resourceNameKPIBaseSearch, "TestAcc_DataSourceKPIBaseSearchLifecycle_test_base_search2"),
		Steps: []resource.TestStep{
			// Step 1: create the resource
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
			},
			// Step 2: lookup by title
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
			// Step 3: lookup by id
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.itsi_kpi_base_search.test", "title", "TestAcc_DataSourceKPIBaseSearchLifecycle_test_base_search2"),
					resource.TestCheckResourceAttr("data.itsi_kpi_base_search.test", "metrics.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("data.itsi_kpi_base_search.test", "metrics.*", map[string]string{
						"title":           "metric 1",
						"threshold_field": "count",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.itsi_kpi_base_search.test", "metrics.*", map[string]string{
						"title":           "metric 2",
						"threshold_field": "percent_increase",
					}),
				),
			},
		},
	})
}

// TestAccDataSourceKPIBaseSearchNotFound verifies that looking up a non-existent
// KPI base search by title or by id produces the expected error.
func TestAccDataSourceKPIBaseSearchNotFound(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: util.Dedent(`
					data "itsi_kpi_base_search" "test" {
						title = "nonexistent_kpi_base_search_title_xyz"
					}
				`),
				ExpectError: regexp.MustCompile(`KPI BS with title .* not found`),
			},
			{
				Config: util.Dedent(`
					data "itsi_kpi_base_search" "test" {
						id = "nonexistent_kpi_base_search_id_xyz"
					}
				`),
				ExpectError: regexp.MustCompile(`KPI BS with id .* not found`),
			},
		},
	})
}
