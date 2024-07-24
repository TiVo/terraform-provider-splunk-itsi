package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDataSourceEntityTypeSchema(t *testing.T) {
	testDataSourceSchema(t, new(dataSourceEntityType))
}

func TestAccDataSourceEntityTypeLifecycle(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckResourceDestroy(resourceNameEntityType, "TestAcc_sample_entity_type"),
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
			},
		},
	})
}
