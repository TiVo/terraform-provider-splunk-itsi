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

/*
  Alternatively, provider configuation can be provided by using the following environment variables:
    * ITSI_HOST
    * ITSI_PORT
    * ITSI_ACCESS_TOKEN
    * ITSI_USER
    * ITSI_PASSWORD
    * ITSI_INSECURE
*/
