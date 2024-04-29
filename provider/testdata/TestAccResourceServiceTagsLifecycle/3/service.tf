// Adding tag to set
resource "itsi_service" "service_create_tag_test" {
  title = "Test Tag Lifecycle"
  tags  = ["tag1", "tag3", "tag5"]
}