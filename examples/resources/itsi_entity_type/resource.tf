resource "itsi_entity_type" "dashboard_drilldowns" {
  title = "Dashboard Drilldowns"
  dashboard_drilldown {
    dashboard_id   = "windows_windowsupdate"
    dashboard_type = "xml_dashboard"

    title = "Windows Update - Windows"
  }
  dashboard_drilldown {
    base_url = "https://itsi.oi.tivo.com/en-US/app/itsi/homeview?view=standard&viewType=service_topology&earliest=-60m%40m&latest=now"
    params = {
      mso_id = "service"
    }
    title = "Service Analyzer (for MSO)"
  }
}

resource "itsi_entity_type" "Kubernetes_Pod" {
  description = "Kubernetes Pod type"
  title       = "Kubernetes Pod"
  data_drilldown {
    entity_field_filter {
      data_field   = "pod-name"
      entity_field = "pod-name"
    }
    entity_field_filter {
      data_field   = "pod-namespace"
      entity_field = "pod-namespace"
    }
    static_filters {
      metric_name = "kube.pod.*"
    }
    title = "Kubernetes Pod metrics"
    type  = "metrics"
  }
  data_drilldown {
    entity_field_filter {
      data_field   = "namespace"
      entity_field = "pod-namespace"
    }
    entity_field_filter {
      data_field   = "pod"
      entity_field = "pod-name"
    }
    static_filters = {
      sourcetype = "kube:objects:pods"
    }
    title = "Kubernetes Pod metadata"
    type  = "events"
  }
  vital_metric {
    is_key = true
    matching_entity_fields = {
      pod-name      = "pod-name"
      pod-namespace = "pod-namespace"
    }
    metric_name = "Average CPU Usage"
    search      = "| mstats avg(kube.pod.cpu.usage_rate) as val WHERE `itsi_entity_type_k8s_pod_metrics_indexes` by pod-name, pod-namespace span=5m"
  }
  vital_metric {
    matching_entity_fields = {
      pod-name      = "pod-name"
      pod-namespace = "pod-namespace"
    }
    metric_name = "Average Memory Usage"
    search      = "| mstats avg(kube.pod.memory.working_set_bytes) as val WHERE `itsi_entity_type_k8s_pod_metrics_indexes` by pod-name, pod-namespace span=5m"
    unit        = "Bytes"
  }
}