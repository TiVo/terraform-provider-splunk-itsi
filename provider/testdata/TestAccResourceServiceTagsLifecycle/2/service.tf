// Removing tag
resource "itsi_service" "service_create_tag_test" {
  title = "TestAcc_Test_Tag_Lifecycle"
  tags  = ["tag1", "tag3"]
}