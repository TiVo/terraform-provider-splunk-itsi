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

func testAccCheckResourceExists(s *terraform.State, resourcetype resourceName, resourceTitle string) (err error) {
	ok, err := checkResourceExists(s, resourcetype, resourceTitle)
	if err == nil && !ok {
		err = fmt.Errorf("Resource %s %s does not exist", resourcetype, resourceTitle)
	}
	return
}

func testAccCheckResourceDestroy(s *terraform.State, resourcetype resourceName, resourceTitle string) (err error) {
	ok, err := checkResourceExists(s, resourcetype, resourceTitle)
	if err == nil && ok {
		err = fmt.Errorf("Resource %s %s still exists", resourcetype, resourceTitle)
	}
	return
}

func checkResourceExists(s *terraform.State, resourcetype resourceName, resourceTitle string) (bool, error) {
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "itsi_"+string(resourcetype) && rs.Primary.Attributes["title"] == resourceTitle {
			return resourceExists(resourcetype, resourceTitle)
		}
	}
	return false, nil
}

func resourceExists(resourcetype resourceName, resourceTitle string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(60)*time.Second)
	defer cancel()

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
