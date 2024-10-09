package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/util"
)

const (
	itsiResourceTypeEntity = "entity"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &resourceEntity{}
	_ tfmodel           = &entityModel{}
)

// =================== [ Entity ] ===================

type entityModel struct {
	ID types.String `tfsdk:"id"`

	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`

	Aliases types.Map `tfsdk:"aliases"`
	Info    types.Map `tfsdk:"info"`

	EntityTypeIDs types.Set `tfsdk:"entity_type_ids"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (m entityModel) objectype() string {
	return itsiResourceTypeEntity
}

func (m entityModel) title() string {
	return m.Title.ValueString()
}

func entityBase(clientConfig models.ClientConfig, key string, title string) *models.ItsiObj {
	base := models.NewItsiObj(clientConfig, key, title, itsiResourceTypeEntity)
	return base
}

type resourceEntity struct {
	client models.ClientConfig
}

func NewResourceEntity() resource.Resource {
	return &resourceEntity{}
}

func (r *resourceEntity) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureResourceClient(ctx, resourceNameEntity, req, &r.client, resp)
}

func (r *resourceEntity) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	configureResourceMetadata(req, resp, resourceNameEntity)
}

func (r *resourceEntity) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Entity object within ITSI.",
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
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

// =================== [ Entity API / Builder] ===================

type entityBuildWorkflow struct{}

var _ apibuildWorkflow[entityModel] = &entityBuildWorkflow{}

//lint:ignore U1000 used by apibuilder
func (w *entityBuildWorkflow) buildSteps() []apibuildWorkflowStepFunc[entityModel] {
	return []apibuildWorkflowStepFunc[entityModel]{
		w.basics,
		w.entityTypes,
		w.fields,
	}
}

func (w *entityBuildWorkflow) basics(ctx context.Context, obj entityModel) (map[string]any, diag.Diagnostics) {
	return map[string]any{
		"object_type": itsiResourceTypeEntity,
		"sec_grp":     itsiDefaultSecurityGroup,
		"title":       obj.Title.ValueString(),
		"description": obj.Description.ValueString(),
	}, nil
}

func (w *entityBuildWorkflow) entityTypes(ctx context.Context, obj entityModel) (res map[string]any, diags diag.Diagnostics) {
	var entityTypeIDs []string
	diags.Append(obj.EntityTypeIDs.ElementsAs(ctx, &entityTypeIDs, false)...)
	res = map[string]any{"entity_type_ids": entityTypeIDs}
	return
}

func (w *entityBuildWorkflow) fields(ctx context.Context, obj entityModel) (res map[string]any, diags diag.Diagnostics) {
	idFields, infoFields := util.NewSet[string](), util.NewSet[string]()
	idValues, infoValues := util.NewSetFromSlice([]string{obj.Title.ValueString()}), util.NewSet[string]()
	var aliases, info map[string]string
	diags.Append(obj.Aliases.ElementsAs(ctx, &aliases, false)...)
	diags.Append(obj.Info.ElementsAs(ctx, &info, false)...)

	res = map[string]any{}
	for k, v := range aliases {
		res[k] = strings.Split(v, ",")
		idFields.Add(k)
		for _, value := range res[k].([]string) {
			idValues.Add(value)
		}
	}

	for k, v := range info {
		res[k] = strings.Split(v, ",")
		infoFields.Add(k)
		for _, value := range res[k].([]string) {
			infoValues.Add(value)
		}
	}

	res["identifier"] = map[string][]string{"fields": idFields.ToSlice(), "values": idValues.ToSlice()}
	res["informational"] = map[string][]string{"fields": infoFields.ToSlice(), "values": infoValues.ToSlice()}

	return
}

// =================== [ Entity API / Parser ] ===================

type entityParseWorkflow struct{}

var _ apiparseWorkflow[entityModel] = &entityParseWorkflow{}

//lint:ignore U1000 used by apiparser
func (w *entityParseWorkflow) parseSteps() []apiparseWorkflowStepFunc[entityModel] {
	return []apiparseWorkflowStepFunc[entityModel]{
		w.basics,
		w.entityTypes,
		w.fields,
	}
}

