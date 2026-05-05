package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	rsschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

// Ensure interface compliance
var _ datasource.DataSource = &dataSourceKpiBaseSearch{}
var _ datasource.DataSourceWithConfigure = &dataSourceKpiBaseSearch{}

type dataSourceKpiBaseSearch struct {
	client models.ClientConfig
}

type dataSourceKpiBaseSearchState struct {
	ID    types.String `tfsdk:"id" json:"_key"`
	Title types.String `tfsdk:"title" json:"title"`

	Metrics []Metric `tfsdk:"metrics"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func NewKpiBaseSearchDataSource() datasource.DataSource {
	return &dataSourceKpiBaseSearch{}
}

func (d *dataSourceKpiBaseSearch) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	configureDataSourceMetadata(req, resp, datasourceNameKPIBaseSearch)
}

func (d *dataSourceKpiBaseSearch) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	configureDataSourceClient(ctx, datasourceNameKPIBaseSearch, req, &d.client, resp)
}

// computedSetNestedBlock converts a resource SetNestedBlock into a datasource
// SetNestedBlock where every attribute is Computed-only, preserving descriptions.
func computedSetNestedBlock(b rsschema.SetNestedBlock) schema.SetNestedBlock {
	attrs := make(map[string]schema.Attribute, len(b.NestedObject.Attributes))
	for name, attr := range b.NestedObject.Attributes {
		switch a := attr.(type) {
		case rsschema.StringAttribute:
			attrs[name] = schema.StringAttribute{Computed: true, Description: a.Description}
		case rsschema.Float64Attribute:
			attrs[name] = schema.Float64Attribute{Computed: true, Description: a.Description}
		}
	}
	return schema.SetNestedBlock{
		NestedObject: schema.NestedBlockObject{
			Attributes: attrs,
		},
	}
}

// KpiSearchDataSource schema
func (d *dataSourceKpiBaseSearch) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to get the ID of an available KPI Base Search.",
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx),
			"metrics":  computedSetNestedBlock(kpiBaseSearchMetricsBlock()),
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
			},
			"title": schema.StringAttribute{
				Required:    true,
				Description: "The title of the KPI Base Search.",
			},
		},
	}
}

// Read data into the Terraform state
func (d *dataSourceKpiBaseSearch) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read KPI BS data source")
	var config dataSourceKpiBaseSearchState

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	timeouts := config.Timeouts
	readTimeout, diags := timeouts.Read(ctx, tftimeout.Read)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	title := config.Title.ValueString()
	base := kpiBaseSearchBase(d.client, config.ID.ValueString(), title)
	b, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read KPI BS object", err.Error())
		return
	}
	if b == nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("title"),
			"KPI BS not found",
			fmt.Sprintf("KPI BS %q not found", title))
		return
	}

	kbsState, diags := newAPIParser(b, new(kpiBaseSearchParseWorkflow)).parse(ctx, b)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	state := &dataSourceKpiBaseSearchState{
		ID:       kbsState.ID,
		Title:    kbsState.Title,
		Metrics:  kbsState.Metrics,
		Timeouts: timeouts,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)

	tflog.Debug(ctx, "Finished reading KPI BS data source", map[string]any{"success": true})
}
