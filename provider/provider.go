package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
				Description: "HTTP timeout in seconds for CRUD requests to Splunk/ITSI API. 0 means no timeout. (Terraform resource timeouts still apply)",
			},
			"insecure": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether the API should be accessed without verifying the TLS certificate.",
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"itsi_kpi_base_search": DatasourceKPIBaseSearch(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"itsi_kpi_base_search": ResourceKPIBaseSearch(),
			"itsi_entity_type":     ResourceEntityType(),
			"itsi_service":         ResourceService(),
			"itsi_neap":            ResourceNotableEventAggregationPolicy(),
		},
	}
}

func providerConfigure(c context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	client := models.ClientConfig{}
	client.BearerToken = d.Get("access_token").(string)
	client.User = d.Get("user").(string)
	client.Password = d.Get("password").(string)
	client.Host = d.Get("host").(string)
	if port, ok := d.Get("port").(int); ok {
		client.Port = port
	} else {
		client.Port = defaultPort
	}
	if timeout, ok := d.Get("timeout").(int); ok {
		client.Timeout = timeout
	} else {
		client.Timeout = defaultTimeout
	}
	client.SkipTLS = d.Get("insecure").(bool)
	client.RetryPolicy = retryPolicy

	client.Concurrency = clientConcurrency

	if client.BearerToken == "" && (client.User == "" || client.Password == "") {
		return nil, diag.Errorf("ITSI provider configuration failed: missing values for Splunk API access_token or user/password")
	}

	return client, nil
}
