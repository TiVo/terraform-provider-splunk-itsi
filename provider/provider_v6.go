package provider

import (
	"context"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/lestrrat-go/backoff/v2"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

const (
	client_concurrency = 15
)

var retryPolicy backoff.Policy = backoff.Exponential(
	backoff.WithMinInterval(500*time.Millisecond),
	backoff.WithMaxInterval(time.Minute),
	backoff.WithJitterFactor(0.05),
	backoff.WithMaxRetries(0),
)

func init() {
	models.InitItsiApiLimiter(client_concurrency)
	InitSplunkSearchLimiter(client_concurrency)
	if os.Getenv("TF_LOG") == "true" {
		models.Verbose = true
	}
	models.Cache = models.NewCache(1000)
}

// Ensure the implementation satisfies the expected interfaces
var (
	_ provider.Provider = &itsiProvider{}
)

type itsiProviderModel struct {
	Host               types.String `tfsdk:"host"`
	Port               types.Int64  `tfsdk:"port"`
	AccessToken        types.String `tfsdk:"access_token"`
	User               types.String `tfsdk:"user"`
	Password           types.String `tfsdk:"password"`
	Timeout            types.Int64  `tfsdk:"timeout"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure"`
}

// New is a helper function to simplify provider server and testing implementation.
func New() provider.Provider {
	return &itsiProvider{}
}

// itsiProvider is the provider implementation.
type itsiProvider struct{}

// Metadata returns the provider type name.
func (p *itsiProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "itsi"
}

// Schema defines the provider-level schema for configuration data.
func (p *itsiProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Required: true,
			},
			"port": schema.Int64Attribute{
				Optional: true,
			},
			"access_token": schema.StringAttribute{
				Optional:    true,
				Description: "Bearer token used to authenticate HTTP requests to Splunk API",
			},
			"user": schema.StringAttribute{
				Optional: true,
			},
			"password": schema.StringAttribute{
				Optional: true,
			},
			"timeout": schema.Int64Attribute{
				Optional:    true,
				Description: "HTTP timeout in seconds for CRUD requests to Splunk/ITSI API. 0 means no timeout. (Terraform resource timeouts still apply)",
			},
			"insecure": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether the API should be accessed without verifying the TLS certificate.",
			},
		},
		Blocks: map[string]schema.Block{},
	}
}

// Configure prepares a ITSI API client for data sources and resources.
func (p *itsiProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config itsiProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown Inventory service Host",
			"The provider cannot create the Inventory API client as there is an unknown configuration value for the Inventory API host. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the INVENTORY_HOST environment variable.",
		)
	}

	if config.Port.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("port"),
			"Unknown Inventory service Port",
			"The provider cannot create the Inventory API client as there is an unknown configuration value for the Inventory API port. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the INVENTORY_PORT environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	host := config.Host.ValueString()
	port := config.Port.ValueInt64()
	access_token := config.AccessToken.ValueString()
	user := config.User.ValueString()
	password := config.Password.ValueString()
	insecure := config.InsecureSkipVerify.ValueBool()
	var timeout int64 = 60

	if port == 0 {
		port = 8089
	}

	if !config.Timeout.IsNull() {
		timeout = config.Timeout.ValueInt64()
	}

	client := models.ClientConfig{}
	client.BearerToken = access_token
	client.User = user
	client.Password = password
	client.Host = host
	client.Port = int(port)
	client.Timeout = int(timeout)
	client.SkipTLS = insecure
	client.RetryPolicy = retryPolicy
	client.Concurrency = client_concurrency

	if client.BearerToken == "" && (client.User == "" || client.Password == "") {
		resp.Diagnostics.AddError(
			"ITSI provider configuration failed",
			"missing values for Splunk API access_token or user/password")
		return
	}

	// Make the Inventory client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Configured ITSI Provider", map[string]any{"success": true})
}

// DataSources defines the data sources implemented in the provider.
func (p *itsiProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource {
			return NewDataSourceEntityType()
		},
	}
}

// Resources defines the resources implemented in the provider.
func (p *itsiProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}
