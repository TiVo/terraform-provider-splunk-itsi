# Sample time variate standard deviation threshold template

resource "itsi_kpi_threshold_template" "stdev" {
  title                                 = "Sample time variate standard deviation threshold template"
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
        search              = ""
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
        search              = ""
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
        search              = ""
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
        search              = ""
      }
    }
  }
}

# Sample time variate static threshold template

resource "itsi_kpi_threshold_template" "static" {
  title                                 = "Sample time variate static threshold template"
  description                           = "static"
  adaptive_thresholds_is_enabled        = true
  adaptive_thresholding_training_window = "-7d"
  time_variate_thresholds               = true
  sec_grp                               = "default_itsi_security_group"

  time_variate_thresholds_specification {

    policies {
      policy_name = "default_policy"
      title       = "Default"
      policy_type = "static"
      aggregate_thresholds {
        base_severity_label = "normal"
        metric_field        = ""
        is_max_static       = false
        gauge_min           = -4
        gauge_max           = 4
        render_boundary_min = -4
        render_boundary_max = 4
        search              = ""
        is_min_static       = false
        threshold_levels {
          severity_label  = "critical"
          threshold_value = 4
        }
        threshold_levels {
          severity_label  = "high"
          threshold_value = 3
        }
        threshold_levels {
          severity_label  = "medium"
          threshold_value = 2
        }
        threshold_levels {
          severity_label  = "low"
          threshold_value = 1
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
        search              = ""
      }
    }

    policies {
      policy_name = "0-1"
      title       = "test"
      policy_type = "static"
      aggregate_thresholds {
        base_severity_label = "normal"
        metric_field        = ""
        is_max_static       = false
        gauge_min           = -4
        gauge_max           = 4
        render_boundary_min = -4
        render_boundary_max = 4
        search              = ""
        is_min_static       = false
        threshold_levels {
          severity_label  = "critical"
          threshold_value = 4
        }
        threshold_levels {
          severity_label  = "high"
          threshold_value = 3
        }
        threshold_levels {
          severity_label  = "medium"
          threshold_value = 2
        }
        threshold_levels {
          severity_label  = "low"
          threshold_value = 1
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
        search              = ""
      }
    }
  }


}