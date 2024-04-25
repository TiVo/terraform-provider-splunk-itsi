package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

var providerFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"itsi": func() (tfprotov6.ProviderServer, error) {
		return providerserver.NewProtocol6(New())(), nil
	},
}

var clientConfig models.ClientConfig

func init() {
	clientConfig = itsiClientConfig()
}

func itsiClientConfig() (client models.ClientConfig) {
	client.BearerToken = os.Getenv(envITSIAccessToken)
	client.User = os.Getenv(envITSIUser)
	client.Password = os.Getenv(envITSIPassword)
	client.Host = os.Getenv(envITSIHost)

	port, _ := strconv.Atoi(os.Getenv(envITSIPort))
	if port == 0 {
		port = defaultPort
	}

	client.Port = port
	client.Timeout = defaultTimeout

	insecure := util.Atob(os.Getenv(envITSIInsecure))
	client.SkipTLS = insecure
	client.RetryPolicy = retryPolicy
	client.Concurrency = clientConcurrency
	return
}

func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
}

func testDataSourceSchema[T datasource.DataSource](t *testing.T, d T) {
	ctx := context.Background()
	schemaRequest := datasource.SchemaRequest{}
	schemaResponse := &datasource.SchemaResponse{}

	d.Schema(ctx, schemaRequest, schemaResponse)
	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Failed to validate schema: %s", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)
	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func testResourceSchema[T resource.Resource](t *testing.T, r T) {
	ctx := context.Background()
	schemaRequest := resource.SchemaRequest{}
	schemaResponse := &resource.SchemaResponse{}

	r.Schema(ctx, schemaRequest, schemaResponse)
	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Failed to validate schema: %s", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)
	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func TestProviderSchema(t *testing.T) {
	ctx := context.Background()
	schemaRequest := provider.SchemaRequest{}
	schemaResponse := &provider.SchemaResponse{}

	New().Schema(ctx, schemaRequest, schemaResponse)
	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Failed to validate schema: %s", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)
	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func testAccCheckResourceExists(resourcetype resourceName, resourceTitle string) testingresource.TestCheckFunc {
	return func(s *terraform.State) (err error) {
		ok, err := checkResourceExists(s, resourcetype, resourceTitle)
		if err == nil && !ok {
			err = fmt.Errorf("Resource %s %s does not exist", resourcetype, resourceTitle)
		}
		return
	}
}

func testAccCheckResourceDestroy(resourcetype resourceName, resourceTitle string) testingresource.TestCheckFunc {
	return func(s *terraform.State) (err error) {
		ok, err := checkResourceExists(s, resourcetype, resourceTitle)
		if err == nil && ok {
			err = fmt.Errorf("Resource %s %s still exists", resourcetype, resourceTitle)
		}
		return
	}
}

func checkResourceExists(s *terraform.State, resourcetype resourceName, resourceTitle string) (bool, error) {
	var titleAttribute string
	switch resourcetype {
	case resourceNameCollection:
		titleAttribute = "name"
	case resourceNameCollectionData:
		titleAttribute = "scope"
	default:
		titleAttribute = "title"
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type == "itsi_"+string(resourcetype) && rs.Primary.Attributes[titleAttribute] == resourceTitle {
			return resourceExists(resourcetype, resourceTitle)
		}
	}
	return false, nil
}

func resourceExists(resourcetype resourceName, resourceTitle string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(60)*time.Second)
	defer cancel()

	switch resourcetype {
	case resourceNameCollection, resourceNameCollectionData:
		return collectionModelObjectExists(ctx, resourcetype, resourceTitle)
	default:
		return baseModelObjectExists(ctx, resourcetype, resourceTitle)
	}
}

func collectionModelObjectExists(ctx context.Context, resourceType resourceName, resourceTitle string) (bool, error) {
	if !(resourceType == resourceNameCollection || resourceType == resourceNameCollectionData) {
		return false, fmt.Errorf("resource type %s is not a collection model type.", resourceType)
	}

	switch resourceType {
	case resourceNameCollection:
		collectionID, diags := collectionIDModelFromString(resourceTitle)
		if diags.HasError() {
			return false, fmt.Errorf("failed to parse collection ID from title %s: %s", resourceTitle, diags)
		}
		ok, diags := NewCollectionAPI(collectionID, clientConfig).CollectionExists(ctx, false)
		if diags.HasError() {
			return false, fmt.Errorf("failed to check if collection %s exists: %s", resourceTitle, diags)
		}
		return ok, nil
	case resourceNameCollectionData:
		collectionID, scope, diags := collectionIDModelAndScopeFromString(resourceTitle + ":" + resourceTitle)
		if diags.HasError() {
			return false, fmt.Errorf("failed to parse collection ID and scope from title %s: %s", resourceTitle, diags)
		}
		collectionAPI := NewCollectionAPI(collectionID, clientConfig)

		ok, diags := collectionAPI.CollectionExists(ctx, false)
		if diags.HasError() {
			return false, fmt.Errorf("failed to check if collection %s exists: %s", resourceTitle, diags)
		}
		if !ok {
			return false, nil
		}
		obj, diags := collectionAPI.Query(ctx, fmt.Sprintf(`{"_scope":"%s"}`, scope), []string{}, 0)
		if diags.HasError() {
			return false, fmt.Errorf("failed to query collection data %s: %s", resourceTitle, diags)
		}
		arr, ok := obj.([]interface{})
		if !ok {
			return false, fmt.Errorf("failed to query collection data %s: %s", resourceTitle, diags)
		}
		return len(arr) > 0, nil
	default:
		return false, fmt.Errorf("not implemented")
	}
}

func baseModelObjectExists(ctx context.Context, resourcetype resourceName, resourceTitle string) (bool, error) {
	base := models.NewBase(clientConfig, "", resourceTitle, string(resourcetype))
	b, err := base.Find(ctx)
	if err != nil {
		return false, err
	}
	return b != nil, nil
}

func testAccResourceTitle(title string) string {
	return "TestAcc_" + title
}
