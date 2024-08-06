package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDataSourceKPIBaseSearchSchema(t *testing.T) {
	testDataSourceSchema(t, new(dataSourceKpiBaseSearch))
}

func TestAccDataSourceKPIBaseSearchLifecycle(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckResourceDestroy(resourceNameKPIBaseSearch, "TestAcc_DataSourceKPIBaseSearchLifecycle_test_base_search2"),
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
