# Getting Started
## Table of Contents:
- [Overview](#overview)
- [Install The Splunk ITSI Plugin](#install-the-splunk-itsi-plugin)
- [Creating The First Tree](#creating-the-first-tree)
    * [Splunk Search](#splunk-search)
    * [Entities](#entities)
    * [KPI Base Searches](#kpi-base-searches)
    * [KPI Threshold Templates](#kpi-threshold-templates)
    * [Services](#services)
    * [Declaring Service's Hierarchy](#declaring-service-s-hierarchy)
- [Lookup Usage](#lookup-usage)

## Overview
The main goal of this document is to introduce the main possibilities made available by the Splunk ITSI Terraform provider. The [Creating The First Tree Section](https://github.com/TiVo/terraform-provider-splunk-itsi/wiki/Getting-Started-Guide#overview) describes the process of constructing a service tree from scratch, but it is assumed that the reader is already familiar with the basics of the Splunk, Splunk ITSI, and Terraform.
## Install The Splunk ITSI Plugin
To load a provider from the hashicorp registry add following lines to your provider's configuration file:
```terraform
terraform {
  required_providers {
    itsi = {
      source  = "tivo/splunk-itsi"
      ## required version
      version = "0.8.4"
    }
  }
}
```
The Splunk ITSI Terraform provider supports connections via user credentials or a bearer-token:

<table>
<tr>
<td>

```terraform
provider "itsi" {
  user     = changeme
  password = changeme
  host     = changeme
  port     = changeme
  #Disable ssl checks:
  #insecure = true
}
```

</td>
<td>

```terraform
provider "itsi" {
    access_token = changeme

    host = changeme
    port = changeme
}
```

</td>
</tr>
</table>

## Creating The First Tree

### Splunk Search

[The Splunk search data source](https://registry.terraform.io/providers/TiVo/splunk-itsi/latest/docs/data-sources/splunk_search) provides a method of getting Splunk search job results into Terraform. This data source allows retrieving data in various ways:
- Specifying several searches in one resource returns concatenated data, which could be useful for keeping queries simple and deterministic;
- Use `join_fields` for the array of fields to perform a full join on them.
  The data is returned as a list of maps, where keys are field names and values - field values. Combined with terraform expressions, the itsi_splunk_search data source provides a powerful mechanism to create dynamic configurations that are powered by data from Splunk itself.

For building a service tree, let's create a search data source that brings all of the data required for entities into Terraform. The Splunk UI analogue of this operation would be importing entities from a search.

In this example, let the service nodes that are leaves of the tree hierarchy each represent a host. The first search, specified in the schema below will bring the entity title and type; the second one will bring additional info field about each server's role:

```terraform
data "itsi_splunk_search" "guide_itsi_entity_search" {

  search {
    query = <<-EOT
       search index=_internal host=itsi*
        | stats dc(host) by host
        | eval entity_type="guide_itsi_host"
        | table host, entity_type
    EOT

    earliest_time = "-4h"
  }

  search {
    query = <<-EOT
      | rest splunk_server=itsi* /services/server/info
      | stats values(server_roles) as server_roles by splunk_server
      | rename splunk_server as host
      | fieldformat server_roles=tostring(server_roles)
    EOT
    earliest_time = "-15m"
  }
  join_fields = ["host"]
}
```

To output search results add:

```terraform
output "splunk_search_data" {
  value = data.itsi_splunk_search.guide_itsi_entity_search.results
}
```
As `join_fields = ["host"]` was used - `server_roles` info field was joined with all hosts the second search obtained info about:

<pre><code>
Outputs:
splunk_search_data = tolist([
  tomap({
    "entity_type" = "guide_itsi_host"
    "host" = "itsi-validator01.oip1.tivo.com"
    <strong>"server_roles" = "cluster_search_head indexer kv_store search_head"</strong>
  }),
  tomap({
    "entity_type" = "guide_itsi_host"
    "host" = "itsi-deployer01.oip1.tivo.com"
  }),
  tomap({
    "entity_type" = "guide_itsi_host"
    "host" = "itsi-searchhead01.oip1.tivo.com"
  }),
  tomap({
    "entity_type" = "guide_itsi_host"
    "host" = "itsi-searchhead02.oip1.tivo.com"
  }),
  tomap({
    "entity_type" = "guide_itsi_host"
    "host" = "itsi-searchhead03.oip1.tivo.com"
  }),
])
</code></pre>

### Entities
[The entity data resource](https://registry.terraform.io/providers/TiVo/splunk-itsi/latest/docs/resources/entity) represents the ITSI entity object. The below example shows how to create an entity using the Splunk search defined in [the previous section](https://github.com/TiVo/terraform-provider-splunk-itsi/wiki/Getting-Started-Guide#splunk-search):

```terraform
resource "itsi_entity_type" "guide_itsi_host" {
  title = "guide_itsi_host"
}

resource "itsi_entity" "guide_itsi_entities" {
  for_each = { for entity in data.itsi_splunk_search.guide_itsi_entity_search.results:  entity["host"] => entity }

  title           = each.key
  aliases = {
    "entity"      =  each.key
    "entityTitle" =  each.key
    "host" = each.key
  }
  info = {
    "entityType" = each.value["entity_type"]
    "serverRoles" = try(each.value["server_roles"], "unknown")
  }

  entity_type_ids = [itsi_entity_type.guide_itsi_host.id]
}
```
In this example the entity type association is achieved by creating the `entity_type` resource and linking it in the entity resource: `entity_type_ids = [ itsi_entity_type.guide_itsi_host.id]`. If the entity type already exists, you can instead find its ID by using [the entity type data source](https://registry.terraform.io/providers/TiVo/splunk-itsi/latest/docs/data-sources/entity_type).

Since the search that enriched each entity with `server_roles` info may not cover all hosts, this field is provided a fallback value of 'unknown':`"serverRoles" = try(each.value["server_roles"], "unknown")`. After terraform apply command is done, we can verify entities in UI:

![guide_entities_list](https://user-images.githubusercontent.com/10566979/172616295-b30f3d67-f1ec-417e-a2bc-67444a80e315.png)

### KPI Base Searches
[The KPI base search data resource](https://registry.terraform.io/providers/TiVo/splunk-itsi/latest/docs/resources/kpi_base_search) provides the ability to manage a search definition across multiple KPIs in ITSI.

The example below demonstrates a KPI base search that monitors the average duration of the successful Splunk search jobs that were run by each host:

```terraform
resource "itsi_kpi_base_search" "guide_kpi_base_search_success_job_runtime" {
  title = "ITSI GUIDE: Success Job Runtime"
  alert_period = "5"
  search_alert_earliest = "60"
  alert_lag = "60"
  is_entity_breakdown = false
  entity_breakdown_id_fields = ""
  is_service_entity_filter = true
  entity_id_fields = "host"
  base_search =  <<-EOT
      index=_internal sourcetype=scheduler status="success"
      | bin span=5m _time
      | search host=*itsi*
      | stats avg(run_time) as avg_runtime by _time, host
    EOT
  description = "Success Job Runtime"
  metrics {
    title = "Avg Job Runtime"
    threshold_field = "avg_runtime"
    unit = "sec"
    aggregate_statop = "avg"
    entity_statop = "avg"
    fill_gaps = "last_available_value"
  }

  sec_grp = "default_itsi_security_group"
  source_itsi_da = "itsi"
}
```

The schema of the KPI base search maps intuitively to the ITSI UI web form, but there are some fields worthy of specific mention.  The following table is a map of corresponding UI fields: specify `alert_period = "5"` to schedule KPI base search for every 5 minutes, `search_alert_earliest = "60"` to select a 60-sec calculation window, etc.

<table>
<tr>
<td>

```terraform
  alert_period = "5"
  search_alert_earliest = "60"
  alert_lag = "60"
  is_entity_breakdown = false
  entity_breakdown_id_fields = ""
  is_service_entity_filter = true
  entity_id_fields = "host"
```
</td>
<td>
<img src=https://user-images.githubusercontent.com/10566979/172702322-0128a8f8-8a1f-4890-9af0-77f12f94359e.png></img>

</td>
</tr>
</table>

In this case, to specify that search results should be filtered by entity when presented within services: define `is_service_entity_filter = true`, where the entity filter field is `entity_id_fields = "host"`.

Let's specify an additional KPI base search that counts skipped or failed Splunk search jobs that are run by each host. This KPI base search has several metrics which are split by pseudo-entities corresponding to the name of the Splunk `app` that contains each search.

```terraform
resource "itsi_kpi_base_search" "guide_kpi_base_search_non_success_job_count" {
  title = "ITSI GUIDE: Non-success Job Count"
  alert_period = "1"
  search_alert_earliest = "30"
  alert_lag = "120"
  is_entity_breakdown = true
  entity_breakdown_id_fields = "app"
  is_service_entity_filter = true
  entity_id_fields = "host"
  base_search =  <<-EOT
      index=_internal sourcetype=scheduler status!=success
      | bin span=5m _time
      | eval failed=if((status="delegated_remote_completion" AND success=0) OR status="delegated_remote_error", 1, 0)
      | eval skipped=if(status="skipped", 1, 0)
      | search host=*itsi*
      | stats sum(failed) as failed_count, sum(skipped) as skipped_count by _time, host, app
    EOT
  description = "Non-success Job Count"
  entity_breakdown_id_fields = "app"
  entity_id_fields = "host"
  is_entity_breakdown = true
  is_service_entity_filter = true
  metrics {
    title = "Failed Job Count"
    threshold_field = "failed_count"
    unit = "-"
    aggregate_statop = "median"
    entity_statop = "sum"
    fill_gaps = "null_value"
  }
  metrics {
    title = "Skipped Job Count"
    threshold_field = "skipped_count"
    unit = "-"
    aggregate_statop = "median"
    entity_statop = "sum"
    fill_gaps = "null_value"
  }

  sec_grp = "default_itsi_security_group"
  source_itsi_da = "itsi"
}
```
To specify that search results should be split by the Splunk app as an entity: define `is_entity_breakdown = true`, where the entity split by (breakdown) field is `entity_breakdown_id_fields = "app"`.

<table>
<tr>
<td>


```terraform
  alert_period = "1"
  search_alert_earliest = "30"
  alert_lag = "120"
  is_entity_breakdown = true
  entity_breakdown_id_fields = "app"
  is_service_entity_filter = true
  entity_id_fields = "host"
```
</td>
<td>
<img src=https://user-images.githubusercontent.com/10566979/172702182-d2153757-a692-475e-8f10-6c2af072bd76.png></img>
</td>
</tr>
</table>

### KPI Threshold Templates
The Splunk ITSI Terraform provider offers[ data source](https://registry.terraform.io/providers/TiVo/splunk-itsi/latest/docs/data-sources/kpi_threshold_template) and [resource data](https://registry.terraform.io/providers/TiVo/splunk-itsi/latest/docs/resources/kpi_threshold_template) approaches to work with threshold templates. The data source approach could be useful if your infrastructure already has a bunch of legacy threshold templates that you wish to leverage and when you do not want to import and manage the KPI Threshold Templates via Terraform.

The provider supports time-variate or adaptive thresholds only, so this guide will cover them as an example.

For the purposes of this example, let's assume a simple case.  In this case, our templates are more strict on Friday evening only, with the intention of producing notable events hinting that things might break in advance of the weekend starting. :) In this way, only one `Fri: 18:00 (+3 hours)` time policy is required for demonstration.

The example below shows a time-variate static KPI Threshold Template, with one time policy:

```terraform
resource "itsi_kpi_threshold_template" "guide_static_sample" {
  title                                 = "Guide: Static time-variate sample"
  description                           = "static"
  adaptive_thresholds_is_enabled        = false
  adaptive_thresholding_training_window = "-7d"
  time_variate_thresholds               = true
  sec_grp                               = "default_itsi_security_group"

  time_variate_thresholds_specification {

    policies {
      policy_name = "default_policy"
      title       = "Default"
      policy_type = "static"
      aggregate_thresholds {
        render_boundary_min = -4
        render_boundary_max = 4
        threshold_levels {
          severity_label  = "high"
          threshold_value = 90
        }
        threshold_levels {
          severity_label  = "medium"
          threshold_value = 70
        }
        threshold_levels {
          severity_label  = "low"
          threshold_value = 50
        }
      }

      entity_thresholds {
        render_boundary_max = 100
        render_boundary_min = 0
      }
    }

    policies {
      policy_name = "friday_evening"
      title       = "Fri: 18:00 (+3 hours)  (UTC-00:00)"
      time_blocks {
        cron = "0 18 * * 4"
        interval = 180
      }
      policy_type = "static"
      aggregate_thresholds {
        render_boundary_min = 0
        render_boundary_max = 77
        threshold_levels {
          severity_label  = "high"
          threshold_value = 70
        }
        threshold_levels {
          severity_label  = "medium"
          threshold_value = 50
        }
        threshold_levels {
          severity_label  = "low"
          threshold_value = 30
        }
      }

      entity_thresholds {
        render_boundary_max = 100
        render_boundary_min = 0
      }
    }
  }
}
```
We can verify that the applied result looks as expected with the UI KPI Threshold Template viewer:
![guide_static_threshold_template](https://user-images.githubusercontent.com/10566979/172671887-ae1d8539-4b12-4809-bf36-8ebfcc05db7c.png)

An adaptive (eg. standard deviation) KPI Threshold Template could be defined instead by making the following changes:

```diff
-resource "itsi_kpi_threshold_template" "guide_static_sample" {
-  title                                 = "Guide: Static time-variate sample"
-  description                           = "static"
-  adaptive_thresholds_is_enabled        = false
+
+
+resource "itsi_kpi_threshold_template" "guide_stdev_sample" {
+  title                                 = "Guide: Adaptive stdev sample"
+  description                           = "adaptive"
+  adaptive_thresholds_is_enabled        = true
   adaptive_thresholding_training_window = "-7d"
   time_variate_thresholds               = true
   sec_grp                               = "default_itsi_security_group"

   time_variate_thresholds_specification {

     policies {
       policy_name = "default_policy"
       title       = "Default"
-      policy_type = "static"
+      policy_type = "stdev"
       aggregate_thresholds {
         render_boundary_min = -4
         render_boundary_max = 4
         threshold_levels {
           severity_label  = "high"
-          threshold_value = 90
+          threshold_value = 1.75
+          dynamic_param = 1.75
         }
         threshold_levels {
           severity_label  = "medium"
-          threshold_value = 70
+          threshold_value = 1.25
+          dynamic_param = 1.25
         }
         threshold_levels {
           severity_label  = "low"
-          threshold_value = 50
+          threshold_value = 0.75
+          dynamic_param = 0.75
         }
       }
       entity_thresholds {
         render_boundary_max = 100
         render_boundary_min = 0
       }
     }

     policies {
       policy_name = "friday_evening"
       title       = "Fri: 18:00 (+3 hours)  (UTC-00:00)"
       time_blocks {
         cron = "0 18 * * 4"
         interval = 180
       }
-      policy_type = "static"
+      policy_type = "stdev"
       aggregate_thresholds {
         render_boundary_min = 0
         render_boundary_max = 77
         threshold_levels {
           severity_label  = "high"
-          threshold_value = 70
+          threshold_value = 1.25
+          dynamic_param = 1.25
         }
         threshold_levels {
           severity_label  = "medium"
-          threshold_value = 50
+          threshold_value = 0.75
+          dynamic_param = 0.75
         }
         threshold_levels {
           severity_label  = "low"
-          threshold_value = 30
+          threshold_value = 0.5
+          dynamic_param = 0.5
         }
       }

```


### Services
[The Service data resource ](https://registry.terraform.io/providers/TiVo/splunk-itsi/latest/docs/resources/service)corresponds to the ITSI service object. The below example creates services for each entity defined in [Entities section](https://github.com/TiVo/terraform-provider-splunk-itsi/wiki/Getting-Started-Guide#entities). The declared KPIs are linked to the KPI base searches created in [KPI base searches section](https://github.com/TiVo/terraform-provider-splunk-itsi/wiki/Getting-Started-Guide#kpi-base-searches). Each KPI is tied to the base search via `base_search_id     = itsi_kpi_base_search.guide_kpi_base_search_success_job_runtime.id` and the metric that is associated via title: `base_search_metric = "Avg Job Runtime"`. The ITSI Splunk Terraform provider validates the metric for typos in the title, so if the metric doesn't exist per the current base search an error message will be displayed.
```terraform
resource "itsi_service" "guide_service_leaf" {
  for_each = itsi_entity.guide_itsi_entities
  title          = each.value.title
  enabled = true
  entity_rules {
    rule {
      field = "host"
      field_type = "alias"
      rule_type  = "matches"
      value      = each.value.title
    }
    rule {
      field = "entityType"
      field_type = "info"
      rule_type  = "matches"
      value      = data.itsi_entity_type.guide_itsi_host.title
    }
  }
  kpi {
    title              = "Avg Success Status Job Runtime"
    base_search_id     = itsi_kpi_base_search.guide_kpi_base_search_success_job_runtime.id
    base_search_metric = "Avg Job Runtime"
    custom_threshold {
      aggregate_thresholds {
        render_boundary_max = 100
        render_boundary_min = 0
        threshold_levels {
          severity_label  = "medium"
          threshold_value = 75
        }
        threshold_levels {
          severity_label  = "high"
          threshold_value = 88
        }
        threshold_levels {
          severity_label  = "low"
          threshold_value = 50
        }
      }
      entity_thresholds {
        render_boundary_max = 100
        render_boundary_min = 0
      }
    }
    search_type           = "shared_base"
    threshold_template_id = ""
    type                  = "kpis_primary"
    urgency               = 5
  }
  kpi {
    title              = "Skipped Job Count"
    base_search_id     = itsi_kpi_base_search.guide_kpi_base_search_non_success_job_count.id
    base_search_metric = "Skipped Job Count"
    threshold_template_id = itsi_kpi_threshold_template.guide_static_sample.id
  }
  kpi {
    title              = "Failed Job Count"
    base_search_id     = itsi_kpi_base_search.guide_kpi_base_search_non_success_job_count.id
    base_search_metric = "Failed Job Count"
    threshold_template_id = itsi_kpi_threshold_template.guide_stdev_sample.id
  }
  security_group = "default_itsi_security_group"
}
```
The entity filtering for the services is configured via the `entity_rules` schema:
<table>
<tr>
<td>

```terraform
entity_rules {
  rule {
    field = "host"
    field_type = "alias"
    rule_type  = "matches"
    value      = each.value.title
  }
  rule {
    field = "entityType"
    field_type = "info"
    rule_type  = "matches"
    value      =
     data.itsi_entity_type.guide_itsi_host.title
  }
}
```
</td>
<td>
<img src=https://user-images.githubusercontent.com/10566979/172697021-5289f0ec-5931-4812-b14b-54bb87c7792c.png></img>
</td>
</tr>
</table>

A KPI Threshold Template may be associated with each KPI via `threshold_template_id = itsi_kpi_threshold_template.guide_static_sample.id`. If a static, not time-variate or templated threshold, is desired for this KPI instead, then the `custom_threshold` can be specified instead.

<table>
<tr>
<td>

```terraform
aggregate_thresholds {
  render_boundary_max = 100
  render_boundary_min = 0
  threshold_levels {
    severity_label  = "high"
    threshold_value = 88
  }
  threshold_levels {
    severity_label  = "medium"
    threshold_value = 75
  }
  threshold_levels {
    severity_label  = "low"
    threshold_value = 50
  }
}
```

</td>
<td>
<img src=https://user-images.githubusercontent.com/10566979/172698848-8c08af0c-b7c9-499b-a348-2a38ea550157.png></img>
</td>
</tr>
</table>


### Declaring Service Hierarchy/Dependencies

The Splunk ITSI Terraform provider also the declaration of a dependent service in any service object. All KPI dependencies for each service can be specified via the `service_depends_on` schema.
The example below shows the creation of the parent service for the leaf services created in the previous section. This service depends on each leaf services' health score KPIs. The field `service_depends_on.value.shkpi_id` is generated by Splunk and is known only after a Terraform apply command is performed. The creation/update of the parent service will be postponed until all of its service dependencies are satisfied.

```terraform
resource "itsi_service" "guide_service_root" {
  title = "GUIDE: ITSI Job Metrics"
  enabled = true
  dynamic "service_depends_on" {
    for_each = itsi_service.guide_service_leaf
    content {
      service = service_depends_on.value.id
      kpis    = [ service_depends_on.value.shkpi_id ]
    }
  }
}
```
The below table shows content of the terraform state file's `guide_service_root` and the corresponding ITSI UI view.
<table>
<tr>
<td>

```json
"service_depends_on": [
  {
    "kpis": ["SHKPI-450dd3c7-cf02-425b-9328-43ceeac9771e"],
    "service": "450dd3c7-cf02-425b-9328-43ceeac9771e"
  },
  {
    "kpis": ["SHKPI-5c553737-810f-4051-8e13-2441b6727458"],
    "service": "5c553737-810f-4051-8e13-2441b6727458"
  },
  {
    "kpis": ["SHKPI-c73a1212-c2ba-4d4f-a666-ca951f15ef18"],
    "service": "c73a1212-c2ba-4d4f-a666-ca951f15ef18"
  },
  {
    "kpis": ["SHKPI-ef0b5f07-82d1-4ce9-b3d9-2fd28e1f414d"],
    "service": "ef0b5f07-82d1-4ce9-b3d9-2fd28e1f414d"
  },
  {
    "kpis": ["SHKPI-fff7e390-25a6-44cf-9512-366208420624"],
    "service": "fff7e390-25a6-44cf-9512-366208420624"
  }
]

```
</td>
<td>
<img src=https://user-images.githubusercontent.com/10566979/172704183-c1904ff9-adcd-4fb8-81e1-1e51d9c8fd25.png></img>
</td>
</tr>
</table>

Service hierarchy and entity filtering rules can be verified in the ITSI Service Analyzer:
![guide_service_tree](https://user-images.githubusercontent.com/10566979/172705586-fa2d4853-6040-44ef-8ba7-8e1d1a4713bd.png)


## Lookup Usage
The Splunk ITSI Terraform provider includes multiple resources to define and control Splunk KV Store collections. When combined with [the Splunk Terraform provider](https://registry.terraform.io/providers/splunk/splunk/1.4.13) and its ability to control Splunk lookup table objects, a complete solution to manage Splunk lookups through terraform is available.

Extend your configuration file with the Splunk Terraform provider:
```terraform
terraform {
  required_providers {
    itsi = {
      source  = "tivo/splunk-itsi"
      version = "0.8.4"
    }
    splunk = {
      source  = "splunk/splunk"
    }
  }
}
```
Add configuration options:
<table>
<tr>
<td>

```terraform
provider "splunk" {
  username     = changeme
  password     = changeme
  url          = changeme
  #insecure_skip_verify = true
}
```

</td>
<td>

```terraform
provider "splunk" {
  url = changeme
  auth_token = changeme
}
```

</td>
</tr>
</table>

[The Splunk Collection resource ](https://registry.terraform.io/providers/TiVo/splunk-itsi/latest/docs/resources/splunk_collection) manages Splunk KV Store collections. To use a collection within Splunk via the SPL `lookup` command, a lookup table definition is additionally required:

```terraform
resource "itsi_splunk_collection" "guide_test_metadata" {
  name = "GUIDE_TEST_METADATA"
}

resource "splunk_configs_conf" "guide_test_metadata" {
  name = "transforms/${itsi_splunk_collection.guide_test_metadata.name}"
  variables = {
    "collection" : itsi_splunk_collection.guide_test_metadata.name
    "external_type" : "kvstore"
    "fields_list": "service_title,service_id,service_hs,service_kpi"
  }

  acl {
    app   = "itsi"
    owner = "nobody"
    read  = ["*"]
    write = ["admin", "power"]
    sharing = "global"
  }
}

```
After a Terraform apply command, the lookup definition will appear:
![guide_lookup_definition_](https://user-images.githubusercontent.com/10566979/172709746-9245f8db-85f6-4912-bbd6-12f288837676.png)

The example below creates some data to drive the creation of entries with a collection. Each entry should be a map with keys: `"service_title,service_id,service_hs,service_kpi"` as was defined in the lookup object above.

```terraform
locals {
  test_metadata = {
    for service in itsi_service.guide_service_leaf: service["title"] =>
      {
        service_title: service["title"]
        service_id: service["id"]
        service_hs: service["shkpi_id"]
        service_kpi: join(",", [for kpi in service["kpi"]: kpi["title"]])
      }
  }
}
```

A collection can be updated with the list of records using the [itsi_splunk_collection_entries](https://registry.terraform.io/providers/TiVo/splunk-itsi/latest/docs/resources/splunk_collection_entries) resource or with a single record using the [itsi_splunk_collection_entry](https://registry.terraform.io/providers/TiVo/splunk-itsi/latest/docs/resources/splunk_collection_entry) resource:

```terraform
resource "itsi_splunk_collection_entry" "service_attr" {
  for_each = local.test_metadata
  collection_name = itsi_splunk_collection.guide_test_metadata.name
  key  = each.key
  data = each.value
}
```

After a Terraform apply command, the lookup table's data can be verified with a simple SPL search:
![guide_lookup_data](https://user-images.githubusercontent.com/10566979/172709724-30904c9f-f2dd-496c-b827-6929a38d3a4d.png)






