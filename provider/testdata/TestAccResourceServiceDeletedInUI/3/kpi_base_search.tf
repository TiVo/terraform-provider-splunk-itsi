resource "itsi_kpi_base_search" "test_kpis_deleted_in_ui" {
  title                         = "TestAcc_ResourceServiceDeletedInUI_helper_kpi_bs"
  description                   = "abc"
  actions                       = null
  alert_lag                     = "5"
  alert_period                  = "5"
  base_search                   = <<-EOT
	  | makeresults count=10
	EOT
  entity_alias_filtering_fields = null
  entity_breakdown_id_fields    = "index"
  entity_id_fields              = "pqdn"
  is_entity_breakdown           = true
  is_service_entity_filter      = true
  metric_qualifier              = null
  search_alert_earliest         = "5"
  sec_grp                       = "default_itsi_security_group"
  source_itsi_da                = "itsi"
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
    title                    = "metric 1.1"
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
    title                    = "metric 1.2"
    unit                     = "%"
  }
}
