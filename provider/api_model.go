package provider

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

type tfmodel interface {
	objectype() string
	title() string
}

func tfmodelID(m any) *types.String {
	val := reflect.ValueOf(m)
	typ := reflect.TypeOf(m)

	if typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("tfmodelID: expected a pointer to a tfmodel struct, got %s", typ.Kind()))
	}

	val = val.Elem()
	typ = typ.Elem()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get(`tfsdk`)
		if tag == "id" {
			id := val.Field(i).Addr().Interface().(*types.String)
			return id
		}
	}

	panic("tfmodelID: tfmodel struct does not have an id field")
}

// api model builder

type apibuildWorkflowStepFunc[T tfmodel] func(ctx context.Context, obj T) (map[string]interface{}, diag.Diagnostics)
type apibuildWorkflow[T tfmodel] interface {
	buildSteps() []apibuildWorkflowStepFunc[T]
}

// api model builder,
// given a tfmodel and a workflow, builds an api model
type apibuilder[T tfmodel] struct {
	client   models.ClientConfig
	workflow apibuildWorkflow[T]
}

func newAPIBuilder[T tfmodel](client models.ClientConfig, buildWorkflow apibuildWorkflow[T]) *apibuilder[T] {
	return &apibuilder[T]{
		client:   client,
		workflow: buildWorkflow,
	}
}

func (b *apibuilder[T]) build(ctx context.Context, obj T) (m *models.Base, diags diag.Diagnostics) {
	body := make(map[string]interface{})

	for _, step := range b.workflow.buildSteps() {
		stepbody, d := step(ctx, obj)
		if diags.Append(d...); d.HasError() {
			return
		}

		maps.Copy(body, stepbody)
	}
	id := tfmodelID(&obj).ValueString()
	m = models.NewBase(b.client, id, obj.title(), obj.objectype())
	if err := m.PopulateRawJSON(ctx, body); err != nil {
		diags.AddError(fmt.Sprintf("%s/%s: failed to populate api model", obj.objectype(), id), err.Error())
	}

	tflog.Trace(ctx, fmt.Sprintf("API Model Builder : [ %s/%s ] :\n%s\n", obj.objectype(), id, string(m.RawJson)))

	return
}

// api model parser

type apiparseWorkflowStepFunc[T tfmodel] func(ctx context.Context, fields map[string]interface{}, res *T) diag.Diagnostics
type apiparseWorkflow[T tfmodel] interface {
	parseSteps() []apiparseWorkflowStepFunc[T]
}

// api model parser,
// given an API (models.Base) model and a workflow, returns a tfmodel
type apiparser[T tfmodel] struct {
	base     *models.Base
	workflow apiparseWorkflow[T]
}

func newAPIParser[T tfmodel](base *models.Base, parseWorkflow apiparseWorkflow[T]) *apiparser[T] {
	return &apiparser[T]{
		base:     base,
		workflow: parseWorkflow,
	}
}

func (p *apiparser[T]) parse(ctx context.Context, base *models.Base) (obj T, diags diag.Diagnostics) {
	if base == nil || base.RawJson == nil {
		diags.AddError(fmt.Sprintf("Unable to populate %s tfmodel", base.ObjectType), "base object is nil or empty.")
		return
	}

	fields, err := base.RawJson.ToInterfaceMap()
	if err != nil {
		diags.AddError("Unable to populate entity model", err.Error())
		return
	}

	id := tfmodelID(&obj)
	*id = types.StringValue(base.RESTKey)

	for _, step := range p.workflow.parseSteps() {
		if diags.Append(step(ctx, fields, &obj)...); diags.HasError() {
			return
		}
	}

	return
}

//////////////////////////////////////////
//////////////////////////////////////////
//////////////////////////////////////////

type neapBuildWorkflow struct{}

