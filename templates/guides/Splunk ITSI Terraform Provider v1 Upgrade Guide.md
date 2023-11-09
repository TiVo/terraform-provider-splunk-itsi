# Splunk ITSI Terraform Provider v1 Upgrade Guide

Version 1.0.0 of the Splunk ITSI terraform provider is a major release and includes some breaking changes that you will need to consider when upgrading.

## Table of Contents:
- [Changes to the KPI threshold template resource](#changes-to-the-kpi-threshold-template-resource)
- [Splunk Collection Entries resource deprecation](#splunk-collection-entries-resource-deprecation)


## Changes to the KPI threshold template resource

~> All attributes of KPI threshold template resource's `time_variate_thresholds_specification.policies.aggregate_thresholds`, such as `base_severity_label`, `gauge_max`, `gauge_min`, `is_max_static`, `is_min_static`,
`metric_field`, `render_boundary_max`, `render_boundary_min`, `search` have become required. Practitioners should update their terraform configuration for kpi threshold templates and make sure that those attributes have values set.

!> KPI threshold template resource's terraform state schema has changed as well.
In order to migrate to the new resource version, practitioners should remove the terraform state of the KPI threshold template objects created with the old 0.x provider versions (use `terraform state rm`), and re-import the objects back using `terraform import`.

Please refer to the [kpi_threshold_template docs](https://registry.terraform.io/providers/TiVo/splunk-itsi/1.0.0/docs/resources/kpi_threshold_template) for the full schema reference and usage examples.

## Splunk Collection Entries resource deprecation

~> `splunk_collection_entries` and `splunk_collection_entry` resources are now considered deprecated and may be removed in future major releases.

Practitioners should consider migrating to the new [collection_data resource](https://registry.terraform.io/providers/TiVo/splunk-itsi/1.0.0/docs/resources/collection_data).
