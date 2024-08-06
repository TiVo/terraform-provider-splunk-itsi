resource "itsi_entity" "test" {
  title       = "TestAcc_ResourceEntityLifecycle_ExampleHost"
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
