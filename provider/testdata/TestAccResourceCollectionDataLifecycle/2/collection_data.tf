/*
    TestAccResourceCollectionDataLifecycle - Step 2
    Test the update of a collection data resource
*/

locals {
  test_collection = [
    {
      name  = "apple",
      type  = "fruit",
      color = "red"
    },
    {
      name  = "banana",
      type  = "fruit",
      color = "yellow"
    },
    {
      name  = "potato",
      type  = "vegetable",
      color = ["brown", "white"]
    },
    {
      name  = "tomato",
      type  = "fruit",
      color = "green",
      test  = 100500
    },
    {
      name  = "lettuce",
      type  = "vegetable",
      color = "green",
      test  = false
    },
    {
      name  = "pumpkin",
      type  = "vegetable",
      color = "orange",
      test  = null
    },
    {
      name  = "grape",
      type  = "fruit",
      color = [{ "123" : "123" }],
    },
    {
      name  = "pepper",
      type  = "vegetable",
      color = "green",
      test  = 123.456
    },
    {
      name  = "kiwi",
      type  = "fruit",
      color = "green"
      test  = { "test" = 123.31231, 2 = null }

    },
    {
      name  = "cucumber",
      type  = "vegetable",
      color = ["green", "123"]
    },
    {
      name  = "radish",
      type  = "vegetable",
      color = "red",
      test  = ["123", 123, 123.456, null, true]
    },
  ]
}



resource "itsi_collection_data" "test" {
  scope = "TestAcc_collection_data_test"

  collection {
    name = itsi_splunk_collection.test.name
  }

  dynamic "entry" {
    for_each = { for item in local.test_collection : item.name => item }
    content {
      data = jsonencode(entry.value)
    }
  }

  entry {
    #id = random_id.rng3.hex
    data = jsonencode({
      name  = "orange"
      color = [["orange123"]]
    })
  }

}
