package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

const (
	collectionDefaultUser          = "nobody"
	collectionDefaultApp           = "itsi"
	collectionEntryDataDescription = "Collection entry `data` must be JSON encoded map where keys are field names, " +
		"and values are strings, numbers, booleans, or arrays of those types."
	collectionEntryInvalidError = "Invalid collection entry data"
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

type collectionEntryModel struct {
	ID   types.String `tfsdk:"id"`
	Data types.String `tfsdk:"data"`
}

func (e *collectionEntryModel) DataHash() string {
	return util.Sha256([]byte(e.Data.ValueString()))
}

func (e *collectionEntryModel) Pack(data map[string]interface{}) (diags diag.Diagnostics) {
	b, err := json.Marshal(data)
	if err != nil {
		diags.AddError("Failed to marshal collection entry data", err.Error())
		return
	}
	e.Data = types.StringValue(string(b))
	return
}

func (e *collectionEntryModel) Unpack() (data map[string]interface{}, diags diag.Diagnostics) {
	data = make(map[string]interface{})
	rowMap := make(map[string]interface{})
	err := json.Unmarshal([]byte(e.Data.ValueString()), &rowMap)
	if err != nil {
		diags.AddError("Wrong collection entry data",
			fmt.Sprintf("Unable to unmarshal collection entry data: %s;\n%s", e.Data, err.Error()))
		return
	}
	for k, v := range rowMap {
		if slice, ok := v.([]interface{}); ok {
			if len(slice) == 1 {
				data[k] = slice[0]
			} else {
				data[k] = slice
			}

		} else {
			data[k] = v
		}
	}

	return
}

type collectionDataModel struct {
	ID         types.String      `tfsdk:"id"`
	Collection collectionIDModel `tfsdk:"collection"`
	Scope      types.String      `tfsdk:"scope"`
	Generation types.Int64       `tfsdk:"generation"`
	Entries    types.Set         `tfsdk:"entry"`
}

// Normalize func allows for supressing the diff when a fields value changes from
// a single value to a list of that 1 value or vice versa
func (d *collectionDataModel) Normalize(ctx context.Context) (diags diag.Diagnostics) {
	var srcEntries []collectionEntryModel
	if diags.Append(d.Entries.ElementsAs(ctx, &srcEntries, false)...); diags.HasError() {
		return
	}
	entries := make([]collectionEntryModel, len(srcEntries))

	for i, entry := range srcEntries {
		entries[i].ID = entry.ID
		if entry.Data.IsUnknown() {
			entries[i].Data = entry.Data
			continue
		}

		data, diags_ := entry.Unpack()
		if diags.Append(diags_...); diags.HasError() {
			return
		}

		if diags.Append(entries[i].Pack(data)...); diags.HasError() {
			return
		}
	}

	var diags_ diag.Diagnostics
	d.Entries, diags_ = types.SetValueFrom(ctx, d.Entries.ElementType(ctx), entries)
	diags.Append(diags_...)
	return
}

// validations

type entryDataValidator struct{}

func (v entryDataValidator) Description(ctx context.Context) string {
	return collectionEntryDataDescription
}

func (v entryDataValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v entryDataValidator) ValidateEntry(data string) (diags diag.Diagnostics) {
	var obj interface{}
	if err := json.Unmarshal([]byte(data), &obj); err != nil {
		diags.AddError(
			collectionEntryInvalidError,
			"Collection entry data is not a valid JSON object",
		)
		return
	}
	objMap, ok := obj.(map[string]interface{})
	if !ok {
		diags.AddError(
			collectionEntryInvalidError,
			"Collection entry data is not a valid JSON map",
		)
		return
	}
	for k := range objMap {
		//validate key
		switch {
		case strings.EqualFold(k, "_key"):
			diags.AddError(
				collectionEntryInvalidError,
				"Collection entry data object must not have a _key field. "+
					"Please use the entries `id` field to set _key.",
			)
		case strings.EqualFold(k, "_scope"):
			diags.AddError(
				collectionEntryInvalidError,
				"Collection entry data object must not have a _scope field. "+
					"Please use the collection_data `scope` field to configure entries scope.",
			)
		case strings.EqualFold(k, "_gen"):
			fallthrough
		case strings.EqualFold(k, "_instance"):
			fallthrough
		case strings.EqualFold(k, "_user"):
			diags.AddError(
				collectionEntryInvalidError,
				fmt.Sprintf("Collection entry data object must not have a %s field "+
					"because it is reserved for internal use. Please use a different field name.", k),
			)
		}

	}

	return
}

func (v entryDataValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// If the value is unknown or null, there is nothing to validate.
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	diags := v.ValidateEntry(req.ConfigValue.ValueString())
	for _, d := range diags {
		resp.Diagnostics.Append(diag.WithPath(req.Path, d))
	}

}

func collectionDataEntryIsValid() entryDataValidator {
	return entryDataValidator{}
}

// resource methods

func (r *resourceCollectionData) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *resourceCollectionData) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_collection_data"
}

