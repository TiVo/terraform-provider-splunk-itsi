package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

func TestResourceCollectionDataSchema(t *testing.T) {
	testResourceSchema(t, new(resourceCollectionData))
}

func TestResourceCollectionDataPlan(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: providerFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: util.Dedent(`
					provider "itsi" {
						host     = "itsi.example.com"
						user     = "user"
						password = "password"
						port     = 8089
						timeout  = 20
					}

					resource "itsi_collection_data" "test_data" {
					  scope = "example_scope"
					  collection {
					    name = "collection-data-test"
					  }

					  entry {
					    data = jsonencode({
					      name  = "abc"
					      color = [["123"]]
					    })
					  }
					}
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccResourceCollectionDataLifecycle(t *testing.T) {
	var scope = testAccResourceTitle("collection_data_test")

	resource.Test(t, resource.TestCase{
		// ExternalProviders: map[string]resource.ExternalProvider{
		// 	"random": {
		// 		Source: "registry.terraform.io/hashicorp/random",
		// 	},
		// },
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckResourceDestroy(resourceNameCollectionData, scope),
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_collection_data.test", "scope", scope),
					resource.TestCheckResourceAttr("itsi_collection_data.test", "generation", "0"),
					testAccCheckResourceExists(resourceNameCollectionData, scope),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectUnknownValue("itsi_collection_data.test", tfjsonpath.New("generation")),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectKnownValue("itsi_collection_data.test", tfjsonpath.New("generation"), knownvalue.Int64Exact(0)),
					},
				},
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_collection_data.test", "scope", scope),
					resource.TestCheckResourceAttr("itsi_collection_data.test", "generation", "1"),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectKnownValue("itsi_collection_data.test", tfjsonpath.New("generation"), knownvalue.Int64Exact(1)),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_collection_data.test", "scope", scope),
					resource.TestCheckResourceAttr("itsi_collection_data.test", "entry.#", "0"),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_collection_data.test", "scope", scope),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_collection_data.test", "scope", scope),
					resource.TestCheckResourceAttr("itsi_collection_data.test", "entry.#", "2"),
					resource.TestCheckResourceAttr("itsi_collection_data.test", "generation", "4"),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_collection_data.test", "scope", scope),
					resource.TestCheckResourceAttr("itsi_collection_data.test", "entry.#", "2"),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectUnknownValue("itsi_collection_data.test", tfjsonpath.New("generation")),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccResourceCollectionDataID(t *testing.T) {
	var scope = testAccResourceTitle("collection_data_id_test")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckResourceDestroy(resourceNameCollectionData, scope),
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_collection_data.test", "scope", scope),
					testAccCheckResourceExists(resourceNameCollectionData, scope),
					resource.TestCheckResourceAttr("itsi_collection_data.test", "entry.#", "2"),
					resource.TestCheckResourceAttr("itsi_collection_data.test", "entry.0.id", "apple"),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectKnownValue("itsi_collection_data.test", tfjsonpath.New("entry").AtSliceIndex(0).AtMapKey("id"), knownvalue.StringExact("apple")),
						plancheck.ExpectUnknownValue("itsi_collection_data.test", tfjsonpath.New("entry").AtSliceIndex(1).AtMapKey("id")),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectKnownValue("itsi_collection_data.test", tfjsonpath.New("entry").AtSliceIndex(0).AtMapKey("id"), knownvalue.StringExact("apple")),
						plancheck.ExpectKnownValue("itsi_collection_data.test", tfjsonpath.New("entry").AtSliceIndex(1).AtMapKey("id"), knownvalue.NotNull()),
					},
				},
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_collection_data.test", "scope", scope),
					testAccCheckResourceExists(resourceNameCollectionData, scope),
					resource.TestCheckResourceAttr("itsi_collection_data.test", "entry.#", "2"),
					resource.TestCheckResourceAttr("itsi_collection_data.test", "entry.0.id", "apple"),
					resource.TestCheckResourceAttr("itsi_collection_data.test", "entry.1.id", "banana"),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectKnownValue("itsi_collection_data.test", tfjsonpath.New("entry").AtSliceIndex(0).AtMapKey("id"), knownvalue.StringExact("apple")),
						plancheck.ExpectKnownValue("itsi_collection_data.test", tfjsonpath.New("entry").AtSliceIndex(1).AtMapKey("id"), knownvalue.StringExact("banana")),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectKnownValue("itsi_collection_data.test", tfjsonpath.New("entry").AtSliceIndex(0).AtMapKey("id"), knownvalue.StringExact("apple")),
						plancheck.ExpectKnownValue("itsi_collection_data.test", tfjsonpath.New("entry").AtSliceIndex(1).AtMapKey("id"), knownvalue.StringExact("banana")),
					},
				},
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				ExpectError:              regexp.MustCompile(`Duplicate entry ID`),
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				ExpectError:              regexp.MustCompile(`Duplicate entry ID`),
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				ExpectError:              regexp.MustCompile(`Duplicate collection items`),
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_collection_data.test", "scope", scope),
					testAccCheckResourceExists(resourceNameCollectionData, scope),
					resource.TestCheckResourceAttr("itsi_collection_data.test", "entry.#", "50"),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}
