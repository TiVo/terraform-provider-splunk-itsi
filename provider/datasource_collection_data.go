package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

var (
	_ datasource.DataSource              = &dataSourceCollectionData{}
	_ datasource.DataSourceWithConfigure = &dataSourceCollectionData{}
)

type dataSourceCollectionData struct {
	client models.ClientConfig
}

type dataSourceCollectionDataModel struct {
	Collection collectionIDModel `tfsdk:"collection"`
	Query      types.String      `tfsdk:"query"`
	Fields     types.Set         `tfsdk:"fields"`
	Data       types.String      `tfsdk:"data"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func NewDataSourceCollectionData() datasource.DataSource {
	return &dataSourceCollectionData{}
}

func (d *dataSourceCollectionData) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	configureDataSourceClient(ctx, datasourceNameCollectionData, req, &d.client, resp)
}

func (d *dataSourceCollectionData) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	configureDataSourceMetadata(req, resp, datasourceNameCollectionData)
}

func (d *dataSourceCollectionData) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Collection data",
		Blocks: map[string]schema.Block{
			"collection": collectionIDSchema(),
			"timeouts":   timeouts.Block(ctx),
		},
		Attributes: map[string]schema.Attribute{
			"query": schema.StringAttribute{
				MarkdownDescription: "Query to filter the data requested",
				Optional:            true,
			},
			"fields": schema.SetAttribute{
				MarkdownDescription: "List of fields to include (1) or exclude (0). A fields value cannot contain both include and exclude specifications except for exclusion of the _key field. If not specified, all fields will be returned",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"data": schema.StringAttribute{
				MarkdownDescription: "JSON encoded entries",
				Computed:            true,
			},
		},
	}
}

func (d *dataSourceCollectionData) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read entity collection_data datasource")
	var state dataSourceCollectionDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	readTimeout, diags := state.Timeouts.Read(ctx, tftimeout.Read)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	if state.Collection.App.IsNull() {
		state.Collection.App = types.StringValue(collectionDefaultApp)
	}
	if state.Collection.Owner.IsNull() {
		state.Collection.Owner = types.StringValue(collectionDefaultUser)
	}

	api := NewCollectionAPI(state.Collection, d.client)

	var fields []string
	if resp.Diagnostics.Append(state.Fields.ElementsAs(ctx, &fields, false)...); resp.Diagnostics.HasError() {
		return
	}

	obj, diags := api.Query(ctx, state.Query.ValueString(), fields, 0)
	resp.Diagnostics.Append(diags...)

	data, err := json.Marshal(obj)
	if err != nil {
		resp.Diagnostics.AddError("Unable to marshal JSON for collection data", err.Error())
	}

	state.Data = types.StringValue(string(data))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading collection_data datasource", map[string]any{"success": true})
}
