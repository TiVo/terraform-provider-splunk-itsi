// Tag creation
resource "itsi_service" "service_create_tag_test" {
  title = "Test Tag Lifecycle"
  tags  = ["tag1", "tag2", "tag3"]
}