package provider

import (
	"encoding/json"
	"log"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tmccombs/hcl2json/convert"
)

var cleanerRegex *regexp.Regexp

var templateResourceSchema = `
resource "{{.ResourceType}}" "{{.ResourceName}}" {
    {{range $index, $field := .Fields}}
{{displayData $field}}
    {{end}}
}
`

func init() {
	var err error
	cleanerRegex, err = regexp.Compile(`(?m)^\s*$[\r\n]*|[\r\n]+\s+\z`)
	if err != nil {
		log.Fatal(err)
	}
}

func newTemplate(resourcetpl *resourceTemplate) (*template.Template, error) {
	fmap := template.FuncMap{
		"displayData": resourcetpl.displayData,
	}
	return template.New("resource").Funcs(fmap).Parse(templateResourceSchema)
}

type TFFormatter func(*models.Base) (string, error)

var Formatters map[string]TFFormatter = map[string]TFFormatter{
	"kpi_base_search":        kpiBSTFFormat,
	"kpi_threshold_template": kpiThresholdTemplateTFFormat,
	"entity":                 entityTFFormat,
	"service":                serviceTFFormat,
}

func JSONify(base *models.Base, formatter TFFormatter) (json.RawMessage, error) {
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

type resourceTemplate struct {
	ResourceName string
	ResourceType string
	Data         *schema.ResourceData
	Fields       []string
	Schema       map[string]*schema.Schema
}

func NewResourceTemplate(data *schema.ResourceData, resourceSchema map[string]*schema.Schema, title, resourceType string) (rt *resourceTemplate, err error) {
	rt = &resourceTemplate{
		ResourceType: resourceType,
		Data:         data,
		Fields:       []string{},
		Schema:       resourceSchema,
	}

	for k := range resourceSchema {
		rt.Fields = append(rt.Fields, k)
	}
	sort.Strings(rt.Fields)
	rt.ResourceName, err = rt.escape(data.Get(title).(string))
	return rt, err
}

func (rt *resourceTemplate) escape(name string) (string, error) {
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

func (rt *resourceTemplate) displayData(f interface{}) string {
	field := f.(string)

	var value interface{}
	var ok bool

	switch rt.Schema[field].Type {
	// https://github.com/hashicorp/terraform/issues/23138
	case schema.TypeBool:
		value, ok = rt.Data.GetOkExists(field)
	default:
		value, ok = rt.Data.GetOk(field)
	}
	if ok {
		sc := rt.Schema[field]
		return rt.display(field, value, sc, 2)
	}

	return ""
}

func (rt *resourceTemplate) display(index string, element interface{}, sc *schema.Schema, ndepth int) string {
	if element == nil {
		return ""
	}
	if sc.Computed && !sc.Optional {
		return ""
	}
	if sc.Optional && sc.Default != nil && sc.Default == element {
		return ""
	}
	if sc.Optional && sc.Default == nil && element == "" {
		return ""
	}
	whitespaces := strings.Repeat("    ", ndepth)
	suffix := whitespaces
	mapSuffix := whitespaces
	if len(index) > 0 {
		suffix = fmt.Sprintf("%s%s = ", suffix, index)
		mapSuffix = fmt.Sprintf("%s%s ", mapSuffix, index)
	}

	v := reflect.ValueOf(element)
	switch v.Kind() {

	case reflect.String:
		if sc.Optional && sc.Default != nil && v.String() == "" {
			return ""
		}
		if strings.Contains(v.String(), "\n") || strings.Contains(v.String(), "\\") || strings.Contains(v.String(), "\"") {
			split := strings.Split(v.String(), "\n")
			for i := 0; i < len(split); i++ {
				split[i] = fmt.Sprintf("%s%s", whitespaces, split[i])
			}
			return fmt.Sprintf(`
%s <<-EOT
%s
%sEOT
`, suffix, strings.Join(split, "\n"), whitespaces)
		}
		return fmt.Sprintf(`%s"%s"`, suffix, v)

	case reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128:
		return fmt.Sprintf("%s%v", suffix, v)

	case reflect.Slice:
		submap := false
		values := []string{}
		s := reflect.ValueOf(element)
		if s.Len() == 0 {
			return ""
		}
		for i := 0; i < s.Len(); i++ {
			if !s.Index(i).IsNil() {
				vv := reflect.ValueOf(s.Index(i).Interface())
				switch vv.Kind() {
				case reflect.Map:
					submap = true
					values = append(values, rt.display(index, vv.Interface(), sc, ndepth))
				default:
					values = append(values, rt.display("", vv.Interface(), sc, ndepth+1))
				}
			}
		}
		if submap {
			return strings.Join(values, "\n")
		} else {
			return fmt.Sprintf(`
%s[
%s
%s]
`, suffix, strings.Join(values, ",\n"), whitespaces)
		}

	case reflect.Map:
		values := []string{}
		fmap := map[string]reflect.Value{}
		fields := []string{}
		for _, f := range v.MapKeys() {
			fstring := fmt.Sprintf("%v", f)
			fields = append(fields, fstring)
			fmap[fstring] = f
		}
		sort.Strings(fields)
		for _, fstring := range fields {
			if !v.MapIndex(fmap[fstring]).IsNil() {
				var subsc *schema.Schema
				switch typed := sc.Elem.(type) {
				case *schema.Resource:
					subsc = typed.Schema[fstring]
				case *schema.Schema:
					subsc = typed
				default:
					subsc = sc
				}
				values = append(values, rt.display(fstring, v.MapIndex(fmap[fstring]).Interface(), subsc, ndepth+1))
			}
		}
		return fmt.Sprintf(`
%s{
%s
%s}
`, mapSuffix, strings.Join(values, "\n"), whitespaces)

	case reflect.Ptr:
		switch typed := element.(type) {
		case *schema.Schema:
			return rt.display(index, typed.Elem, typed, ndepth)
		case *schema.Set:
			return rt.display(index, typed.List(), sc, ndepth)
		default:
			// run the command by deferencing the pointer
			return rt.display(index, v.Elem(), sc, ndepth)
		}

	default:
		panic(fmt.Sprintf("handled type: for field %s, %+v is of type %T", index, element, element))
	}
}

func unpackResourceMap[T any](in map[string]interface{}) (map[string]T, error) {
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
