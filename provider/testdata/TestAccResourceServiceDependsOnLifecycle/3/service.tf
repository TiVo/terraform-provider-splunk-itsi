resource "itsi_service" "service_create_parent" {
    title = "Service Test on Create Parent"
}

resource "itsi_service" "service_create_leaf" {
    title = "Service Test on Create Leaf"
}

resource "itsi_service" "service_create_leaf_overloaded" {
    title = "Service Test on Create Leaf Overloaded"
}