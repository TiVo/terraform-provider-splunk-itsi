resource "itsi_kpi_base_search" "my_kpi_base_search" {
  title                         = "this_is_a_test"
  actions                       = ""
  alert_lag                     = "31"
  alert_period                  = "5"
  base_search                   = "index=_internal source=*metrics.log group=tcpin_connections \n | eval sourceHost=if(isnull(hostname), sourceHost,hostname)"
  description                   = "This is a description for a KPI base search"
  entity_alias_filtering_fields = null
  entity_breakdown_id_fields    = "host"
  entity_id_fields              = "host"
  is_entity_breakdown           = false
  is_service_entity_filter      = false
  metric_qualifier              = ""
  metrics {
    aggregate_statop         = "dc"
    entity_statop            = "avg"
    fill_gaps                = "null_value"
    gap_custom_alert_value   = "0"
    gap_severity             = "unknown"
    gap_severity_color       = "#CCCCCC"
    gap_severity_color_light = "#EEEEEE"
    gap_severity_value       = "-1"
    threshold_field          = "sourceIp"
    title                    = "Forwarder Count"
    unit                     = ""
  }
  search_alert_earliest = "5"
  sec_grp               = "default_itsi_security_group"
  source_itsi_da        = "itsi"
}
