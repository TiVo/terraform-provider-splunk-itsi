/*
    TestAccResourceCollectionDataID - Step 5
    Test that a Duplicate ID error can be caught in case when that ID is managed by another instance of the collection data resource under a different scope.
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

}

resource "itsi_collection_data" "test2" {
  scope = "TestAcc_collection_data_id_test2"

  collection {
    name = itsi_splunk_collection.test.name
  }

  entry {
    id   = "banana"
    data = jsonencode({ name = "banana", color = "yellow" })
  }

  depends_on = [itsi_collection_data.test]
}
