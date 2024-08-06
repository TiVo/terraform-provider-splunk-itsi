resource "itsi_service" "test_kpis_2" {
  title                                      = "TestAcc_ResourceServiceKpisHandleUnknownTemplateId_Test_service_kpis_2"
  description                                = null
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
    base_search_id        = itsi_kpi_base_search.test_kpis_linked_kpibs_3.id
    base_search_metric    = "metric 1.1"
    threshold_template_id = itsi_kpi_threshold_template.test_kpis_kpi_threshold_template_3.id
    description           = null
    title                 = "KPI 1"
    search_type           = "shared_base"
    urgency               = 4
  }
  kpi {
    base_search_id        = itsi_kpi_base_search.test_kpis_linked_kpibs_3.id
    base_search_metric    = "metric 1.2"
    description           = null
    threshold_template_id = itsi_kpi_threshold_template.test_kpis_kpi_threshold_template_3.id
    title                 = "KPI 2"
    search_type           = "shared_base"
    urgency               = 7
  }
  kpi {
    base_search_id        = itsi_kpi_base_search.test_kpis_linked_kpibs_4.id
    base_search_metric    = "metric 2.1"
    threshold_template_id = itsi_kpi_threshold_template.test_kpis_kpi_threshold_template_3.id
    title                 = "KPI 3"
    search_type           = "shared_base"
    urgency               = 8
    description           = null

  }

}

