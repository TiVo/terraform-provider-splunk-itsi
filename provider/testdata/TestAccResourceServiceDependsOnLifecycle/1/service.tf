resource "itsi_service" "service_create_parent" {
  title = "Service Test on Create Parent"
  service_depends_on {
    kpis = [
      itsi_service.service_create_leaf.shkpi_id
    ]
    service = itsi_service.service_create_leaf.id
  }
}

resource "itsi_service" "service_create_leaf" {
  title = "Service Test on Create Leaf"
}