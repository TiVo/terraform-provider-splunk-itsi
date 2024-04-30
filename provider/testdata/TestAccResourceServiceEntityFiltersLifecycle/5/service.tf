// add entity_rules to the existing resource
resource "itsi_service" "service_create_filter_test" {
  title = "TestAcc_Test_Service_Create_filter_test"
  entity_rules {
    rule {
      field      = "entityTitle"
      field_type = "title"
      rule_type  = "matches"
      value      = "android_tivoos"
    }
  }
  entity_rules {
    rule {
      field      = "entityField"
      field_type = "info"
      rule_type  = "not"
      value      = "android_mobile"
    }
  }
}