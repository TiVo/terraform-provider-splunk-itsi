---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "itsi_entity_type Data Source - itsi"
subcategory: ""
description: |-
  Use this data source to get the ID of an available entity type.
---

# itsi_entity_type (Data Source)

Use this data source to get the ID of an available entity type.

## Example Usage

```terraform
data "itsi_entity_type" "host" {
  title = "Host"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `title` (String) The name of the entity type

### Read-Only

- `id` (String) Identifier for this entity type