func (w *entityParseWorkflow) basics(ctx context.Context, fields map[string]any, res *entityModel) (diags diag.Diagnostics) {
	stringMap, err := unpackMap[string](mapSubset(fields, []string{"title", "description"}))
	if err != nil {
		diags.AddError("Unable to populate entity type model", err.Error())
		return
	}
	res.Title = types.StringValue(stringMap["title"])
	res.Description = types.StringValue(stringMap["description"])
	return
}

func (w *entityParseWorkflow) entityTypes(ctx context.Context, fields map[string]any, res *entityModel) (diags diag.Diagnostics) {
	if v, ok := fields["entity_type_ids"]; ok && v != nil {
		entityTypeIds, err := UnpackSlice[string](v.([]interface{}))
		if err != nil {
			diags.AddError("Unable to populate entity model", err.Error())
			return
		}
		res.EntityTypeIDs, diags = types.SetValueFrom(ctx, types.StringType, entityTypeIds)
	} else {
		emptySet := []string{}
		res.EntityTypeIDs, diags = types.SetValueFrom(ctx, types.StringType, emptySet)
	}
	return
}

func (w *entityParseWorkflow) fields(ctx context.Context, fields map[string]any, res *entityModel) (diags diag.Diagnostics) {
	var d diag.Diagnostics
	fieldsMap, err := unpackMap[map[string]interface{}](mapSubset[string](fields, []string{"identifier", "informational"}))
	if err != nil {
		diags.AddError("Unable to populate entity model", err.Error())
		return
	}

	for tfField, itsiField := range map[*types.Map]string{&res.Aliases: "identifier", &res.Info: "informational"} {
		tfMap := map[string]string{}

		itsiObject := fieldsMap[itsiField]
		for _, k := range itsiObject["fields"].([]interface{}) {
			itsiValues, ok := fields[k.(string)].([]interface{})
			if !ok {
				diags.AddError("Unable to populate entity model", fmt.Sprintf("entity resource (%v): type assertion failed for '%v/fields' field", res.ID.ValueString(), itsiField))
				return
			}
			if len(itsiValues) == 0 {
				diags.AddError("Unable to populate entity model", fmt.Sprintf("entity resource (%v): missing value for '%v/fields/%v' field", res.ID.ValueString(), itsiField, k.(string)))
				return
			}
			values, err := UnpackSlice[string](itsiValues)
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

	return
}

// =================== [ Entity Resource CRUD ] ===================

func (r *resourceEntity) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state entityModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	timeouts := state.Timeouts
	readTimeout, diags := timeouts.Read(ctx, tftimeout.Read)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	base := entityBase(r.client, state.ID.ValueString(), state.Title.ValueString())
	b, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read entity", err.Error())
		return
	}
	if b == nil || b.RawJson == nil {
		resp.State.Raw = tftypes.Value{}
		return
	}

	state, diags = newAPIParser(b, new(entityParseWorkflow)).parse(ctx, b)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	state.Timeouts = timeouts
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *resourceEntity) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan entityModel
	if resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...); resp.Diagnostics.HasError() {
		return
	}

	base, diags := newAPIBuilder(r.client, new(entityBuildWorkflow)).build(ctx, plan)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, tftimeout.Create)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

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
	if resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...); resp.Diagnostics.HasError() {
		return
	}

	base, diags := newAPIBuilder(r.client, new(entityBuildWorkflow)).build(ctx, plan)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, tftimeout.Update)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

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

	deleteTimeout, diags := state.Timeouts.Delete(ctx, tftimeout.Delete)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	base := entityBase(r.client, state.ID.ValueString(), state.Title.ValueString())
	b, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete entity", err.Error())
		return
	}
	if b == nil {
		return
	}
	resp.Diagnostics.Append(b.Delete(ctx)...)
}

func (r *resourceEntity) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx, cancel := context.WithTimeout(ctx, tftimeout.Read)
	defer cancel()

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

	state, diags := newAPIParser(b, new(entityParseWorkflow)).parse(ctx, b)
	if resp.Diagnostics.Append(diags...); diags.HasError() {
		return
	}

	var timeouts timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &timeouts)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Timeouts = timeouts

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
