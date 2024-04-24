package provider

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

// COLLECTION MODELS

type collectionIDModel struct {
	Name  types.String `tfsdk:"name"`
	App   types.String `tfsdk:"app"`
	Owner types.String `tfsdk:"owner"`
}

func (c *collectionIDModel) Key() string {
	return fmt.Sprintf("%s/%s/%s", c.Owner.ValueString(), c.App.ValueString(), c.Name.ValueString())
}

func collectionIDModelFromString(key string) (c collectionIDModel, diags diag.Diagnostics) {
	cleanKey := strings.TrimSpace(key)
	parts := strings.Split(cleanKey, "/")

	if len(cleanKey) == 0 || len(parts) > 3 {
		errorDetails := util.Dedent(fmt.Sprintf(`
			Collection key must be in the format '[collection_owner/][collection_app/]<collection_name>'.
			E.g. '%[1]s/%[2]s/example', '%[2]s/example', 'example' are valid collection names.
			When the collection's owner and/or app are not provided, the default values of %[1]q and %[2]q are assumed respectively.
		`, collectionDefaultUser, collectionDefaultApp))

		diags.AddError(fmt.Sprintf("Invalid collection key %q", cleanKey), errorDetails)
		return
	}

	owner := collectionDefaultUser
	app := collectionDefaultApp
	name := ""

	for i, f := range []*string{&name, &app, &owner} {
		if i < len(parts) {
			*f = parts[len(parts)-i-1]
		}
	}

	c = collectionIDModel{
		Name:  types.StringValue(name),
		App:   types.StringValue(app),
		Owner: types.StringValue(owner),
	}
	return
}

func collectionIDModelAndScopeFromString(key string) (c collectionIDModel, scope string, diags diag.Diagnostics) {
	cleanKey := strings.TrimSpace(key)
	parts := strings.Split(cleanKey, ":")

	if len(cleanKey) == 0 || len(parts) > 2 {
		errorDetails := util.Dedent(fmt.Sprintf(`
			Collection scope key must be in the format '[collection_owner/][collection_app/]<collection_name>[:scope]'.
			E.g. '%[1]s/%[2]s/example:%[3]s', '%[1]s/%[2]s/example', '%[2]s/example', 'example', are valid values for a collection scope and all refer to the same collection scope.
			When scope is ommited, the %[3]q scope is assumed.
		`, collectionDefaultUser, collectionDefaultApp, collectionDefaultScope))
		diags.AddError("Invalid collection scope format", errorDetails)
		return
	}

	scope = collectionDefaultScope
	if len(parts) == 2 {
		scope = parts[1]
	}

	c, d := collectionIDModelFromString(parts[0])
	diags.Append(d...)

	return
}

type collectionConfigModel struct {
	//collectionIDModel    #TODO: <--- use the embedded struct, once this is supported by the terraform-plugin-framework ( https://github.com/hashicorp/terraform-plugin-framework/issues/242 )

	Name          types.String `tfsdk:"name"`
	App           types.String `tfsdk:"app"`
	Owner         types.String `tfsdk:"owner"`
	FieldTypes    types.Map    `tfsdk:"field_types"`
	Accelerations types.List   `tfsdk:"accelerations"`
}

func (c *collectionConfigModel) CollectionIDModel() collectionIDModel {
	return collectionIDModel{
		Name:  c.Name,
		App:   c.App,
		Owner: c.Owner,
	}
}

func (c *collectionConfigModel) Normalize() (m collectionConfigModel) {
	m = *c

	if m.App.IsNull() || m.App.ValueString() == "" {
		m.App = types.StringValue(collectionDefaultApp)
	}

	if m.Owner.IsNull() || m.Owner.ValueString() == "" {
		m.Owner = types.StringValue(collectionDefaultUser)
	}

	if m.FieldTypes.IsNull() || len(m.FieldTypes.Elements()) == 0 {
		m.FieldTypes, _ = types.MapValue(types.StringType, map[string]attr.Value{})
	}

	if m.Accelerations.IsNull() || len(m.Accelerations.Elements()) == 0 {
		m.Accelerations, _ = types.ListValue(types.StringType, []attr.Value{})
	}

	return m
}

