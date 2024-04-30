resource "itsi_service" "service_create_parent" {
  title = "TestAcc_Service_Test_on_Create_Parent"
  service_depends_on {
    kpis = [
      itsi_service.service_create_leaf.shkpi_id
    ]
    service = itsi_service.service_create_leaf.id
  }
}

resource "itsi_service" "service_create_leaf" {
  title = "TestAcc_Service_Test_on_Create_Leaf"
}