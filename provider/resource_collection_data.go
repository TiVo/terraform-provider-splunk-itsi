package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &resourceCollectionData{}
)

// resource data object
type resourceCollectionData struct {
	client models.ClientConfig
}

func NewResourceCollectionData() resource.Resource {
	return &resourceCollectionData{}
}

// resource terraform models

type collectionModel struct {
	Name  string `tfsdk:"name"`
	App   string `tfsdk:"app"`
	Owner string `tfsdk:"owner"`
}

func (c *collectionModel) Key() string {
	return fmt.Sprintf("%s/%s/%s", c.Owner, c.App, c.Name)
}

type collectionEntryModel struct {
	ID   types.String `tfsdk:"id"`
	Data string       `tfsdk:"data"`
}

func (e *collectionEntryModel) DataHash() string {
	return util.Sha256([]byte(e.Data))
}

func (e *collectionEntryModel) Pack(data map[string]interface{}) (diags diag.Diagnostics) {
	b, err := json.Marshal(data)
	if err != nil {
		diags.AddError("Failed to marshal collection entry data", err.Error())
		return
	}
	e.Data = string(b)
	return
}

func (e *collectionEntryModel) Unpack() (data map[string]interface{}, diags diag.Diagnostics) {
	data = make(map[string]interface{})
	rowMap := make(map[string]interface{})
	err := json.Unmarshal([]byte(e.Data), &rowMap)
	if err != nil {
		diags.AddError("Wrong collection entry data",
			fmt.Sprintf("Unable to unmarshal collection entry data: %s;\n%s", e.Data, err.Error()))
		return
	}
	for k, v := range rowMap {
		if slice, ok := v.([]interface{}); ok {
			if len(slice) > 1 {
				data[k] = slice
			} else {
				data[k] = slice[0]
			}

		} else {
			data[k] = v
		}
	}

	return
}

type collectionDataModel struct {
	ID         types.String           `tfsdk:"id"`
	Collection collectionModel        `tfsdk:"collection"`
	Scope      string                 `tfsdk:"scope"`
	Generation types.Int64            `tfsdk:"generation"`
	Entries    []collectionEntryModel `tfsdk:"entry"`
}

func (d *collectionDataModel) Normalize() (diags diag.Diagnostics) {
	entries := make([]collectionEntryModel, len(d.Entries))

	for i, entry := range d.Entries {
		data, diags_ := entry.Unpack()
		if diags.Append(diags_...); diags.HasError() {
			return
		}

		row := make(map[string]interface{})
		for k, v := range data {
			if singleValue, ok := v.(string); ok {
				row[k] = []string{singleValue}
			} else if multiValue, ok := v.([]interface{}); ok {
				row[k] = multiValue
			} else {
				diags.AddError(fmt.Sprintf("Unable to read %s collection data", d.Collection.Key()), fmt.Sprintf("invalid collection value %#v", v))
				return
			}
		}

		entries[i].Pack(data)
		entries[i].ID = entry.ID
	}

	d.Entries = entries
	return
}

// collectionAPI client
type collectionAPI struct {
	collectionModel
	client models.ClientConfig
}

func NewCollectionAPI(m collectionModel, c models.ClientConfig) *collectionAPI {
	return &collectionAPI{m, c}
}

func (api *collectionAPI) Model(objectType string) *models.CollectionApi {
	return models.NewCollection(api.client, api.Name, api.App, api.Owner, api.Name, objectType)
}

func (api *collectionAPI) CollectionExists(ctx context.Context, require bool) (exists bool, diags diag.Diagnostics) {
	collection := api.Model("collection_config_no_body")
	collection.SetCodeHandle(http.StatusNotFound, util.Ignore)
	collection_, err := collection.Read(ctx)
	if err != nil {
		diags.AddError(fmt.Sprintf("Unable to read %s collection config", api.Key()), err.Error())
		return
	}
	if exists = collection_ != nil; require && !exists {
		diags.AddError("collection not found",
			fmt.Sprintf("Collection %s does not exist", api.Key()))
	}
	return
}

// collectionDataAPI client

type collectionDataAPI struct {
	*collectionAPI
	collectionDataModel
}

func NewCollectionDataAPI(m collectionDataModel, c models.ClientConfig) *collectionDataAPI {
	return &collectionDataAPI{NewCollectionAPI(m.Collection, c), m}
}