func (w *neapBuildWorkflow) basics(ctx context.Context, obj neapModel) (map[string]interface{}, diag.Diagnostics) {
	return map[string]interface{}{
		"object_type": obj.objectype(),
		"title":       obj.Title.ValueString(),
		"description": obj.Description.ValueString(),

		"disabled":                    util.Btoi(obj.Disabled.ValueBool()),
		"priority":                    obj.Priority.ValueInt64(),
		"run_time_based_actions_once": obj.RunTimeBasedActionsOnce.ValueBool(),
		"service_topology_enabled":    obj.ServiceTopologyEnabled.ValueBool(),
		"entity_factor_enabled":       obj.EntityFactorEnabled.ValueBool(),

		"ace_enabled": 0, //TODO: support smart mode
	}, nil
}

func (w *neapBuildWorkflow) episodeInfo(ctx context.Context, obj neapModel) (map[string]interface{}, diag.Diagnostics) {
	return map[string]interface{}{
		"group_title":             obj.GroupTitle.ValueString(),
		"group_description":       obj.GroupDescription.ValueString(),
		"group_severity":          obj.GroupSeverity.ValueString(),
		"group_assignee":          obj.GroupAssignee.ValueString(),
		"group_status":            obj.GroupStatus.ValueString(),
		"group_instruction":       obj.GroupInstruction.ValueString(),
		"group_custom_instuction": obj.GroupCustomInstruction.ValueString(),
		"group_dashboard":         obj.GroupDashboard.ValueString(),
		"group_dashboard_context": obj.GroupDashboardContext.ValueString(),
	}, nil
}

func (w *neapBuildWorkflow) criteria(ctx context.Context, obj neapModel) (map[string]interface{}, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	var splitByFields []string
	diags = append(diags, obj.SplitByField.ElementsAs(ctx, &splitByFields, false)...)

	filterCriteria, d := obj.FilterCriteria.apiModel(neapCriteriaTypeFilter)
	diags.Append(d...)

	breakingCriteria, d := obj.BreakingCriteria.apiModel(neapCriteriaTypeBreaking)
	diags.Append(d...)

	if diags.HasError() {
		return nil, diags
	}

	return map[string]interface{}{
		"split_by_field":    strings.Join(splitByFields, ","),
		"filter_criteria":   filterCriteria,
		"breaking_criteria": breakingCriteria,
	}, diags
}

func (w *neapBuildWorkflow) rules(ctx context.Context, obj neapModel) (map[string]interface{}, diag.Diagnostics) {
	var d, diags diag.Diagnostics
	rules := make([]map[string]any, len(obj.Rules))

	for i, rule := range obj.Rules {
		rules[i], d = rule.apiModel()
		diags.Append(d...)
	}

	return map[string]interface{}{
		"rules": rules,
	}, diags
}

//lint:ignore U1000 used by apibuilder
func (w *neapBuildWorkflow) buildSteps() []apibuildWorkflowStepFunc[neapModel] {
	return []apibuildWorkflowStepFunc[neapModel]{w.basics, w.episodeInfo, w.criteria, w.rules}
}

//////////////////////////////////////////
//////////////////////////////////////////
//////////////////////////////////////////

type neapParseWorkflow struct{}

func (w *neapParseWorkflow) basics(ctx context.Context, fields map[string]interface{}, res *neapModel) (diags diag.Diagnostics) {
	unexpectedErrorMsg := "NEAP: Unexpected error while populating basic fields of a NEAP model"
	strFields, err := unpackMap[string](mapSubset(fields, []string{"title", "description"}))
	if err != nil {
		diags.AddError(unexpectedErrorMsg, err.Error())
	}
	boolFields, err := unpackMap[bool](mapSubset(fields, []string{"run_time_based_actions_once", "service_topology_enabled", "entity_factor_enabled"}))
	if err != nil {
		diags.AddError(unexpectedErrorMsg, err.Error())
	}

	res.Title = types.StringValue(strFields["title"])
	res.Description = types.StringValue(strFields["description"])

	res.Disabled = types.BoolValue(int(fields["disabled"].(float64)) != 0)
	res.Priority = types.Int64Value(int64(fields["priority"].(float64)))
	res.RunTimeBasedActionsOnce = types.BoolValue(boolFields["run_time_based_actions_once"])
	res.ServiceTopologyEnabled = types.BoolValue(boolFields["service_topology_enabled"])
	res.EntityFactorEnabled = types.BoolValue(boolFields["entity_factor_enabled"])

	return nil
}

