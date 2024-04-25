resource "itsi_entity" "test" {
  title       = "TestAccExampleHost"
  description = "TEST DESCRIPTION update"
  aliases = {
    "host"        = "entityTest.example.com"
    "entityTitle" = "entityTest.example.com"
  }
  info = {
    "env" : "test"
    "entityType" : "123"
  }
}
