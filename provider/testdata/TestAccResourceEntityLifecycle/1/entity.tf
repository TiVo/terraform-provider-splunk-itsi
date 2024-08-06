resource "itsi_entity" "test" {
  title       = "TestAcc_ResourceEntityLifecycle_ExampleHost"
  description = "entityTest.example.com"
  aliases = {
    "host"        = "entityTest.example.com"
    "entityTitle" = "entityTest.example.com"
  }
  info = {
    "env" : "test"
    "entityType" : "123"
  }
}
