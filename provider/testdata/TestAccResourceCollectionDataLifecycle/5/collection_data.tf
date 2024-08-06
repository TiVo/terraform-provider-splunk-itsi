/*
    TestAccResourceCollectionDataLifecycle - Step 5
    Test creating a collection item containing unknown data
*/

provider "random" {}

resource "random_integer" "rnd" {
  min = 1
  max = 5
}

resource "itsi_collection_data" "test" {
  scope = "TestAcc_ResourceCollectionDataLifecycle_collection_data_test"

  collection {
    name = itsi_splunk_collection.test.name
  }

  entry {
    id   = "a"
    data = jsonencode({ test = "1" })
  }

  entry {
    id   = "b"
    data = jsonencode({ name = "${random_integer.rnd.result}" })
  }

}
