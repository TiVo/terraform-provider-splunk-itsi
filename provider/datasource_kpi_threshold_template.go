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

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &dataSourceKpiThresholdTemplate{}
	_ datasource.DataSourceWithConfigure = &dataSourceKpiThresholdTemplate{}
)

func NewDataSourceKpiThresholdTemplate() datasource.DataSource {
	return &dataSourceKpiThresholdTemplate{}
}

type dataSourceKpiThresholdTemplate struct {
	client models.ClientConfig
}

type dataSourceKpiThresholdTemplateModel struct {
	Title types.String `tfsdk:"title"`
	ID    types.String `tfsdk:"id"`
}

func (d *dataSourceKpiThresholdTemplate) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	configureDataSourceClient(ctx, datasourceNameKPIThresholdTemplate, req, &d.client, resp)
}

// Metadata returns the data source type name.
func (d *dataSourceKpiThresholdTemplate) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	configureDataSourceMetadata(req, resp, datasourceNameKPIThresholdTemplate)
}

// Schema defines the schema for the data source.
func (d *dataSourceKpiThresholdTemplate) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get the ID of an available KPI threshold template.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this KPI Threshold template",
				Computed:    true,
			},
			"title": schema.StringAttribute{
				Description: "The name of the KPI Threshold template",
				Required:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *dataSourceKpiThresholdTemplate) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read KPI Threshold template data source")
	var config dataSourceKpiThresholdTemplateModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	title := config.Title.ValueString()
	base := kpiThresholdTemplateBase(d.client, config.ID.ValueString(), title)
	b, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read KPI Threshold template object", err.Error())
		return
	}
	if b == nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("title"),
			"KPI Threshold template not found",
			fmt.Sprintf("KPI Threshold template %q not found", title))
		return
	}

	state := dataSourceEntityTypeModel{
		ID:    types.StringValue(b.RESTKey),
		Title: types.StringValue(title),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading  KPI Threshold template data source", map[string]any{"success": true})
}
