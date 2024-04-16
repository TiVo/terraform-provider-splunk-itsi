package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

const itsiResourceTypeKPIThresholdTemplate = "kpi_threshold_template"

// tf models

type kpiThresholdTemplateManifestModel struct {
	ID            types.String `tfsdk:"id"`
	Title         types.String `tfsdk:"title"`
	Description   types.String `tfsdk:"description"`
	SecurityGroup types.String `tfsdk:"security_group"`

	//Spec          types.Dynamic `tfsdk:"spec"`

	Spec types.String `tfsdk:"spec"`
}

func (m kpiThresholdTemplateManifestModel) objectype() string {
	return itsiResourceTypeKPIThresholdTemplate
}

func (m kpiThresholdTemplateManifestModel) title() string {
	return m.Title.String()
}

// resource schema

type resourceKPIThresholdTemplateManifest struct {
	client models.ClientConfig
}

func NewResourceKPIThresholdTemplateManifest() resource.Resource {
	return &resourceKPIThresholdTemplateManifest{}
}

func (r *resourceKPIThresholdTemplateManifest) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *resourceKPIThresholdTemplateManifest) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + "kpi_threshold_template_manifest"
}

func (r *resourceKPIThresholdTemplateManifest) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Required: true,
			},
			"security_group": schema.StringAttribute{
				Required: true,
			},
			// "spec": schema.DynamicAttribute{
			// 	Required: true,
			// },
			"spec": schema.StringAttribute{
				Required: true,
			},
		},
	}
}

// TF <-> ITSI Build / Parse Workflows

// build workflow

type kpittmBuildWorkflow struct{}

var _ apibuildWorkflow[kpiThresholdTemplateManifestModel] = &kpittmBuildWorkflow{}

//lint:ignore U1000 used by apibuilder
func (w *kpittmBuildWorkflow) buildSteps() []apibuildWorkflowStepFunc[kpiThresholdTemplateManifestModel] {
	return []apibuildWorkflowStepFunc[kpiThresholdTemplateManifestModel]{w.basics, w.spec}
}

func (w *kpittmBuildWorkflow) basics(ctx context.Context, obj kpiThresholdTemplateManifestModel) (map[string]any, diag.Diagnostics) {
	return map[string]any{
		"object_type": obj.objectype(),
		"title":       obj.Title.ValueString(),
		"description": obj.Description.ValueString(),
		"sec_grp":     obj.SecurityGroup.ValueString(),
	}, nil
}

func (w *kpittmBuildWorkflow) spec(ctx context.Context, obj kpiThresholdTemplateManifestModel) (specFields map[string]any, diags diag.Diagnostics) {
	//str := obj.Spec.String()
	//fmt.Println(str)

	specFields = make(map[string]any)

	err := json.Unmarshal([]byte(obj.Spec.ValueString()), &specFields)
	if err != nil {
		diags.AddError("NEAP: Unable to parse spec", err.Error())
	}

	return
}

// parse workflow

type kpittmParseWorkflow struct{}

var _ apiparseWorkflow[kpiThresholdTemplateManifestModel] = &kpittmParseWorkflow{}

//lint:ignore U1000 used by apiparser
func (w *kpittmParseWorkflow) parseSteps() []apiparseWorkflowStepFunc[kpiThresholdTemplateManifestModel] {
	return []apiparseWorkflowStepFunc[kpiThresholdTemplateManifestModel]{w.basics, w.spec}
}

func (w *kpittmParseWorkflow) basics(ctx context.Context, fields map[string]any, res *kpiThresholdTemplateManifestModel) (diags diag.Diagnostics) {
	unexpectedErrorMsg := "NEAP: Unexpected error while populating basic fields of a NEAP model"
	strFields, err := unpackMap[string](mapSubset(fields, []string{"title", "description", "sec_grp"}))
	if err != nil {
		diags.AddError(unexpectedErrorMsg, err.Error())
	}

	res.Title = types.StringValue(strFields["title"])
	res.Description = types.StringValue(strFields["description"])
	res.SecurityGroup = types.StringValue(strFields["sec_grp"])

	return
}

func (w *kpittmParseWorkflow) spec(ctx context.Context, fields map[string]any, res *kpiThresholdTemplateManifestModel) (diags diag.Diagnostics) {

	specFields := map[string]any{
		"adaptive_thresholds_is_enabled":        fields["adaptive_thresholds_is_enabled"],
		"adaptive_thresholding_training_window": fields["adaptive_thresholding_training_window"],
		"time_variate_thresholds":               fields["time_variate_thresholds"],
		"time_variate_thresholds_specification": fields["time_variate_thresholds_specification"],
	}

	json, err := json.Marshal(specFields)
	if err != nil {
		diags.AddError("NEAP: Unable to marshal spec", err.Error())
	}

	// dv, err := types.DynamicType.ValueFromTerraform(ctx, v)
	// if err != nil {
	// 	diags.AddError("KPI TT: Unable to create spec object", err.Error())
	// }
	// res.Spec = types.DynamicValue(dv)

	res.Spec = types.StringValue(string(json))
	return
}

// CRUD Operations

func (r *resourceKPIThresholdTemplateManifest) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state kpiThresholdTemplateManifestModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	base := kpiThresholdTemplateBase(r.client, state.ID.ValueString(), state.Title.ValueString())
	b, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read NEAP", err.Error())
		return
	}
	if b == nil || b.RawJson == nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &kpiThresholdTemplateManifestModel{})...)
		return
	}

	state, diags := newAPIParser(b, new(kpittmParseWorkflow)).parse(ctx, b)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *resourceKPIThresholdTemplateManifest) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var plan kpiThresholdTemplateManifestModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	base, diags := newAPIBuilder(r.client, new(kpittmBuildWorkflow)).build(ctx, plan)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	base, err := base.Create(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create KPI Threshold Template", err.Error())
		return
	}

	// state, diags := newAPIParser(base, new(neapParseWorkflow)).parse(ctx, base)
	// if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
	// 	return
	// }
	//resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	plan.ID = types.StringValue(base.RESTKey)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *resourceKPIThresholdTemplateManifest) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan kpiThresholdTemplateManifestModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	base, diags := newAPIBuilder(r.client, new(kpittmBuildWorkflow)).build(ctx, plan)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}
	existing, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update kpi threshold template", err.Error())
		return
	}
	if existing == nil {
		resp.Diagnostics.AddError("Unable to update kpi threhsold template", "kpi threshold template not found")
		return
	}
	if err := base.Update(ctx); err != nil {
		resp.Diagnostics.AddError("Unable to update kpi threshold template", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *resourceKPIThresholdTemplateManifest) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state kpiThresholdTemplateManifestModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	base := kpiThresholdTemplateBase(r.client, state.ID.ValueString(), state.Title.ValueString())
	b, err := base.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete kpi threshold template", err.Error())
		return
	}
	if b == nil {
		return
	}
	if err := b.Delete(ctx); err != nil {
		resp.Diagnostics.AddError("Unable to delete kpi threshold template", err.Error())
		return
	}
}

func (r *resourceKPIThresholdTemplateManifest) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	b := kpiThresholdTemplateBase(r.client, "", req.ID)
	b, err := b.Find(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to find entity type model", err.Error())
		return
	}
	if b == nil {
		resp.Diagnostics.AddError("Entity type not found", fmt.Sprintf("Entity type '%s' not found", req.ID))
		return
	}

	state, diags := newAPIParser(b, new(kpittmParseWorkflow)).parse(ctx, b)
	if resp.Diagnostics.Append(diags...); diags.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
