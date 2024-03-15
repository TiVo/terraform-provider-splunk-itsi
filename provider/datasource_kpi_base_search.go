package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
}

func NewKpiBaseSearchDataSource() datasource.DataSource {
	return &dataSourceKpiBaseSearch{}
}

func (d *dataSourceKpiBaseSearch) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kpi_base_search"
}

func (d *dataSourceKpiBaseSearch) Configure(ctx context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(models.ClientConfig)
	if !ok {
		tflog.Error(ctx, "Unable to prepare client")
		return
	}
	d.client = client
}

// KpiSearchDataSource schema
func (d *dataSourceKpiBaseSearch) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to get the ID of an available KPI Base Search.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource",
				Computed:            true,
			},
			"title": schema.StringAttribute{
				Required:    true,
				Description: "Name of this KPI base search.",
			},
		},
	}
}

// Read data into the Terraform state
func (d *dataSourceKpiBaseSearch) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read KPI BS data source")
	var config dataSourceKpiBaseSearchState

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	title := config.Title.ValueString()
	base := kpiBaseSearchBase(d.client, config.ID.ValueString(), title)
	b, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read KPI BS object", err.Error())
		return
	}
	json, err := b.RawJson.ToInterfaceMap()
	if err != nil {
		resp.Diagnostics.AddError("Unable to read KPI JSON object", err.Error())
		return
	}
	if b == nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("title"),
			"KPI BS not found",
			fmt.Sprintf("KPI BS %q not found", title))
		return
	}

	state := &dataSourceKpiBaseSearchState{}

	resp.Diagnostics.Append(marshalBasicTypesByTag("json", json, state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	tflog.Debug(ctx, "Finished reading KPI BS data source", map[string]any{"success": true})
}
