
resource "itsi_splunk_collection_entries" "test_collection_fruits" {
  collection_name = "test_collection"
  preserve_keys   = true
  scope           = "fruits"

  data = [
    [
      {
        name   = "_key",
        values = ["first"]
      },
      {
        name   = "color",
        values = ["red"]
      },
      {
        name   = "fruit",
        values = ["apple"]
      }
    ],
    [
      {
        name   = "_key",
        values = ["fourth"]
      },
      {
        name   = "color",
        values = ["orange"]
      },
      {
        name   = "fruit",
        values = ["orange"]
      }
    ],
    [
      {
        name   = "_key",
        values = ["fifth"]
      },
      {
        name   = "color",
        values = ["yellow"]
      },
      {
        name   = "fruit",
        values = ["pineapple"]
      }
    ],
  ]
}