func (w *neapParseWorkflow) episodeInfo(ctx context.Context, fields map[string]interface{}, res *neapModel) (diags diag.Diagnostics) {
	strFields, err := unpackMap[string](mapSubset(fields, []string{
		"group_title",
		"group_description",
		"group_severity",
		"group_assignee",
		"group_status",
		"group_instruction",
		"group_custom_instuction",
		"group_dashboard",
		"group_dashboard_context",
	}))

	if err != nil {
		diags.AddError("NEAP: Unable to parse episode info fields. ", err.Error())
		return
	}

	res.GroupTitle = types.StringValue(strFields["group_title"])
	res.GroupDescription = types.StringValue(strFields["group_description"])
	res.GroupSeverity = types.StringValue(strFields["group_severity"])
	res.GroupAssignee = types.StringValue(strFields["group_assignee"])
	res.GroupStatus = types.StringValue(strFields["group_status"])
	res.GroupInstruction = types.StringValue(strFields["group_instruction"])
	res.GroupCustomInstruction = types.StringValue(strFields["group_custom_instuction"])
	res.GroupDashboard = types.StringValue(strFields["group_dashboard"])
	res.GroupDashboardContext = types.StringValue(strFields["group_dashboard_context"])

	return nil
}

func (w *neapParseWorkflow) criteria(ctx context.Context, fields map[string]interface{}, res *neapModel) (diags diag.Diagnostics) {
	splitByFields := []string{}
	if splitByField, ok := fields["split_by_field"]; ok && splitByField != "" {
		splitByFields = strings.Split(splitByField.(string), ",")
	}
	res.SplitByField, diags = types.SetValueFrom(ctx, types.StringType, splitByFields)
	if diags.HasError() {
		return
	}

	itsiFilterCriteria, ok := fields["filter_criteria"].(map[string]interface{})
	if !ok {
		diags.AddError("NEAP: Unable to parse filter criteria", "filter_criteria is missing or not in the expected format")
		return
	}

	var d diag.Diagnostics
	res.FilterCriteria, d = newNEAPCriteriaFromAPIModel(itsiFilterCriteria)
	diags.Append(d...)

	itsiBreakingCriteria, ok := fields["breaking_criteria"].(map[string]interface{})
	if !ok {
		diags.AddError("NEAP: Unable to parse breaking criteria", "breaking_criteria is missing or not in the expected format")
		return
	}
	res.BreakingCriteria, d = newNEAPCriteriaFromAPIModel(itsiBreakingCriteria)
	diags.Append(d...)

	if diags.HasError() {
		return
	}

	return nil
}

func (w *neapParseWorkflow) rules(ctx context.Context, fields map[string]interface{}, res *neapModel) (diags diag.Diagnostics) {
	rules, err := unpackSlice[map[string]any](fields["rules"])
	if err != nil {
		diags.AddError("NEAP: Unable to parse rules", err.Error())
		return
	}

	res.Rules = make([]neapRuleModel, len(rules))
	for i, rule := range rules {
		r, d := NEAPRuleFromAPIModel(rule)
		if diags.Append(d...); diags.HasError() {
			return
		}
		res.Rules[i] = r
	}

	return
}

//lint:ignore U1000 used by apiparser
func (w *neapParseWorkflow) parseSteps() []apiparseWorkflowStepFunc[neapModel] {
	return []apiparseWorkflowStepFunc[neapModel]{w.basics, w.episodeInfo, w.criteria, w.rules}
}