func (r *resourceCollectionData) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Collection data resource",
		Blocks: map[string]schema.Block{
			"collection": collectionIDSchema(),
			"entry": schema.SetNestedBlock{
				MarkdownDescription: "Block representing an entry in the collection",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "_key for the collection entry",
							Computed:            true,
							Optional:            true,
						},
						"data": schema.StringAttribute{
							MarkdownDescription: collectionEntryDataDescription,
							Required:            true,
							Validators:          []validator.String{collectionDataEntryIsValid()},
						},
					},
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Computed instance ID of the resource, used w/ 'generation' to prevent row duplication in a given scope",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"scope": schema.StringAttribute{
				MarkdownDescription: "Scope of the collection data",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("default"),
			},
			"generation": schema.Int64Attribute{
				MarkdownDescription: "Computed latest generation of changes",
				Computed:            true,
			},
		},
	}

}

/*
Custom plan handling for collection data:
1.
Populates planned entries ID fields using respective IDs from the state,
if a planned entry's data hash matches an existing entry in the state.
This reduces the number of entries that are shown in the diff to only those that are actually changing.
TODO: Review/improve this solution, once the terraform-plugin-framework has a better way to handle this.
* https://github.com/hashicorp/terraform-plugin-framework/issues/717
* https://github.com/hashicorp/terraform-plugin-framework/pull/718
* https://github.com/hashicorp/terraform-plugin-framework/issues/720
2.
Normalizes the planned entries JSON data, so that the diff is more readable.
In particular, it would omit diff if a field value changes between a single value and a list of one value, if that value stays the same.
3.
Due to a side effect of the normalization process (that can cause drifts),
the resource generation value is also computed here on resource updates.
This is necessary to prevent drifts, where "generation" would be the only field that changes,(causing pointless resource updates).
*/
func (r *resourceCollectionData) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return // destroy plan, nothing to do
	}
	tflog.Trace(ctx, "Preparing to modify plan for a collecton data resource")

	var diags diag.Diagnostics
	var config, state, plan collectionDataModel
	if diags = req.State.Get(ctx, &state); !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(diags...)
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	tflog.Trace(ctx, "collection_data ModifyPlan - Parsed req config", map[string]interface{}{"config": config, "plan": plan, "state": state})

	if plan.Entries.IsUnknown() {
		resp.Diagnostics.AddWarning("unknown entries", "collection_data entries are known only after apply")
		return
	}
	if resp.Diagnostics.Append(plan.Normalize(ctx)...); resp.Diagnostics.HasError() {
		return
	}

	var configEntries, stateEntries, planEntries []collectionEntryModel
	idByDataHash := make(map[string]string)

	for model, entries := range map[*collectionDataModel]*[]collectionEntryModel{&state: &stateEntries, &config: &configEntries, &plan: &planEntries} {
		if !model.Entries.IsNull() {
			if resp.Diagnostics.Append(model.Entries.ElementsAs(ctx, entries, false)...); resp.Diagnostics.HasError() {
				return
			}
		}
	}

	for i := range stateEntries {
		idByDataHash[stateEntries[i].DataHash()] = stateEntries[i].ID.ValueString()
	}
	tflog.Trace(ctx, "collection_data ModifyPlan - idByDataHash", map[string]interface{}{"idByDataHash": idByDataHash})

	for i := range planEntries {
		if planEntries[i].ID.IsUnknown() && configEntries[i].ID.IsNull() {
			if id, ok := idByDataHash[planEntries[i].DataHash()]; ok {
				planEntries[i].ID = types.StringValue(id)
				tflog.Trace(ctx, "collection_data ModifyPlan - Entry found", map[string]interface{}{"id": id})
			} else {
				tflog.Trace(ctx, "collection_data ModifyPlan - Entry not found", map[string]interface{}{"data": planEntries[i].Data.ValueString()})
			}
		}
	}

	if !config.Entries.IsUnknown() {
		plan.Entries, diags = types.SetValueFrom(ctx, plan.Entries.ElementType(ctx), planEntries)
		resp.Diagnostics.Append(diags...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// diff state vs plan entries, compute "generation" value if resource changes

	stateEntriesHash, planEntriesHash := make(map[string]struct{}), make(map[string]struct{})
	planHasUnknownValues := false
	for _, e := range stateEntries {
		stateEntriesHash[fmt.Sprintf("%s%s", e.ID, e.DataHash())] = struct{}{}
	}
	for _, e := range planEntries {
		isUnknown := e.Data.IsUnknown()
		planHasUnknownValues = isUnknown || planHasUnknownValues
		if !isUnknown {
			planEntriesHash[fmt.Sprintf("%s%s", e.ID, e.DataHash())] = struct{}{}
		}
	}

	if planHasUnknownValues {
		plan.Generation = types.Int64Unknown()
	} else if state.Scope == plan.Scope && reflect.DeepEqual(stateEntriesHash, planEntriesHash) {
		plan.Generation = state.Generation
	} else if !req.State.Raw.IsNull() {
		plan.Generation = types.Int64Value(state.Generation.ValueInt64() + 1)
	}

	diags = resp.Plan.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Finished modifying plan for collecton data resource")

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

	state.Entries, diags = types.SetValueFrom(ctx, state.Entries.ElementType(ctx), entries)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	diags = state.Normalize(ctx)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)

	tflog.Trace(ctx, "Finished reading collecton data resource")
}

