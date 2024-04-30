/*
    TestAccResourceCollectionDataLifecycle - Step 4
    Test that we can create a collection data resource with a dynamic number of entries
*/

provider "random" {}

resource "random_integer" "number_of_entries" {
  min = 1
  max = 5
}

resource "itsi_collection_data" "test" {
  scope = "TestAcc_collection_data_test"

  collection {
    name = itsi_splunk_collection.test.name
  }

  dynamic "entry" {
    for_each = { for i in range(0, random_integer.number_of_entries.result + 1) : i => i }
    content {
      data = jsonencode({ name = "entry ${entry.key}" })
    }
  }
}
