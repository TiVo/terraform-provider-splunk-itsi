/*
    TestAccResourceCollectionDataID - Step 2
    Test changing the IDs of a collection data resource entry.
    One entry has the ID config removed, and one entry has the ID config added.

    Only the entry with the ID config added should have its ID changed.
*/

resource "itsi_collection_data" "test" {
  scope = "TestAcc_collection_data_id_test"

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

}
