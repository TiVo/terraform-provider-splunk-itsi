resource "itsi_service" "service_create_parent" {
  title = "TestAcc_Service_Test_on_Create_Parent"
  service_depends_on {
    kpis = [
      itsi_service.service_create_leaf.shkpi_id
    ]
    service = itsi_service.service_create_leaf.id
  }
  service_depends_on {
    kpis = [
      itsi_service.service_create_leaf_overloaded.shkpi_id
    ]
    service = itsi_service.service_create_leaf_overloaded.id
    overloaded_urgencies = {
      (itsi_service.service_create_leaf_overloaded.shkpi_id) = 8
    }

  }
}

resource "itsi_service" "service_create_leaf" {
  title = "Service_Test_on_Create_Leaf"
}

resource "itsi_service" "service_create_leaf_overloaded" {
  title = "Service_Test_on_Create_Leaf_Overloaded"
}