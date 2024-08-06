resource "itsi_splunk_collection" "test" {
  name = "TestAcc_ResourceCollectionLifecycle_collection_test"

  field_types = {
    name        = "string"
    color       = "string"
    test        = "bool"
    description = "string"
  }

  accelerations = [
    jsonencode({ name = 1 }),
    jsonencode({ color = 1 }),
  ]

}
