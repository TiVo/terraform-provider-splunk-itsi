# Splunk ITSI Terraform Provider Versioning Strategy

The Terraform Provider for Splunk ITSI follows Semantic Versioning (SemVer). Here's a brief overview of what to expect:

* Major versions (vX.0.0): Breaking schema changes can be introduced. Upgrading to a new major version might require manual intervention or changes in Terraform configurations. It's crucial to consult the release notes and migration guides for any breaking changes.
* Minor versions (vX.Y.0): Backward-compatible new features and enhancements are added. Schema compatibility is maintained between these versions.
* Patch versions (vX.Y.Z): Backward-compatible bug fixes. These versions are safe to upgrade without making any changes to Terraform configurations.

As a best practice, before upgrading to a new version, especially a major version, testing the upgrade in a controlled environment is recommended. Reading the associated release notes and migration guides can provide insights into the changes and potential implications.

