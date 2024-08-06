/*
    TestAccResourceCollectionDataID - Step 6
    Test that we can create items with extremely long IDs.
*/

locals {
  n           = 50
  long_string = join("", [for i in range(0, 200) : "TestAccResourceCollectionDataID"])

  entries = { for i in range(0, local.n) : "${i}_${local.long_string}" =>
    jsonencode({ data = local.long_string })
  }
}

resource "itsi_collection_data" "test" {
  scope = "TestAcc_ResourceCollectionDataID_collection_data_id_test"

  collection {
    name = itsi_splunk_collection.test.name
  }

  dynamic "entry" {
    for_each = local.entries
    content {
      id   = entry.key
      data = entry.value
    }
  }
}
