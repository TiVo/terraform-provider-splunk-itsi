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
    custom_threshold {
      aggregate_thresholds {
        base_severity_label = "normal"
        gauge_max           = 96.8
        gauge_min           = 0
        is_max_static       = false
        is_min_static       = true
        metric_field        = "count"
        render_boundary_max = 100
        render_boundary_min = 0
        search              = ""
        threshold_levels {
          dynamic_param   = 0
          severity_label  = "medium"
          threshold_value = 75
        }
        threshold_levels {
          dynamic_param   = 0
          severity_label  = "high"
          threshold_value = 88
        }
        threshold_levels {
          dynamic_param   = 0
          severity_label  = "low"
          threshold_value = 50
        }
      }
      entity_thresholds {
        base_severity_label = "normal"
        gauge_max           = 82.5
        gauge_min           = 0
        is_max_static       = false
        is_min_static       = true
        metric_field        = "count"
        render_boundary_max = 100
        render_boundary_min = 0
        search              = ""
        threshold_levels {
          dynamic_param   = 0
          severity_label  = "medium"
          threshold_value = 75
        }
        threshold_levels {
          dynamic_param   = 0
          severity_label  = "low"
          threshold_value = 50
        }
      }
    }
    search_type           = "shared_base"
    threshold_template_id = ""
    title                 = "Test custom static threshold KPI 1"
    type                  = "kpis_primary"
    urgency               = 5
  }
  security_group = "default_itsi_security_group"
  title          = "Test custom static threshold"
}