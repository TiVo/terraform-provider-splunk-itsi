resource "itsi_service" "test_ui_delete" {
  title                                      = "TestAcc_ResourceServiceDeletedInUI_service"
  description                                = ""
  enabled                                    = true
  is_healthscore_calculate_by_entity_enabled = true
  security_group                             = "default_itsi_security_group"
  tags                                       = ["custom_shs"]
  entity_rules {
    rule {
      field      = "entity"
      field_type = "alias"
      rule_type  = "matches"
      value      = "TEST"
    }
  }
  kpi {
    base_search_id        = itsi_kpi_base_search.test_kpis_deleted_in_ui.id
    base_search_metric    = "metric 1.1"
    threshold_template_id = itsi_kpi_threshold_template.test_kpis_kpi_threshold_template_deleted_in_ui.id
    description           = null
    title                 = "KPI 1"
    search_type           = "shared_base"
    urgency               = 4
  }
  kpi {
    base_search_id        = itsi_kpi_base_search.test_kpis_deleted_in_ui.id
    base_search_metric    = "metric 1.2"
    description           = null
    threshold_template_id = itsi_kpi_threshold_template.test_kpis_kpi_threshold_template_deleted_in_ui.id
    title                 = "KPI 2"
    search_type           = "shared_base"
    urgency               = 7
  }

}

