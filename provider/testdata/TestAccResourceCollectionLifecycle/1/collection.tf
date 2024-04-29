resource "itsi_splunk_collection" "test" {
  name = "TestAcc_collection_test"

  field_types = {
    name  = "string"
    color = "string"
    test  = "bool"
  }

  accelerations = [
    jsonencode({ name = 1 }),
    jsonencode({ color = 1 }),
  ]

}
