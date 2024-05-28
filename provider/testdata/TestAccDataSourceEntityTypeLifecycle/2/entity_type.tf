
resource "itsi_entity_type" "test" {
  title       = "TestAcc_sample_entity_type"
  description = "TestAcc EXAMPLE"
}

data "itsi_entity_type" "test" {
  title = "TestAcc_sample_entity_type"
}
