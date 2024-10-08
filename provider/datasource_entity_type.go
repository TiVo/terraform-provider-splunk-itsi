package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

var (
	_ datasource.DataSource              = &dataSourceEntityType{}
	_ datasource.DataSourceWithConfigure = &dataSourceEntityType{}
)

type dataSourceEntityType struct {
	client models.ClientConfig
}

type dataSourceEntityTypeModel struct {
	ID    types.String `tfsdk:"id"`
	Title types.String `tfsdk:"title"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func entityTypeBase(clientConfig models.ClientConfig, key string, title string) *models.ItsiObj {
	base := models.NewItsiObj(clientConfig, key, title, "entity_type")
	return base
}

func NewDataSourceEntityType() datasource.DataSource {
	return &dataSourceEntityType{}
}

func (d *dataSourceEntityType) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	configureDataSourceClient(ctx, datasourceNameEntityType, req, &d.client, resp)
}

func (d *dataSourceEntityType) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	configureDataSourceMetadata(req, resp, datasourceNameEntityType)
}

func (d *dataSourceEntityType) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get the ID of an available entity type.",
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx),
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this entity type",
				Computed:    true,
			},
			"title": schema.StringAttribute{
				Description: "The name of the entity type",
				Required:    true,
			},
		},
	}
}

func (d *dataSourceEntityType) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read entity type data source")
	var config dataSourceEntityTypeModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	timeouts := config.Timeouts
	readTimeout, diags := timeouts.Read(ctx, tftimeout.Read)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	title := config.Title.ValueString()
	base := entityTypeBase(d.client, config.ID.ValueString(), title)
	b, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Entity Type object", err.Error())
		return
	}
	if b == nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("title"),
			"Entity type not found",
			fmt.Sprintf("Entity type %q not found", title))
		return
	}

	state := dataSourceEntityTypeModel{
		ID:       types.StringValue(b.RESTKey),
		Title:    types.StringValue(title),
		Timeouts: timeouts,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading entity type data source", map[string]any{"success": true})
}
