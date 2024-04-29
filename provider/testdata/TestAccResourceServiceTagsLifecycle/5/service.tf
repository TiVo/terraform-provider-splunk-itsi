// Adding tags on existing resource
resource "itsi_service" "service_create_tag_test" {
  title = "Test Tag Lifecycle"
  tags  = ["tag6", "tag7"]
}