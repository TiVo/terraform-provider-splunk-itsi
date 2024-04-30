/*
    TestAccResourceCollectionDataID - Step 1
    Test successful creation of a collection data resource with multiple entries,
    with one entry having an ID provided explictly, and one having an ID generated by the provider.
*/

resource "itsi_collection_data" "test" {
  scope = "TestAcc_collection_data_id_test"

  collection {
    name = itsi_splunk_collection.test.name
  }

  entry {
    id   = "apple"
    data = jsonencode({ name = "apple" })
  }

  entry {
    data = jsonencode({ name = "banana" })
  }

}