package provider

import (
	"context"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
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
	clientConcurrency = 15
	defaultTimeout    = 60
	defaultPort       = 8089
	cacheSize         = 1000
)

var retryPolicy backoff.Policy = backoff.Exponential(
	backoff.WithMinInterval(500*time.Millisecond),
	backoff.WithMaxInterval(time.Minute),
	backoff.WithJitterFactor(0.05),
	backoff.WithMaxRetries(0),
)

func init() {
	models.InitItsiApiLimiter(clientConcurrency)
	InitSplunkSearchLimiter(clientConcurrency)
	if os.Getenv("TF_LOG") == "true" {
		models.Verbose = true
	}
	models.Cache = models.NewCache(cacheSize)
}

// Ensure the implementation satisfies the expected interfaces
var (
	_ provider.ProviderWithFunctions = &itsiProvider{}
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
				Optional:            true,
				MarkdownDescription: "Bearer token used to authenticate HTTP requests to Splunk API",
			},
			"user": schema.StringAttribute{
				Optional: true,
			},
			"password": schema.StringAttribute{
				Optional: true,
			},
			"timeout": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "HTTP timeout in seconds for CRUD requests to Splunk/ITSI API. 0 means no timeout. (Terraform resource timeouts still apply)",
			},
			"insecure": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether the API should be accessed without verifying the TLS certificate.",
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
			"Unknown Splunk/ITSI REST API Host",
			"The provider cannot create the Splunk/ITSI API client as there is an unknown configuration value for the Splunk REST API host. "+
				"Either target apply the source of the value first or set the value statically in the configuration.",
		)
	}

	if config.Port.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("port"),
			"Unknown Splunk/ITSI REST API Port",
			"The provider cannot create the Splunk/ITSI API client as there is an unknown configuration value for the Splunk REST API port. "+
				"Either target apply the source of the value first or set the value statically in the configuration.")
	}

	if resp.Diagnostics.HasError() {
		return
	}

	host := config.Host.ValueString()
	port := config.Port.ValueInt64()
	accessToken := config.AccessToken.ValueString()
	user := config.User.ValueString()
	password := config.Password.ValueString()
	insecure := config.InsecureSkipVerify.ValueBool()
	var timeout int64 = defaultTimeout

	if port == 0 {
		port = defaultPort
	}

	if !config.Timeout.IsNull() {
		timeout = config.Timeout.ValueInt64()
	}

	client := models.ClientConfig{}
	client.BearerToken = accessToken
	client.User = user
	client.Password = password
	client.Host = host
	client.Port = int(port)
	client.Timeout = int(timeout)
	client.SkipTLS = insecure
	client.RetryPolicy = retryPolicy
	client.Concurrency = clientConcurrency

	if client.BearerToken == "" && (client.User == "" || client.Password == "") {
		resp.Diagnostics.AddError(
			"ITSI provider configuration failed",
			"missing values for Splunk API access_token or user/password")
		return
	}

	// Make the Splunk/ITSI client available during DataSource and Resource
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
		func() datasource.DataSource {
			return NewDataSourceCollection()
		},
		func() datasource.DataSource {
			return NewDataSourceCollectionData()
		},
		func() datasource.DataSource {
			return NewDataSourceKpiThresholdTemplate()
		},
		func() datasource.DataSource {
			return NewDataSourceSplunkSearch()
		},
		func() datasource.DataSource {
			return NewKpiBaseSearchDataSource()
		},
	}
}

// Resources defines the resources implemented in the provider.
func (p *itsiProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		func() resource.Resource {
			return NewResouceCollection()
		},
		func() resource.Resource {
			return NewResourceCollectionData()
		},
		func() resource.Resource {
			return NewResourceEntity()
		},
		func() resource.Resource {
			return NewResourceEntityType()
		},
		func() resource.Resource {
			return NewResourceKpiThresholdTemplate()
		},
		func() resource.Resource {
			return NewKpiBaseSearch()
		},
		func() resource.Resource {
			return NewResourceService()
		},
		func() resource.Resource {
			return NewResourceNEAP()
		},
		func() resource.Resource {
			return NewResourceKPIThresholdTemplateManifest()
		},
	}
}

func (p *itsiProvider) Functions(_ context.Context) []func() function.Function {
	return []func() function.Function{}
}
