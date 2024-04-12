package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type jsonStringType int

const (
	jsonStringValidationError = "JSON string validation failed"

	jsonStringTypeNull    jsonStringType = 1 << 0
	jsonStringTypeObject  jsonStringType = 1 << 1
	jsonStringTypeArray   jsonStringType = 1 << 2
	jsonStringTypeString  jsonStringType = 1 << 3
	jsonStringTypeNumber  jsonStringType = 1 << 4
	jsonStringTypeBoolean jsonStringType = 1 << 5
	jsonStringTypeAny     jsonStringType = jsonStringTypeNull |
		jsonStringTypeObject |
		jsonStringTypeArray |
		jsonStringTypeString |
		jsonStringTypeNumber |
		jsonStringTypeBoolean
)

var validateStringID = struct {
	MinLength, MaxLength int
	RE                   *regexp.Regexp
	RegexpDescription    string
}{
	1, 255,
	regexp.MustCompile(`^[a-zA-Z][-_0-9a-zA-Z]*$`),
	"must begin with a letter and contain only alphanumerics, hyphens and underscores",
}

func validateStringIdentifier() []validator.String {
	return []validator.String{
		stringvalidator.LengthBetween(validateStringID.MinLength, validateStringID.MaxLength),
		stringvalidator.RegexMatches(
			validateStringID.RE,
			validateStringID.RegexpDescription,
		),
	}
}

// (1) [ String Validators ] ___________________________________________________

func stringvalidatorIsJSON(jsonType jsonStringType) validator.String {
	return jsonStringValidator{jsonType}
}

// (1.1) [ jsonStringValidator ] _________________________________________________

type jsonStringValidator struct{ t jsonStringType }

var _ validator.String = jsonStringValidator{}

func (v jsonStringValidator) AllowedTypes() (result []string) {
	if v.t == 0 {
		v.t = jsonStringTypeAny
	}

	for typename, typemeask := range map[string]jsonStringType{
		"null":    jsonStringTypeNull,
		"object":  jsonStringTypeObject,
		"array":   jsonStringTypeArray,
		"string":  jsonStringTypeString,
		"number":  jsonStringTypeNumber,
		"boolean": jsonStringTypeBoolean,
	} {
		if v.t&typemeask != 0 {
			result = append(result, typename)
		}
	}
	return
}

func (v jsonStringValidator) Description(_ context.Context) string {
	return fmt.Sprintf("string must be a valid JSON of one of the following types: %v", v.AllowedTypes())
}

func (v jsonStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v jsonStringValidator) ValidateJSON(data string) (diags diag.Diagnostics) {
	if v.t == 0 {
		v.t = jsonStringTypeAny
	}

	allowedTypes := v.AllowedTypes()
	var obj interface{}
	if err := json.Unmarshal([]byte(data), &obj); err != nil {
		diags.AddError(
			jsonStringValidationError,
			fmt.Sprintf("Expected a valid JSON string, got:\n%s", data),
		)
		return
	}

	if obj == nil && v.t&jsonStringTypeNull == 0 {
		diags.AddError(jsonStringValidationError, "Expected a non-null JSON object")
		return
	}

	if _, ok := obj.(map[string]interface{}); ok && v.t&jsonStringTypeObject == 0 {
		diags.AddError(jsonStringValidationError, fmt.Sprintf("JSON cannot be an object, allowed types: %v", allowedTypes))
		return
	}

	if _, ok := obj.([]interface{}); ok && v.t&jsonStringTypeArray == 0 {
		diags.AddError(jsonStringValidationError, fmt.Sprintf("JSON cannot be an array, allowed types: %v", allowedTypes))
		return
	}

	if _, ok := obj.(string); ok && v.t&jsonStringTypeString == 0 {
		diags.AddError(jsonStringValidationError, fmt.Sprintf("JSON cannot be a string, allowed types: %v", allowedTypes))
		return
	}

	if _, ok := obj.(float64); ok && v.t&jsonStringTypeNumber == 0 {
		diags.AddError(jsonStringValidationError, fmt.Sprintf("JSON cannot be a number, allowed types: %v", allowedTypes))
	}

	if _, ok := obj.(bool); ok && v.t&jsonStringTypeBoolean == 0 {
		diags.AddError(jsonStringValidationError, fmt.Sprintf("JSON cannot be a boolean, allowed types: %v", allowedTypes))
	}

	return
}

func (v jsonStringValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	diags := v.ValidateJSON(req.ConfigValue.ValueString())
	for _, d := range diags {
		resp.Diagnostics.Append(diag.WithPath(req.Path, d))
	}
}
