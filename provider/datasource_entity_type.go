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
}

func entityTypeBase(clientConfig models.ClientConfig, key string, title string) *models.Base {
	base := models.NewBase(clientConfig, key, title, "entity_type")
	return base
}

func NewDataSourceEntityType() datasource.DataSource {
	return &dataSourceEntityType{}
}

func (d *dataSourceEntityType) Configure(ctx context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
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

func (d *dataSourceEntityType) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_entity_type"
}

func (d *dataSourceEntityType) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get the ID of an available entity type.",
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
		ID:    types.StringValue(b.RESTKey),
		Title: types.StringValue(title),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading entity type data source", map[string]any{"success": true})
}
