resource "itsi_service" "service_create_parent" {
  title = "TestAcc_Service_Test_on_Create_Parent"
}

resource "itsi_service" "service_create_leaf" {
  title = "TestAcc_Service_Test_on_Create_Leaf"
}

resource "itsi_service" "service_create_leaf_overloaded" {
  title = "TestAcc_Service_Test_on_Create_Leaf_Overloaded"
}