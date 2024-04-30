resource "itsi_kpi_threshold_template" "test_kpis_kpi_threshold_template_1" {
  title                                 = "TestAcc_stdev_test_linked_kpi_threshold_template_1"
  description                           = "stdev"
  adaptive_thresholds_is_enabled        = true
  adaptive_thresholding_training_window = "-7d"
  time_variate_thresholds               = true
  sec_grp                               = "default_itsi_security_group"
  time_variate_thresholds_specification {
    policies {
      policy_name = "default_policy"
      title       = "Default"
      policy_type = "stdev"
      aggregate_thresholds {
        base_severity_label = "normal"
        metric_field        = ""
        is_max_static       = false
        gauge_min           = -4
        gauge_max           = 4
        render_boundary_min = -4
        render_boundary_max = 4
        is_min_static       = false
        threshold_levels {
          severity_label  = "critical"
          dynamic_param   = 2.75
          threshold_value = 2.75
        }
        threshold_levels {
          severity_label  = "high"
          dynamic_param   = 2.25
          threshold_value = 2.25
        }
        threshold_levels {
          severity_label  = "medium"
          dynamic_param   = 1.75
          threshold_value = 1.75
        }
        threshold_levels {
          severity_label  = "low"
          dynamic_param   = 1.25
          threshold_value = 1.25
        }
      }

      entity_thresholds {
        base_severity_label = "normal"
        gauge_max           = 100
        gauge_min           = 0
        is_max_static       = false
        is_min_static       = false
        metric_field        = ""
        render_boundary_max = 100
        render_boundary_min = 0
      }
    }
    policies {
      policy_name = "0-1"
      title       = "test"
      policy_type = "stdev"
      aggregate_thresholds {
        base_severity_label = "normal"
        metric_field        = ""
        is_max_static       = false
        gauge_min           = -4
        gauge_max           = 4
        render_boundary_min = -4
        render_boundary_max = 4
        is_min_static       = false
        threshold_levels {
          severity_label  = "critical"
          dynamic_param   = 2.75
          threshold_value = 2.75
        }
        threshold_levels {
          severity_label  = "high"
          dynamic_param   = 2.25
          threshold_value = 2.25
        }
        threshold_levels {
          severity_label  = "medium"
          dynamic_param   = 1.75
          threshold_value = 1.75
        }
        threshold_levels {
          severity_label  = "low"
          dynamic_param   = 1.25
          threshold_value = 1.25
        }
      }

      entity_thresholds {
        base_severity_label = "normal"
        gauge_max           = 100
        gauge_min           = 0
        is_max_static       = false
        is_min_static       = false
        metric_field        = ""
        render_boundary_max = 100
        render_boundary_min = 0
      }
    }
  }
}

resource "itsi_kpi_threshold_template" "test_kpis_static" {
  adaptive_thresholding_training_window = "-7d"
  adaptive_thresholds_is_enabled        = false
  description                           = "kpi_threshold_template_1"
  sec_grp                               = "default_itsi_security_group"
  time_variate_thresholds               = true
  title                                 = "[Custom]: Static test kpi_threshold_template_1"
  time_variate_thresholds_specification {
    policies {
      policy_name = "default_policy"
      policy_type = "static"
      title       = "Default"
      aggregate_thresholds {
        base_severity_label = "normal"
        gauge_max           = 600
        gauge_min           = 0
        is_max_static       = true
        is_min_static       = true
        metric_field        = null
        render_boundary_max = 600
        render_boundary_min = 0
        threshold_levels {
          dynamic_param   = 0
          severity_label  = "critical"
          threshold_value = 500
        }
        threshold_levels {
          dynamic_param   = 0
          severity_label  = "normal"
          threshold_value = 0
        }
      }
      entity_thresholds {
        base_severity_label = "normal"
        gauge_max           = 0
        gauge_min           = 0
        is_max_static       = true
        is_min_static       = true
        metric_field        = null
        render_boundary_max = 0
        render_boundary_min = 0
      }
    }
  }
}