package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

var (
	_ datasource.DataSource              = &dataSourceCollection{}
	_ datasource.DataSourceWithConfigure = &dataSourceCollection{}
)

type dataSourceCollectionModel struct {
	//collectionConfigModel    #TODO: <--- use the embedded struct, once this is supported by the terraform-plugin-framework ( https://github.com/hashicorp/terraform-plugin-framework/issues/242 )

	Name          types.String `tfsdk:"name"`
	App           types.String `tfsdk:"app"`
	Owner         types.String `tfsdk:"owner"`
	FieldTypes    types.Map    `tfsdk:"field_types"`
	Accelerations types.List   `tfsdk:"accelerations"`

	Fields types.Set `tfsdk:"fields"`
}

func (d *dataSourceCollectionModel) collectionConfigModel() *collectionConfigModel {
	return &collectionConfigModel{
		Name:          d.Name,
		App:           d.App,
		Owner:         d.Owner,
		FieldTypes:    d.FieldTypes,
		Accelerations: d.Accelerations,
	}
}

type dataSourceCollection struct {
	client models.ClientConfig
}

func NewDataSourceCollection() datasource.DataSource {
	return &dataSourceCollection{}
}

func (d *dataSourceCollection) Configure(ctx context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(models.ClientConfig)
	if !ok {
		return
	}
	d.client = client
}

func (d *dataSourceCollection) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_splunk_collection"
}

func (d *dataSourceCollection) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the details of a collection.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the collection",
				Required:            true,
			},
			"app": schema.StringAttribute{
				MarkdownDescription: "App of the collection. Defaults to 'itsi'.",
				Optional:            true,
				Computed:            true,
				Validators:          validateStringIdentifier(),
			},
			"owner": schema.StringAttribute{
				MarkdownDescription: "Owner of the collection. Defaults to 'nobody'.",
				Optional:            true,
				Computed:            true,
				Validators:          validateStringIdentifier(),
			},
			"field_types": schema.MapAttribute{
				MarkdownDescription: "Field name -> field type mapping for the collection's columns. ",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"accelerations": schema.ListAttribute{
				Validators: []validator.List{
					listvalidator.SizeAtMost(1000),
				},
				ElementType: types.StringType,
				Computed:    true,
			},
			"fields": schema.SetAttribute{
				MarkdownDescription: "Collection fields",
				ElementType:         types.StringType,
				Computed:            true,
			},
		},
	}
}

func (d *dataSourceCollection) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Trace(ctx, "Preparing to read collecton datasource")
	var config dataSourceCollectionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	api := NewCollectionConfigAPI(config.collectionConfigModel().Normalize(), d.client)

	if resp.Diagnostics.Append(api.Read(ctx)...); resp.Diagnostics.HasError() {
		return
	}

	obj, diags := api.Query(ctx, "", []string{})
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	arr, ok := obj.([]interface{})
	if !ok {
		diags.AddError(fmt.Sprintf("Unable to read %s collection data", api.Key()), "expected array body return type")
		return
	}

	fieldsMap := map[string]struct{}{}

	for _, item := range arr {

		item_, ok := item.(map[string]interface{})
		if !ok {
			diags.AddError(fmt.Sprintf("Unable to read %s collection data", api.Key()), "expected map in array body return type")
		}

		for k := range item_ {
			if k[0] != '_' {
				fieldsMap[k] = struct{}{}
			}
		}
	}

	fields := make([]string, 0, len(fieldsMap))
	for f := range fieldsMap {
		fields = append(fields, f)
	}

	api.config = api.config.Normalize()
	config.App = api.config.App
	config.Owner = api.config.Owner
	config.FieldTypes = api.config.FieldTypes
	config.Accelerations = api.config.Accelerations
	config.Fields, diags = types.SetValueFrom(ctx, config.Fields.ElementType(ctx), fields)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, config)
}