func (r *resourceCollectionData) createOrUpdate(ctx context.Context, config, plan collectionDataModel, update bool) (state collectionDataModel, diags diag.Diagnostics) {
	var planEntries, configEntries []collectionEntryModel

	if diags.Append(plan.Entries.ElementsAs(ctx, &planEntries, false)...); diags.HasError() {
		return
	}
	if diags.Append(config.Entries.ElementsAs(ctx, &configEntries, false)...); diags.HasError() {
		return
	}

	for i := range planEntries {
		if planEntries[i].ID.IsUnknown() && configEntries[i].ID.IsNull() {
			planEntries[i].ID = types.StringValue(uuid.New().String())
		}
		diags.Append(collectionDataEntryIsValid().ValidateEntry(planEntries[i].Data.ValueString())...)
	}
	if diags.HasError() {
		return
	}

	var diags_ diag.Diagnostics
	plan.Entries, diags_ = types.SetValueFrom(ctx, plan.Entries.ElementType(ctx), planEntries)
	if diags.Append(diags_...); diags.HasError() {
		return
	}

	api := NewCollectionDataAPI(plan, r.client)
	_, diags_ = api.CollectionExists(ctx, true)
	if diags.Append(diags_...); diags.HasError() {
		return
	}

	diags.Append(api.Save(ctx)...)
	if diags.HasError() {
		return
	}

	if update {
		if diags.Append(api.deleteOldRows(ctx)...); diags.HasError() {
			return
		}
	}

	if diags.Append(plan.Normalize(ctx)...); diags.HasError() {
		return
	}

	state = plan

	return
}

func (r *resourceCollectionData) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Preparing to create collecton data resource")
	var config, plan collectionDataModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	tflog.Trace(ctx, "collection_data Create - Parsed req config", map[string]interface{}{"config": config, "plan": plan})

	plan.ID = types.StringValue(uuid.New().String())
	plan.Generation = types.Int64Value(0)

	newState, diags := r.createOrUpdate(ctx, config, plan, false)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
	tflog.Trace(ctx, "Finished creating collecton data resource")
}

func (r *resourceCollectionData) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Preparing to update collecton data resource")
	var config, state, plan collectionDataModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	tflog.Trace(ctx, "collection_data Update - Parsed req config", map[string]interface{}{"config": config, "plan": plan, "state": state})

	newState, diags := r.createOrUpdate(ctx, config, plan, true)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
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
