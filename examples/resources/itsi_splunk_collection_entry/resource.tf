resource "itsi_splunk_collection_entry" "test_collection_sun" {
  collection_name = "test_collection"
  key             = "sun"
  data = {
    name  = "sun"
    type  = "star"
    order = "0"
  }
}
