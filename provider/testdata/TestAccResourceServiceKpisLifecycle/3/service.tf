resource "itsi_service" "test_kpis" {
  title                                      = "Test service kpis"
  enabled                                    = true
  is_healthscore_calculate_by_entity_enabled = true
  security_group                             = "default_itsi_security_group"
  tags                                       = null
  entity_rules {
    rule {
      field      = "entity"
      field_type = "alias"
      rule_type  = "matches"
      value      = "TEST"
    }
  }
  kpi {
    base_search_id        = itsi_kpi_base_search.test_kpis_linked_kpibs_1.id
    base_search_metric    = "metric 1.3"
    threshold_template_id = itsi_kpi_threshold_template.test_kpis_kpi_threshold_template_1.id
    title                 = "KPI 1"
    urgency               = 4
  }
  kpi {
    base_search_id        = itsi_kpi_base_search.test_kpis_linked_kpibs_1.id
    base_search_metric    = "metric 1.2"
    threshold_template_id = itsi_kpi_threshold_template.test_kpis_static.id
    title                 = "KPI 2"
    description           = "test"
    type                  = "kpis_primary"
    urgency               = 3
  }
  kpi {
    base_search_id        = itsi_kpi_base_search.test_kpis_linked_kpibs_2.id
    base_search_metric    = "metric 2.1"
    description           = "test"
    threshold_template_id = itsi_kpi_threshold_template.test_kpis_kpi_threshold_template_1.id
    title                 = "KPI 3"
    urgency               = 8
  }


}