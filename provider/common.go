package provider

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tmccombs/hcl2json/convert"
)

const itsiDefaultSecurityGroup = "default_itsi_security_group"

type TFFormatter func(*models.ItsiObj) (string, error)

var Formatters map[string]TFFormatter = map[string]TFFormatter{
	//"kpi_base_search": kpiBSTFFormat,
	//"kpi_threshold_template":           kpiThresholdTemplateTFFormat,
	//"entity":                           entityTFFormat,
	//"entity_type":                      entityTypeTFFormat,
	//"service":                          serviceTFFormat,
	//"notable_event_aggregation_policy": notableEventAggregationPolicyTFFormat,
}

func JSONify(base *models.ItsiObj, formatter TFFormatter) (json.RawMessage, error) {
	b, err := formatter(base)
	if err != nil {
		return nil, err
	}
	var options convert.Options

	converted, err := convert.Bytes([]byte(b), base.TFID, options)
	if err != nil {
		return nil, err
	}

	var jm json.RawMessage
	err = json.Unmarshal(converted, &jm)
	return jm, err
}

func Escape(name string) (string, error) {
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		return "", err
	}
	name = reg.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	if unicode.IsDigit(rune(name[0])) {
		name = fmt.Sprintf("_%s", name)
	}
	return name, nil
}

func mapSubset[T comparable](m map[T]any, keys []T) map[T]any {
	ms := make(map[T]any)
	for _, k := range keys {
		if v, ok := m[k]; ok {
			ms[k] = v
		}
	}
	return ms
}

func unpackMap[T any](in map[string]any) (map[string]T, error) {
	out := make(map[string]T)
	for k, v := range in {
		res, ok := v.(T)

		if !ok {
			return nil, fmt.Errorf("failed to unpack %#v to map[string]%T ", in, *new(T))
		}
		out[k] = res
	}

	return out, nil
}

func UnpackSlice[T any](in any) ([]T, error) {
	slice, ok := in.([]any)
	if !ok {
		return nil, fmt.Errorf("failed to unpack %#v to []%T ", in, *new(T))
	}

	out := make([]T, 0, len(slice))
	for _, v := range slice {
		res, ok := v.(T)
		if !ok {
			return nil, fmt.Errorf("failed to unpack %#v to []%T ", in, *new(T))
		}
		out = append(out, res)
	}
	return out, nil
}

func marshalBasicTypesByTag(tag string, in any, out map[string]any) (diags diag.Diagnostics) {
	t := reflect.TypeOf(in).Elem()
	v := reflect.ValueOf(in).Elem()
	for i := range v.NumField() {
		field := v.Field(i)
		_tag := t.Field(i).Tag.Get(tag)
		if _tag == "" {
			// skipping field
			continue
		}
		fieldValue := field.Interface().(attr.Value)

		if fieldValue.IsNull() || fieldValue.IsUnknown() {
			// skipping field
			continue
		}
		switch field.Type().Name() {
		case "StringValue":
			out[_tag] = field.Interface().(basetypes.StringValue).ValueString()
		case "Float64Value":
			out[_tag] = field.Interface().(basetypes.Float64Value).ValueFloat64()
		case "BoolValue":
			out[_tag] = field.Interface().(basetypes.BoolValue).ValueBool()
		}
	}
	return
}
func unmarshalBasicTypesByTag(tag string, in map[string]any, out any) (diags diag.Diagnostics) {

	t := reflect.TypeOf(out).Elem()
	v := reflect.ValueOf(out).Elem()
	for i := range v.NumField() {
		field := v.Field(i)
		_tag := t.Field(i).Tag.Get(tag)
		if value, ok := in[_tag]; ok && value != nil {
			switch field.Type().Name() {
			case "StringValue":
				field.Set(reflect.ValueOf(types.StringValue(fmt.Sprintf("%v", value))))
			case "Float64Value":
				var val float64

				switch v := value.(type) {
				case string:
					val, _ = strconv.ParseFloat(v, 64)
				case float64:
					val = v
				default:
					val = 0
				}
				field.Set(reflect.ValueOf(types.Float64Value(val)))
			case "BoolValue":
				switch v := value.(type) {
				case float64:
					field.Set(reflect.ValueOf(types.BoolValue(v != 0)))
				case bool:
					field.Set(reflect.ValueOf(types.BoolValue(v)))
				}
			}
		} else {
			switch field.Type().Name() {
			case "StringValue":
				field.Set(reflect.ValueOf(types.StringNull()))
			case "Float64Value":
				field.Set(reflect.ValueOf(types.Float64Null()))
			case "Int64Value":
				field.Set(reflect.ValueOf(types.Int64Null()))
			}
		}
	}
	return
}
