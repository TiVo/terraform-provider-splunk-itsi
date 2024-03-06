resource "itsi_collection_data" "fruits" {
  scope = "default"

  collection {
    name = "fruits"
  }

  entry {
    id = "first"
    data = jsonencode({
      name  = "apple"
      color = ["red", "green"]
    })
  }

  entry {
    id = "second"
    data = jsonencode({
      name  = "orange"
      color = "orange"
    })
  }

  entry {
    data = jsonencode({
      name  = "peach"
      color = ["yellow", "orange", "red"]
    })
  }
}
