# Splunk ITSI Terraform Provider v2 Upgrade Guide

Version 2.0.0 of the Splunk ITSI terraform provider is a major release and includes some breaking changes that you will need to consider when upgrading.

## Table of Contents:
- [Changes to the Splunk Search data source](#changes-to-the-splunk-search-data-source)
- [Splunk Lookup data source removal](#splunk-lookup-data-source-removal)
- [Splunk Collection Fields data source removal](#splunk-collection-fields-data-source-removal)
- [Legacy collection entry resources removal](#legacy-collection-entry-resources-removal)

## Changes to the Splunk Search data source

The splunk_search data source has undergone notable schema changes:

* The key update is that the `results` data structure is now encoded as a JSON string, enhancing performance when handling large result sets.
* Searches returning no results will now fail by default. A new boolean option `allow_no_results` have been introduced in case such behavior should be allowed.
* `is_mv` and `mv_separator` fields have been removed, leveraging the JSON structure for more flexible results handling.

Please refer to the [`splunk_search`](https://registry.terraform.io/providers/TiVo/splunk-itsi/2.0.0/docs/data-sources/splunk_search) for the full schema reference and usage examples.

## Splunk lookup data source removal

`splunk_lookup` data source has been removed due to its redundancy, as practitioners can achieve similar functionality through the [`splunk_search` data source](https://registry.terraform.io/providers/TiVo/splunk-itsi/2.0.0/docs/data-sources/splunk_search) by using the `inputlookup` splunk command.

## Splunk Collection Fields data source removal

`splunk_collection_fields` data source has been removed, because its functionality is now entirely covered by the new [`splunk_collection` data source](https://registry.terraform.io/providers/TiVo/splunk-itsi/2.0.0/docs/data-sources/splunk_collection), which allows to retrieve all collection details, not just the list of fields.

## Legacy collection entry resources removal

Legacy resources for collection entry management, `splunk_collection_entry` and `splunk_collection_entries` have been removed. Practitioners must update their terraform code to rely on the [`collection_data` resource](https://registry.terraform.io/providers/TiVo/splunk-itsi/2.0.0/docs/resources/collection_data) instead.
