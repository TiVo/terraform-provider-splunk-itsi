
locals {

  kpis = [for i in range(1, random_integer.n.result) : {
    base_search_metric = "metric${i}"
    title              = "Metric ${i}"
    urgency            = i
  }]

}


resource "random_integer" "n" {
  min = 1
  max = 5

  keepers = {
    timestamp = timestamp()
  }
}


resource "itsi_service" "test" {
  title                                      = "TestAcc_ServiceUnknownKPIs_Service"
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

  dynamic "kpi" {
    for_each = local.kpis
    content {
      base_search_id        = itsi_kpi_base_search.test.id
      base_search_metric    = kpi.value.base_search_metric
      threshold_template_id = itsi_kpi_threshold_template.test.id
      description           = null
      title                 = kpi.value.title
      search_type           = "shared_base"
      urgency               = kpi.value.urgency

    }
  }

}