func (api *collectionDataAPI) Model(includeData bool) (model *models.CollectionApi, diags diag.Diagnostics) {
	data := map[string]interface{}{
		"collection_name": api.Collection.Name,
		"scope":           api.Scope,
		"generation":      api.Generation.ValueInt64(),
		"instance":        api.ID.ValueString(),
	}
	if includeData {
		model = api.collectionAPI.Model("collection_batchsave")
		entries := make([]map[string]interface{}, len(api.Entries))
		for i, entry := range api.Entries {
			rowMap, diags_ := entry.Unpack()
			diags.Append(diags_...)
			rowMap["_instance"] = api.ID.ValueString()
			rowMap["_gen"] = api.Generation.ValueInt64()
			rowMap["_scope"] = api.Scope
			rowMap["_key"] = entry.ID.ValueString()
			entries[i] = rowMap
		}
		data["data"] = entries
		var err error
		model.Body, err = json.Marshal(entries)
		if err != nil {
			diags.AddError(fmt.Sprintf("Unable to marshal %s collection data", api.Key()), err.Error())
			return nil, diags
		}
	} else {
		model = api.collectionAPI.Model("collection_data")
	}
	model.Data = data
	return
}

func (api *collectionDataAPI) Save(ctx context.Context) (diags diag.Diagnostics) {
	model, diags_ := api.Model(true)
	if diags.Append(diags_...); diags.HasError() {
		return
	}
	_, err := model.Create(ctx)
	if err != nil {
		diags.AddError(fmt.Sprintf("Unable to save %s collection data", api.Key()), err.Error())
	}
	return
}

func (api *collectionDataAPI) deleteOldRows(ctx context.Context) (diags diag.Diagnostics) {
	model, diags_ := api.Model(false)
	if diags.Append(diags_...); diags.HasError() {
		return
	}
	q := fmt.Sprintf(`{"$or":[{"_instance":null},{"_instance":{"$ne": "%s"}},{"_gen":null},{"_gen":{"$ne": %d}}]}`, api.ID.ValueString(), api.Generation.ValueInt64())
	q = fmt.Sprintf(`{"$and":[{"_scope":"%s"},%s]}`, api.Scope, q)
	model.Params = "query=" + url.QueryEscape(q)

	_, err := model.Delete(ctx)
	if err != nil {
		diags.AddError(fmt.Sprintf("Unable to delete %s collection data", api.Key()), err.Error())
	}
	return
}

func (api *collectionDataAPI) Read(ctx context.Context) (data []collectionEntryModel, diags diag.Diagnostics) {
	model, diags_ := api.Model(false)
	if diags.Append(diags_...); diags.HasError() {
		return
	}
	model.Params = "query=" + url.QueryEscape(fmt.Sprintf(`{"_scope":"%s"}`, api.Scope))
	var err error
	if model, err = model.Read(ctx); err != nil {
		diags.AddError(fmt.Sprintf("Unable to read %s collection data", api.Key()), err.Error())
		return
	}

	var obj interface{}
	if obj, err = model.Unmarshal(model.Body); err != nil {
		diags.AddError(fmt.Sprintf("Unable to unmarshal %s collection data", api.Key()), err.Error())
		return
	}

	arr, ok := obj.([]interface{})
	if !ok {
		diags.AddError(fmt.Sprintf("Unable to read %s collection data", api.Key()), "expected array body return type")
		return
	}

	data = make([]collectionEntryModel, len(arr))

	for i, item := range arr {
		var entry collectionEntryModel

		item_, ok := item.(map[string]interface{})
		if !ok {
			diags.AddError(fmt.Sprintf("Unable to read %s collection data", api.Key()), "expected map in array body return type")
		}
		row := map[string]interface{}{}

		for k, v := range item_ {
			if k == "_key" {
				entry.ID = types.StringValue(v.(string))
			} else if k[0] != '_' {
				if singleValue, ok := v.(string); ok {
					row[k] = []string{singleValue}
				} else if multiValue, ok := v.([]interface{}); ok {
					row[k] = multiValue
				} else {
					diags.AddError(fmt.Sprintf("Unable to read %s collection data", api.Key()), fmt.Sprintf("invalid collection value %#v", v))
					return
				}
			}
		}
		if diags.Append(entry.Pack(row)...); diags.HasError() {
			return
		}
		data[i] = entry
	}

	return
}

func (api *collectionDataAPI) Delete(ctx context.Context) (diags diag.Diagnostics) {
	model, diags_ := api.Model(false)
	if diags.Append(diags_...); diags.HasError() {
		return
	}
	model.Params = "query=" + url.QueryEscape(fmt.Sprintf(`{"_scope":"%s"}`, api.Scope))
	if _, err := model.Delete(ctx); err != nil {
		diags.AddError(fmt.Sprintf("Unable to delete %s collection data", api.Key()), err.Error())
	}
	return
}

// resource methods

func (r *resourceCollectionData) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(models.ClientConfig)
	if !ok {
		tflog.Error(ctx, "Unable to prepare client")
		return
	}
	r.client = client
}

func (r *resourceCollectionData) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_collection_data"
}

