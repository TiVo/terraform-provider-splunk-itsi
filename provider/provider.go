package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/lestrrat-go/backoff/v2"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

func init() {
	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown

	// Customize the content of descriptions when output. For example you can add defaults on
	// to the exported descriptions if present.
	schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
		desc := s.Description
		if s.Default != nil {
			desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
		}
		return strings.TrimSpace(desc)
	}
}

// Provider returns the ITSI provider
func Provider() *schema.Provider {
	return &schema.Provider{
		ConfigureContextFunc: providerConfigure,
		Schema: map[string]*schema.Schema{
			"host": {
				Type:     schema.TypeString,
				Required: true,
			},
			"port": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  8089,
			},
			"access_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Bearer token used to authenticate HTTP requests to Splunk API",
			},
			"user": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"password": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     60,
				Description: "HTTP timeout in seconds for CRUD requests to Splunk/ITSI API. 0 means no timeout. (Terraform resource timeouts still apply)",
			},
			"insecure": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether the API should be accessed without verifying the TLS certificate.",
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"itsi_splunk_lookup":            DatasourceSplunkLookup(),
			"itsi_splunk_search":            DatasourceSplunkSearch(),
			"itsi_entity_type":              DatasourceEntityType(),
			"itsi_kpi_threshold_template":   DatasourceKPIThresholdTemplate(),
			"itsi_splunk_collection_fields": DatasourceCollectionFields(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"itsi_kpi_threshold_template":    ResourceKPIThresholdTemplate(),
			"itsi_kpi_base_search":           ResourceKPIBaseSearch(),
			"itsi_entity":                    ResourceEntity(),
			"itsi_entity_type":               ResourceEntityType(),
			"itsi_service":                   ResourceService(),
			"itsi_splunk_collection":         ResourceCollection(),
			"itsi_splunk_collection_entry":   ResourceCollectionEntry(),
			"itsi_splunk_collection_entries": ResourceCollectionEntries(),
		},
	}
}

func providerConfigure(c context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	client := models.ClientConfig{}
	client.BearerToken = d.Get("access_token").(string)
	client.User = d.Get("user").(string)
	client.Password = d.Get("password").(string)
	client.Host = d.Get("host").(string)
	client.Port = d.Get("port").(int)
	client.Timeout = d.Get("timeout").(int)
	client.SkipTLS = d.Get("insecure").(bool)
	client.RetryPolicy = backoff.Exponential(
		backoff.WithMinInterval(500*time.Millisecond),
		backoff.WithMaxInterval(time.Minute),
		backoff.WithJitterFactor(0.05),
		backoff.WithMaxRetries(0),
	)

	client.Concurrency = 15
	models.InitItsiApiLimiter(client.Concurrency)
	InitSplunkSearchLimiter(client.Concurrency)
	if os.Getenv("TF_LOG") == "true" {
		models.Verbose = true
	}

	if client.BearerToken == "" && (client.User == "" || client.Password == "") {
		return nil, diag.Errorf("ITSI provider configuration failed: missing values for Splunk API access_token or user/password")
	}

	models.Cache = models.NewCache(1000)
	return client, nil
}
