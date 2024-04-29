/*
    TestAccResourceCollectionDataLifecycle - Step 3
    Test the deletion of all collection data resource entries
*/

resource "itsi_collection_data" "test" {
  scope = "TestAcc_collection_data_test"

  collection {
    name = itsi_splunk_collection.test.name
  }

}
