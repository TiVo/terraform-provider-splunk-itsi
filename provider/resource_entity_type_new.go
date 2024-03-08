package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

const (
	entityTypeDefaultDashboardType = "navigation_link"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &resourceEntityType{}
)

type resourceEntityType struct {
	client models.ClientConfig
}

func NewResourceEntityType() resource.Resource {
	return &resourceEntityType{}
}

func (r *resourceEntityType) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(models.ClientConfig)
	if !ok {
		tflog.Error(ctx, "Unable to prepare client")
		resp.Diagnostics.AddError("Unable to prepare client", "invalid provider data")
		return
	}
	r.client = client
}

func (r *resourceEntityType) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_entity_type"
}

func (r *resourceEntityType) dashboardDrilldownSchema() schema.SetNestedBlock {
	return schema.SetNestedBlock{
		MarkdownDescription: util.Dedent(`
			An array of dashboard drilldown objects.
			Each dashboard drilldown defines an internal or external resource you specify with a URL and parameters
			that map to one of an entity fields. The parameters are passed to the resource when you open the URL.
		`),
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"title": schema.StringAttribute{
					MarkdownDescription: "The name of the dashboard.",
					Required:            true,
				},
				"base_url": schema.StringAttribute{
					MarkdownDescription: util.Dedent(`
						An internal or external URL that points to the dashboard.
						This setting exists because for internal purposes, navigation suggestions are treated as dashboards.
						This setting is only required if is_splunk_dashboard is false.
					`),
					Optional: true,
					Computed: true,
					Default:  stringdefault.StaticString(""),
				},
				"dashboard_id": schema.StringAttribute{
					MarkdownDescription: "A unique identifier for the xml dashboard.",
					Optional:            true,
					Computed:            true,
					Default:             stringdefault.StaticString(""),
				},
				"dashboard_type": schema.StringAttribute{
					MarkdownDescription: util.Dedent(fmt.Sprintf(`
						The type of dashboard being added.
						The following options are available:
						* xml_dashboard - a Splunk XML dashboard. Any dashboards you add must be of this type.
						* navigation_link - a navigation URL. Should be used when base_url is specified.
						Defaults to %s.
					`, entityTypeDefaultDashboardType)),
					Optional: true,
					Computed: true,
					Default:  stringdefault.StaticString(entityTypeDefaultDashboardType),
					Validators: []validator.String{
						stringvalidator.OneOf("xml_dashboard", "navigation_link"),
					},
				},
				"params": schema.MapAttribute{
					MarkdownDescription: "A set of parameters for the entity dashboard drilldown that provide a mapping of a URL parameter and its alias.",
					Optional:            true,
					Computed:            true,
					ElementType:         types.StringType,
					Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
				},
			},
		},
	}
}

func (r *resourceEntityType) dataDrilldownSchema() schema.SetNestedBlock {
	return schema.SetNestedBlock{
		MarkdownDescription: util.Dedent(`
			An array of data drilldown objects.
			Each data drilldown defines filters for raw data associated with entities that belong to the entity type.
		`),
		NestedObject: schema.NestedBlockObject{
			Blocks: map[string]schema.Block{
				"static_filters": schema.SetNestedBlock{
					MarkdownDescription: util.Dedent(`
						Further filter down to the raw data associated with the entity
						based on a set of selected entity alias or informational fields.
					`),
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"data_field": schema.StringAttribute{
								MarkdownDescription: "Data field.",
								Required:            true,
							},
							"entity_field": schema.StringAttribute{
								MarkdownDescription: "Entity field.",
								Required:            true,
							},
						},
					},
					Validators: []validator.Set{
						setvalidator.SizeAtLeast(1),
					},
				},
			},
			Attributes: map[string]schema.Attribute{
				"title": schema.StringAttribute{
					MarkdownDescription: "The name of the drilldown.",
					Required:            true,
				},
				"type": schema.StringAttribute{
					MarkdownDescription: "Type of raw data to associate with. Must be either metrics or events.",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.OneOf("events", "metrics"),
					},
				},
				"static_filters": schema.MapAttribute{
					MarkdownDescription: "Filter down to a subset of raw data associated with the entity using static information like sourcetype.",
					Optional:            true,
					Computed:            true,
					ElementType:         types.StringType,
					Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
				},
			},
		},
	}
}