// COLLECTION RESOURCE IMPLEMENTATION

var resourceCollectionMarkdownDescription = strings.Replace(`
Manages a KV store collection in Splunk.

~> Due to Splunk API limitations removing an item from {{.BT}}field_types{{.BT}} or {{.BT}}accelerations{{.BT}} requires collection recreation, leading to data loss.
Adding or modifying these configurations, however is supported and will not affect existing data.
The terraform provider will issue a warning at plan time if a collection is set to be replaced due to these modifications. Practitioners should exercise caution when modifying these fields.
`, "{{.BT}}", "`", -1)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &resourceCollection{}
)

type resourceCollection struct {
	client models.ClientConfig
}

func NewResouceCollection() resource.Resource {
	return &resourceCollection{}
}

func (r *resourceCollection) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureResourceClient(ctx, resourceNameCollection, req, &r.client, resp)
}

func (r *resourceCollection) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	configureResourceMetadata(req, resp, resourceNameCollection)
}

func collectionIDSchema() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		MarkdownDescription: "Block identifying the collection",
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.RequiresReplace(),
		},
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				MarkdownDescription: "Name of the collection",
				Required:            true,
				Validators:          validateStringIdentifier(),
			},
			"app": schema.StringAttribute{
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				MarkdownDescription: "App of the collection. Defaults to 'itsi'.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(collectionDefaultApp),
				Validators:          validateStringIdentifier(),
			},
			"owner": schema.StringAttribute{
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				MarkdownDescription: "Owner of the collection. Defaults to 'nobody'.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(collectionDefaultUser),
				Validators:          validateStringIdentifier(),
			},
		},
		Validators: []validator.Object{objectvalidator.IsRequired()},
	}
}

func (r *resourceCollection) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := collectionIDSchema().Attributes
	maps.Copy(attrs, map[string]schema.Attribute{
		"field_types": schema.MapAttribute{
			PlanModifiers: []planmodifier.Map{
				fieldTypesPlanModifier{},
			},
			MarkdownDescription: "Field name -> field type mapping for the collection's columns. Field types are used to determine the data type of the column in the collection. Supported field types are: `array`, `number`, `bool`, `time`, `string` and `cidr`.",
			Validators: []validator.Map{
				mapvalidator.ValueStringsAre(stringvalidator.OneOf("array", "number", "bool", "time", "string", "cidr")),
			},
			ElementType: types.StringType,
			Optional:    true,
			Computed:    true,
			Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
		},
		"accelerations": schema.ListAttribute{
			PlanModifiers: []planmodifier.List{
				accelerationsPlanModifier{},
			},
			Validators: []validator.List{
				listvalidator.SizeAtMost(1000),
			},
			ElementType: types.StringType,
			Optional:    true,
			Computed:    true,
			Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
		},
	})

	resp.Schema = schema.Schema{
		MarkdownDescription: resourceCollectionMarkdownDescription,
		Attributes:          attrs,
	}
}

func (r *resourceCollection) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Preparing to read collecton resource")
	var state collectionConfigModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	api := NewCollectionConfigAPI(state, r.client)
	if resp.Diagnostics.Append(api.Read(ctx)...); resp.Diagnostics.HasError() {
		return
	}
	state = api.config.Normalize()
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)

	tflog.Trace(ctx, "Finished reading collecton data resource")
}

func (r *resourceCollection) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Preparing to create collection resource")
	var config, plan collectionConfigModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	tflog.Trace(ctx, "collection_resource Create - Parsed req config", map[string]interface{}{"config": config, "plan": plan})

	plan = plan.Normalize()

	api := NewCollectionConfigAPI(plan, r.client)
	if resp.Diagnostics.Append(api.Create(ctx)...); resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Finished creating collecton resource")
}

