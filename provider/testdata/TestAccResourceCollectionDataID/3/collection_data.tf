/*
    TestAccResourceCollectionDataID - Step 3
    Test that a Duplicate ID error can be caught at apply time.
*/

resource "itsi_collection_data" "test" {
  scope = "TestAcc_ResourceCollectionDataID_collection_data_id_test"

  collection {
    name = itsi_splunk_collection.test.name
  }

  entry {
    data = jsonencode({ name = "apple" })
  }

  entry {
    id   = "banana"
    data = jsonencode({ name = "banana" })
  }

  entry {
    id   = "apple"
    data = jsonencode({ name = "apple", color = "red" })
  }

}
