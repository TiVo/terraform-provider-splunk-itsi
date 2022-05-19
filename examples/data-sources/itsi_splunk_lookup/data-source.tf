data "itsi_splunk_lookup" "test_lookup" {
  name = "test_lookup"
}

output "splunk_lookup_data" {
  value = data.itsi_splunk_lookup.test_lookup.data
}
