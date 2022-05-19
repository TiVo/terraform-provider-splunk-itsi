provider "itsi" {
  user     = "admin"
  password = "changeme"
  #or use Bearer token authentication:
  #access_token = "..."

  host = "itsi.example.com"
  port = 8089

  #Disable ssl checks:
  #insecure = true
}
