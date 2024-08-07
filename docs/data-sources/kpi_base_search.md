---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "itsi_kpi_base_search Data Source - itsi"
subcategory: ""
description: |-
  Use this data source to get the ID of an available KPI Base Search.
---

# itsi_kpi_base_search (Data Source)

Use this data source to get the ID of an available KPI Base Search.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `title` (String) The title of the KPI Base Search.

### Optional

- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `read` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours).
