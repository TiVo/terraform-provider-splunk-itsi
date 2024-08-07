resource "itsi_service" "test_kpis_2" {
  title                                      = "TestAcc_ResourceServiceHandleUnknownKpiBaseSearchId_Test_service_kpis_2"
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

}

