// add rule 
resource "itsi_service" "service_create_filter_test" {
  title = "Test Service Create filter test"
  entity_rules {
    rule {
      field      = "entityTitle"
      field_type = "alias"
      rule_type  = "matches"
      value      = "android_streamer"
    }
    rule {
      field      = "entityField"
      field_type = "info"
      rule_type  = "not"
      value      = "android_mobile"
    }
  }
}