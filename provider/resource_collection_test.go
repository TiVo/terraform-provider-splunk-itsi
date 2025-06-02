package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/tivo/terraform-provider-splunk-itsi/util"
)

func TestResourceCollectionSchema(t *testing.T) {
	testResourceSchema(t, new(resourceCollection))
}

func TestCollectionIDModelAndScopeFromString(t *testing.T) {
	collection := "mycollection"
	owner := "myowner"
	app := "myapp"
	scope := "myscope"
	key := owner + "/" + app + "/" + collection + ":" + scope + ":" + scope

	collectionModel, s, d := collectionIDModelAndScopeFromString(key)
	if d.HasError() {
		t.Fatalf("failed to run collectionIDModelAndScopeFromString with key %s", key)
	}

	if collectionModel.Name != types.StringValue(collection) {
		t.Errorf("Expected collection %s, got %s", collection, collectionModel.Name)
	}
	if collectionModel.Owner != types.StringValue(owner) {
		t.Errorf("Expected owner %s, got %s", owner, collectionModel.Owner)
	}
	if collectionModel.App != types.StringValue(app) {
		t.Errorf("Expected app %s, got %s", app, collectionModel.App)
	}
	if s != scope+":"+scope {
		t.Errorf("Expected scope %s, got %s", scope+":"+scope, s)
	}

	key = owner + "/" + app + "/" + collection + ":" + scope

	collectionModel, s, d = collectionIDModelAndScopeFromString(key)
	if d.HasError() {
		t.Fatalf("failed to run collectionIDModelAndScopeFromString with key %s", key)
	}

	if collectionModel.Name != types.StringValue(collection) {
		t.Errorf("Expected collection %s, got %s", collection, collectionModel.Name)
	}
	if collectionModel.Owner != types.StringValue(owner) {
		t.Errorf("Expected owner %s, got %s", owner, collectionModel.Owner)
	}
	if collectionModel.App != types.StringValue(app) {
		t.Errorf("Expected app %s, got %s", app, collectionModel.App)
	}
	if s != scope {
		t.Errorf("Expected scope %s, got %s", scope, s)
	}

	key = app + "/" + collection + ":" + scope

	collectionModel, s, d = collectionIDModelAndScopeFromString(key)
	if d.HasError() {
		t.Fatalf("failed to run collectionIDModelAndScopeFromString with key %s", key)
	}

	if collectionModel.Name != types.StringValue(collection) {
		t.Errorf("Expected collection %s, got %s", collection, collectionModel.Name)
	}
	if collectionModel.Owner != types.StringValue(collectionDefaultUser) {
		t.Errorf("Expected owner %s, got %s", owner, collectionModel.Owner)
	}
	if collectionModel.App != types.StringValue(app) {
		t.Errorf("Expected app %s, got %s", app, collectionModel.App)
	}
	if s != scope {
		t.Errorf("Expected scope %s, got %s", scope, s)
	}

	key = collection + ":" + scope

	collectionModel, s, d = collectionIDModelAndScopeFromString(key)
	if d.HasError() {
		t.Fatalf("failed to run collectionIDModelAndScopeFromString with key %s", key)
	}

	if collectionModel.Name != types.StringValue(collection) {
		t.Errorf("Expected collection %s, got %s", collection, collectionModel.Name)
	}
	if collectionModel.Owner != types.StringValue(collectionDefaultUser) {
		t.Errorf("Expected owner %s, got %s", collectionDefaultUser, collectionModel.Owner)
	}
	if collectionModel.App != types.StringValue(collectionDefaultApp) {
		t.Errorf("Expected app %s, got %s", collectionDefaultApp, collectionModel.App)
	}
	if s != scope {
		t.Errorf("Expected scope %s, got %s", scope, s)
	}

	key = collection

	collectionModel, s, d = collectionIDModelAndScopeFromString(key)
	if d.HasError() {
		t.Fatalf("failed to run collectionIDModelAndScopeFromString with key %s", key)
	}

	if collectionModel.Name != types.StringValue(collection) {
		t.Errorf("Expected collection %s, got %s", collection, collectionModel.Name)
	}
	if collectionModel.Owner != types.StringValue(collectionDefaultUser) {
		t.Errorf("Expected owner %s, got %s", collectionDefaultUser, collectionModel.Owner)
	}
	if collectionModel.App != types.StringValue(collectionDefaultApp) {
		t.Errorf("Expected app %s, got %s", collectionDefaultApp, collectionModel.App)
	}
	if s != collectionDefaultScope {
		t.Errorf("Expected scope %s, got %s", collectionDefaultScope, s)
	}

	// Test with an invalid ID
	_, _, d = collectionIDModelAndScopeFromString("")
	if !d.HasError() {
		t.Fatal("Expected an error for empty ID, got none")
	}
}

func TestResourceCollectionPlan(t *testing.T) {
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

					resource "itsi_splunk_collection" "test" {
						name = "example_collection"
					}
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccResourceCollectionLifecycle(t *testing.T) {
	t.Parallel()
	var testAccCollectionLifecycle_collectionName = testAccResourceTitle("ResourceCollectionLifecycle_collection_test")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckResourceDestroy(resourceNameCollection, testAccCollectionLifecycle_collectionName),
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_splunk_collection.test", "name", testAccCollectionLifecycle_collectionName),
					testAccCheckResourceExists(resourceNameCollection, testAccCollectionLifecycle_collectionName),
				),
			},
			{
				ProtoV6ProviderFactories: providerFactories,
				ConfigDirectory:          config.TestStepDirectory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("itsi_splunk_collection.test", "name", testAccCollectionLifecycle_collectionName),
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
