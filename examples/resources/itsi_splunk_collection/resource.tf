resource "itsi_splunk_collection" "test_collection" {
  name = "test_collection"
  field_types = {
    type  = "string"
    order = "string"
  }
  accelerations = [jsonencode({ order : 1 })]
}

