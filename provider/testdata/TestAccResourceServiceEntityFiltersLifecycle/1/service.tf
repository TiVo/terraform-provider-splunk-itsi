resource "itsi_service" "service_create_filter_test" {
  title = "TestAcc_Test_Service_Create_filter_test"
  entity_rules {
    rule {
      field      = "entityTitle"
      field_type = "alias"
      rule_type  = "matches"
      value      = "android_streamer"
    }
  }
}