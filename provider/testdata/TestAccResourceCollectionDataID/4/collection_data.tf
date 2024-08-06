/*
    TestAccResourceCollectionDataID - Step 4
    Test that a Duplicate ID error can be caught at plan time.
*/

resource "itsi_collection_data" "test" {
  scope = "TestAcc_ResourceCollectionDataID_collection_data_id_test"

  collection {
    name = itsi_splunk_collection.test.name
  }

  entry {
    id   = "banana"
    data = jsonencode({ name = "banana" })
  }

  entry {
    id   = "banana"
    data = jsonencode({ name = "banana", color = "yellow" })
  }

}
