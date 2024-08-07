---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "itsi_kpi_threshold_template Data Source - itsi"
subcategory: ""
description: |-
  Use this data source to get the ID of an available KPI threshold template.
---

# itsi_kpi_threshold_template (Data Source)

Use this data source to get the ID of an available KPI threshold template.

## Example Usage

```terraform
data "itsi_kpi_threshold_template" "sample_threshold_template" {
  title = "Sample Threshold Template"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `title` (String) The name of the KPI Threshold template

### Optional

- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))

### Read-Only

- `id` (String) Identifier for this KPI Threshold template

<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `read` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours).
