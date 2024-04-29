resource "itsi_service" "service_create_filter_test" {
  title = "Test Service Create filter test"
  entity_rules {
    rule {
      field      = "entityTitle"
      field_type = "alias"
      rule_type  = "matches"
      value      = "android_streamer"
    }
  }
}