func (r *resourceCollection) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Preparing to update collection resource")
	var config, plan collectionConfigModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	tflog.Trace(ctx, "collection_resource Update - Parsed req config", map[string]interface{}{"config": config, "plan": plan})

	plan = plan.Normalize()

	api := NewCollectionConfigAPI(plan, r.client)
	if resp.Diagnostics.Append(api.Update(ctx)...); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *resourceCollection) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Preparing to delete collection resource")
	var state collectionConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	tflog.Trace(ctx, "collection_resource Delete - Parsed req state", map[string]interface{}{"state": state})

	api := NewCollectionConfigAPI(state, r.client)
	resp.Diagnostics.Append(api.Delete(ctx)...)
	tflog.Trace(ctx, "Finished deleting collecton resource")
}

func (r *resourceCollection) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	collectionID, diags := collectionIDModelFromString(req.ID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := collectionConfigModel{
		Owner: collectionID.Owner,
		App:   collectionID.App,
		Name:  collectionID.Name,
	}

	api := NewCollectionConfigAPI(state, r.client)
	if resp.Diagnostics.Append(api.Read(ctx)...); resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, api.config)...)
}

// PLAN MODIFIERS

const (
	collectionReplaceWarning                       = "Collection will be replaced. Data loss may occur. Proceed with caution."
	collectionFieldTypesPlanModifierDescription    = "When set, a collection will be replaced if a field is removed from the field_types attribute."
	collectionAccelerationsPlanModifierDescription = "When set, a collection will be replaced if an acceleration is removed from the accelerations attribute."
)

// field types plan modifier

type fieldTypesPlanModifier struct{}

func (m fieldTypesPlanModifier) Description(_ context.Context) string {
	return collectionFieldTypesPlanModifierDescription
}

func (m fieldTypesPlanModifier) MarkdownDescription(_ context.Context) string {
	return collectionFieldTypesPlanModifierDescription
}

func (m fieldTypesPlanModifier) PlanModifyMap(_ context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if req.StateValue.IsNull() {
		return
	}

	if !req.ConfigValue.IsNull() && req.PlanValue.IsUnknown() {
		resp.Diagnostics.AddAttributeWarning(path.Root("field_types"),
			collectionReplaceWarning,
			"Unknown field_types will force collection replacement.")
		resp.RequiresReplace = true
		return
	}

	fieldTypesPlan := util.NewSet[string]()
	for field := range req.PlanValue.Elements() {
		fieldTypesPlan.Add(field)
	}

	for field := range req.StateValue.Elements() {
		if !fieldTypesPlan.Contains(field) {
			resp.Diagnostics.AddAttributeWarning(path.Root("field_types"),
				collectionReplaceWarning,
				fmt.Sprintf("Field type removal (%s) will force collection replacement.", field))
			resp.RequiresReplace = true
		}
	}
}

// accelerations plan mofier

type accelerationsPlanModifier struct{}

func (m accelerationsPlanModifier) Description(_ context.Context) string {
	return collectionAccelerationsPlanModifierDescription
}

func (m accelerationsPlanModifier) MarkdownDescription(_ context.Context) string {
	return collectionAccelerationsPlanModifierDescription
}

func (m accelerationsPlanModifier) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.StateValue.IsNull() {
		return
	}

	if !req.ConfigValue.IsNull() && req.PlanValue.IsUnknown() {
		resp.Diagnostics.AddAttributeWarning(path.Root("accelerations"),
			collectionReplaceWarning,
			"Unknown accelerations will force collection replacement.")
		resp.RequiresReplace = true
		return
	}

	if len(req.PlanValue.Elements()) < len(req.StateValue.Elements()) {
		resp.Diagnostics.AddAttributeWarning(path.Root("accelerations"),
			collectionReplaceWarning,
			"Acceleration removal will force collection replacement.")
		resp.RequiresReplace = true
	}
}
