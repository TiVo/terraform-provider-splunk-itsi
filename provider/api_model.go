package provider

import (
	"context"
	"fmt"
	"maps"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

/*
	API Model Builder and Parser can be used define multi-step workflows
	for building and parsing API models.

	API Model Builder:
		- Given a tfmodel and a workflow, builds an API model
		- The workflow is a list of functions that take a tfmodel and return a map of fields
		- The final API model is populated with the fields returned by the workflow functions

	API Model Parser:
		- Given an API (models.Base) model and a workflow, returns a tfmodel
		- The workflow is a list of functions that take a map of fields and a tfmodel, and populate the tfmodel
		- The tfmodel is populated by the workflow functions

	NOTE:
		- tfmodel objects must represent the terraform-plugin-framework models and implement the tfmodel interface below
		- The tfmodel structs must have a types.String field with the tag `tfsdk:"id"`, which is used to parse and set an object's ID.
*/

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

type apibuildWorkflowStepFunc[T tfmodel] func(ctx context.Context, obj T) (map[string]any, diag.Diagnostics)
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
	body := make(map[string]any)

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

type apiparseWorkflowStepFunc[T tfmodel] func(ctx context.Context, fields map[string]any, res *T) diag.Diagnostics
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
