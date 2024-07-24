// Adding tags on existing resource
resource "itsi_service" "service_create_tag_test" {
  title = "TestAcc_Test_Tag_Lifecycle"
  tags  = ["tag6", "tag7"]
}