func (r *resourceCollectionData) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Collection data resource",
		Blocks: map[string]schema.Block{
			"collection": schema.SingleNestedBlock{
				Description: "Block identifying the collection",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Description: "Name of the collection",
						Required:    true,
					},
					"app": schema.StringAttribute{
						Description: "App of the collection",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("itsi"),
					},
					"owner": schema.StringAttribute{
						Description: "Owner of the collection",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("nobody"),
					},
				},
			},
			"entry": schema.SetNestedBlock{
				Description: "Block representing an entry in the collection",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "_key for the collection entry",
							Computed:    true,
							Optional:    true,
						},
						"data": schema.StringAttribute{
							Description: "JSON encoded data of the entry",
							Required:    true,
						},
					},
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Computed instance ID of the resource, used w/ 'generation' to prevent row duplication in a given scope",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"scope": schema.StringAttribute{
				Description: "Scope of the collection data",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("default"),
			},
			"generation": schema.Int64Attribute{
				Description: "Computed latest generation of changes",
				Computed:    true,
			},
		},
	}

}

func (r *resourceCollectionData) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return // destroy plan, nothing to do
	}
	if req.State.Raw.IsNull() {
		return // create plan, nothing to do
	}
	tflog.Trace(ctx, "Preparing to modify plan for a collecton data resource")

	var state collectionDataModel
	var plan collectionDataModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	tflog.Trace(ctx, "collection_data ModifyPlan - Parsed req config", map[string]interface{}{"plan": plan, "state": state})

	idByDataHash := make(map[string]string)
	for i := range state.Entries {
		idByDataHash[state.Entries[i].DataHash()] = state.Entries[i].ID.ValueString()
	}
	tflog.Trace(ctx, "collection_data ModifyPlan - idByDataHash", map[string]interface{}{"idByDataHash": idByDataHash})

	for i := range plan.Entries {
		if plan.Entries[i].ID.IsUnknown() {
			if id, ok := idByDataHash[plan.Entries[i].DataHash()]; ok {
				plan.Entries[i].ID = types.StringValue(id)
				tflog.Trace(ctx, "collection_data ModifyPlan - Entry found", map[string]interface{}{"id": id})
			} else {
				tflog.Trace(ctx, "collection_data ModifyPlan - Entry not found", map[string]interface{}{"data": plan.Entries[i].Data})
			}
		}
	}

	diags := resp.Plan.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Finished modifying plan for collecton data resource")

}

func (r *resourceCollectionData) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Preparing to create collecton data resource")
	var state collectionDataModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &state)...)
	tflog.Trace(ctx, "collection_data Create - Parsed req config", map[string]interface{}{"state": state})

	state.ID = types.StringValue(uuid.New().String())
	state.Generation = types.Int64Value(0)

	for i := range state.Entries {
		if state.Entries[i].ID.IsUnknown() {
			state.Entries[i].ID = types.StringValue(uuid.New().String())
		}
	}

	api := NewCollectionDataAPI(state, r.client)
	exists, diags := api.CollectionExists(ctx, true)
	resp.Diagnostics.Append(diags...)
	if !exists {
		return
	}

	diags = api.Save(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Finished creating collecton data resource")
}

func (r *resourceCollectionData) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Preparing to read collecton data resource")
	var state collectionDataModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	api := NewCollectionDataAPI(state, r.client)
	exists, diags := api.CollectionExists(ctx, true)
	resp.Diagnostics.Append(diags...)
	if !exists {
		return
	}

	entries, diags := api.Read(ctx)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "collection_data READ - Read successfull", map[string]interface{}{"entries": entries})

	state.Entries = entries
	diags = state.Normalize()
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)

	tflog.Trace(ctx, "Finished reading collecton data resource")
}

func (r *resourceCollectionData) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Preparing to update collecton data resource")
	var state collectionDataModel
	var plan collectionDataModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	tflog.Trace(ctx, "collection_data Update - Parsed req config", map[string]interface{}{"plan": plan, "state": state})

	plan.Generation = types.Int64Value(state.Generation.ValueInt64() + 1)

	api := NewCollectionDataAPI(plan, r.client)
	exists, diags := api.CollectionExists(ctx, true)
	resp.Diagnostics.Append(diags...)
	if !exists {
		return
	}

	diags = api.Save(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = api.deleteOldRows(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Finished updating collecton data resource")
}

func (r *resourceCollectionData) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Preparing to delete collecton data resource")
	var state collectionDataModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	tflog.Trace(ctx, "collection_data Delete - Parsed req config", map[string]interface{}{"state": state})

	api := NewCollectionDataAPI(state, r.client)
	exists, diags := api.CollectionExists(ctx, false)
	resp.Diagnostics.Append(diags...)
	if !exists || resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(api.Delete(ctx)...)
	tflog.Trace(ctx, "Finished deleting collecton data resource")
}
