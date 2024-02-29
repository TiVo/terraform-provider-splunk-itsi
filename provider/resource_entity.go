package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &resourceEntity{}
)

type entityModel struct {
	ID types.String `tfsdk:"id"`

	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`

	Aliases types.Map `tfsdk:"aliases"`
	Info    types.Map `tfsdk:"info"`

	EntityTypeIDs types.Set `tfsdk:"entity_type_ids"`
}

func entityBase(clientConfig models.ClientConfig, key string, title string) *models.Base {
	base := models.NewBase(clientConfig, key, title, "entity")
	return base
}

func entityModelFromBase(ctx context.Context, b *models.Base) (m entityModel, diags diag.Diagnostics) {
	var d diag.Diagnostics
	if b == nil || b.RawJson == nil {
		diags.AddError("Unable to populate entity model", "base object is nil or empty.")
		return
	}

	interfaceMap, err := b.RawJson.ToInterfaceMap()
	if err != nil {
		diags.AddError("Unable to populate entity model", err.Error())
		return
	}

	stringMap, err := unpackMap[string](mapSubset[string](interfaceMap, []string{"title", "description"}))
	if err != nil {
		diags.AddError("Unable to populate entity model", err.Error())
		return
	}

	m.Title = types.StringValue(stringMap["title"])
	m.Description = types.StringValue(stringMap["description"])

	if v, ok := interfaceMap["entity_type_ids"]; ok && v != nil {
		entityTypeIds, err := unpackSlice[string](v.([]interface{}))
		if err != nil {
			diags.AddError("Unable to populate entity model", err.Error())
			return
		}
		m.EntityTypeIDs, d = types.SetValueFrom(ctx, types.StringType, entityTypeIds)
		if diags.Append(d...); diags.HasError() {
			return
		}
	}

	fieldsMap, err := unpackMap[map[string]interface{}](mapSubset[string](interfaceMap, []string{"identifier", "informational"}))
	if err != nil {
		diags.AddError("Unable to populate entity model", err.Error())
		return
	}

	for tfField, itsiField := range map[*types.Map]string{&m.Aliases: "identifier", &m.Info: "informational"} {
		tfMap := map[string]string{}

		itsiObject := fieldsMap[itsiField]
		for _, k := range itsiObject["fields"].([]interface{}) {
			itsiValues, ok := interfaceMap[k.(string)].([]interface{})
			if !ok {
				diags.AddError("Unable to populate entity model", fmt.Sprintf("entity resource (%v): type assertion failed for '%v/fields' field", b.RESTKey, itsiField))
				return
			}
			if len(itsiValues) == 0 {
				diags.AddError("Unable to populate entity model", fmt.Sprintf("entity resource (%v): missing value for '%v/fields/%v' field", b.RESTKey, itsiField, k.(string)))
				return
			}
			values, err := unpackSlice[string](itsiValues)
			if err != nil {
				diags.AddError("Unable to populate entity model", err.Error())
				return
			}
			tfMap[k.(string)] = strings.Join(values, ",")
		}

		*tfField, d = types.MapValueFrom(ctx, types.StringType, tfMap)
		if diags.Append(d...); diags.HasError() {
			return
		}
	}

	m.ID = types.StringValue(b.RESTKey)

	return
}

func entity(ctx context.Context, clientConfig models.ClientConfig, m entityModel) (config *models.Base, diags diag.Diagnostics) {
	title := m.Title.ValueString()

	body := map[string]interface{}{}
	body["object_type"] = "entity"
	body["sec_grp"] = "default_itsi_security_group"

	body["title"] = title
	body["description"] = m.Description.ValueString()

	idFields, infoFields := util.NewSet[string](), util.NewSet[string]()
	idValues, infoValues := util.NewSetFromSlice[string]([]string{title}), util.NewSet[string]()

	var aliases, info map[string]string
	var entityTypeIDs []string

	if diags.Append(m.Aliases.ElementsAs(ctx, &aliases, false)...); diags.HasError() {
		return
	}
	if diags.Append(m.Info.ElementsAs(ctx, &info, false)...); diags.HasError() {
		return
	}
	if diags.Append(m.EntityTypeIDs.ElementsAs(ctx, &entityTypeIDs, false)...); diags.HasError() {
		return
	}

	for k, v := range aliases {
		body[k] = strings.Split(v, ",")
		idFields.Add(k)
		for _, value := range body[k].([]string) {
			idValues.Add(value)
		}
	}

	for k, v := range info {
		body[k] = strings.Split(v, ",")
		infoFields.Add(k)
		for _, value := range body[k].([]string) {
			infoValues.Add(value)
		}
	}

	body["identifier"] = map[string][]string{"fields": idFields.ToSlice(), "values": idValues.ToSlice()}
	body["informational"] = map[string][]string{"fields": infoFields.ToSlice(), "values": infoValues.ToSlice()}
	body["entity_type_ids"] = entityTypeIDs

	config = entityBase(clientConfig, m.ID.ValueString(), title)
	if err := config.PopulateRawJSON(ctx, body); err != nil {
		diags.AddError("Unable to populate base object", err.Error())
	}
	return
}

type resourceEntity struct {
	client models.ClientConfig
}

func NewResourceEntity() resource.Resource {
	return &resourceEntity{}
}

func (r *resourceEntity) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *resourceEntity) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_entity"
}

func (r *resourceEntity) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Entity object within ITSI.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the entity.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "Name of the entity. Can be any unique value.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "User defined description of the entity.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"aliases": schema.MapAttribute{
				MarkdownDescription: "Map of Field/Value pairs that identify the entity.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			},
			"info": schema.MapAttribute{
				MarkdownDescription: "Map of Field/Value pairs that provide information/description for the entity.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			},
			"entity_type_ids": schema.SetAttribute{
				MarkdownDescription: "A set of _key values for each entity type associated with the entity.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
		},
	}
}

func (r *resourceEntity) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state entityModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	base := entityBase(r.client, state.ID.ValueString(), state.Title.ValueString())
	b, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read entity", err.Error())
		return
	}
	if b == nil || b.RawJson == nil {
		resp.Diagnostics.Append(req.State.Set(ctx, &entityModel{})...)
		return
	}

	state, diags := entityModelFromBase(ctx, b)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.Set(ctx, &state)...)
}

func (r *resourceEntity) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan entityModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	base, diags := entity(ctx, r.client, plan)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	base, err := base.Create(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create entity", err.Error())
		return
	}

	plan.ID = types.StringValue(base.RESTKey)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

}

func (r *resourceEntity) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan entityModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	base, diags := entity(ctx, r.client, plan)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	existing, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update entity", err.Error())
		return
	}
	if existing == nil {
		resp.Diagnostics.AddError("Unable to update entity", "entity not found")
		return
	}
	if err := base.Update(ctx); err != nil {
		resp.Diagnostics.AddError("Unable to update entity", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *resourceEntity) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state entityModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	base := entityBase(r.client, state.ID.ValueString(), state.Title.ValueString())
	b, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete entity", err.Error())
		return
	}
	if b == nil {
		return
	}
	if err := b.Delete(ctx); err != nil {
		resp.Diagnostics.AddError("Unable to delete entity", err.Error())
		return
	}
}

func (r *resourceEntity) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	b := entityBase(r.client, "", req.ID)
	b, err := b.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to find entity model", err.Error())
		return
	}
	if b == nil {
		resp.Diagnostics.AddError("Entity not found", fmt.Sprintf("Entity '%s' not found", req.ID))
		return
	}

	state, diags := entityModelFromBase(ctx, b)
	if resp.Diagnostics.Append(diags...); diags.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