func (r *resourceEntityType) vitalMetricSchema() schema.SetNestedBlock {
	return schema.SetNestedBlock{
		MarkdownDescription: util.Dedent(`
			An set of vital metric objects. Vital metrics are statistical calculations based on
			SPL searches that represent the overall health of entities of that type.
		`),
		NestedObject: schema.NestedBlockObject{
			Blocks: map[string]schema.Block{
				"alert_rule": schema.SetNestedBlock{
					NestedObject: schema.NestedBlockObject{
						Blocks: map[string]schema.Block{
							"entity_filter": schema.SetNestedBlock{
								NestedObject: schema.NestedBlockObject{
									Attributes: map[string]schema.Attribute{
										"field": schema.StringAttribute{
											Required: true,
										},
										"field_type": schema.StringAttribute{
											MarkdownDescription: "Takes values alias, info or title specifying in which category of fields the field attribute is located.",
											Required:            true,
											Validators: []validator.String{
												stringvalidator.OneOf("alias", "entity_type", "info", "title"),
											},
										},
										"value": schema.StringAttribute{
											Required: true,
										},
									},
								},
							},
						},
						Attributes: map[string]schema.Attribute{
							"critical_threshold": schema.Int64Attribute{
								Required: true,
							},
							"warning_threshold": schema.Int64Attribute{
								Required: true,
							},
							"cron_schedule": schema.StringAttribute{
								MarkdownDescription: "Frequency of the alert search",
								Required:            true,
							},
							"is_enabled": schema.BoolAttribute{
								MarkdownDescription: "Indicates if the alert rule is enabled.",
								Optional:            true,
								Computed:            true,
								Default:             booldefault.StaticBool(false),
							},
							"suppress_time": schema.StringAttribute{
								MarkdownDescription: "Frequency of the alert search",
								Required:            true,
								Optional:            true,
								Computed:            true,
								Default:             stringdefault.StaticString("0"),
							},
						},
					},
				},
			},
			Attributes: map[string]schema.Attribute{
				"metric_name": schema.StringAttribute{
					MarkdownDescription: util.Dedent(`
						The title of the vital metric. When creating vital metrics,
						it's a best practice to include the aggregation method and the name of the metric being calculated.
						For example, Average CPU usage.
					`),
					Required: true,
				},
				"search": schema.StringAttribute{
					MarkdownDescription: util.Dedent(`
						The search that computes the vital metric. The search must specify the following fields:
						- val for the value of the metric.
						- _time because the UI attempts to render changes over time. You can achieve this by adding span={time} to your search.
						- Fields as described in the split_by_fields configuration of this vital metric.
						For example, your search should be split by host,region if the split_by_fields configuration is [ "host", "region" ].
					`),
					Required: true,
				},
				"matching_entity_fields": schema.MapAttribute{
					MarkdownDescription: util.Dedent(`
						Specifies the aliases of an entity to use to match with the fields specified by the fields that the search configuration is split on.
						Make sure the value matches the split by fields in the actual search.
						For example:
							- search = "..... by InstanceId, region"
							- matching_entity_fields = {instance_id = "InstanceId", zone = "region"}.
					`),
					ElementType: types.StringType,
					Required:    true,
				},
				"is_key": schema.BoolAttribute{
					MarkdownDescription: util.Dedent(`
						Indicates if the vital metric specified is a key metric.
						A key metric calculates the distribution of entities associated with the entity type to indicate the overall health of the entity type.
						The key metric is rendered as a histogram in the Infrastructure Overview. Only one vital metric can have is_key set to true.
					`),
					Optional: true,
					Computed: true,
					Default:  booldefault.StaticBool(false),
				},
				"unit": schema.StringAttribute{
					MarkdownDescription: "The unit of the vital metric. For example, KB/s.",
					Optional:            true,
					Computed:            true,
					Default:             stringdefault.StaticString(""),
				},
			},
		},
	}
}

func (r *resourceEntityType) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: util.Dedent(`
			An entity_type defines how to classify a type of data source.
			For example, you can create a Linux, Windows, Unix/Linux add-on, VMware, or Kubernetes entity type.
			An entity type can include zero or more data drilldowns and zero or more dashboard drilldowns.
			You can use a single data drilldown or dashboard drilldown for multiple entity types.
		`),
		Attributes: map[string]schema.Attribute{
			"title": schema.StringAttribute{
				MarkdownDescription: "The name of the entity type.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the entity type.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
		},
		Blocks: map[string]schema.Block{
			"dashboard_drilldown": r.dashboardDrilldownSchema(),
			"data_drilldown":      r.dataDrilldownSchema(),
			"vital_metric":        r.vitalMetricSchema(),
		},
	}
}

func (r *resourceEntityType) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

}

func (r *resourceEntityType) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
}

func (r *resourceEntityType) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *resourceEntityType) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

func (r *resourceEntityType) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
}
