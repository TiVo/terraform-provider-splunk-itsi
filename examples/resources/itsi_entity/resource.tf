resource "itsi_entity" "host_entity" {

  title       = "example.com"
  description = "example.com host"

  aliases = {
    "entityTitle" = "example.com"
    "host"        = "example.com"
  }

  info = {
    "env" : "test"
    "entityType" : "host"
  }

}
