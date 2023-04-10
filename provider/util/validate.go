package util

import (
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

func contains(str_list []string, target string) bool {
	for _, str := range str_list {
		if target == str {
			return true
		}
	}
	return false
}

func CheckInputValidString(valid_values []string) func(interface{}, cty.Path) diag.Diagnostics {
	return func(v interface{}, p cty.Path) diag.Diagnostics {
		value := v.(string)
		var diags diag.Diagnostics
		if flag := contains(valid_values, value); !flag {
			diag := diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Invalid value of parameter: %v", value),
				Detail:   fmt.Sprintf("Invalid value: %v\nValid values: %v", value, valid_values),
			}
			diags = append(diags, diag)
		}
		return diags
	}
}

func CheckInputValidInt(lower_bound int, upper_bound int) func(interface{}, cty.Path) diag.Diagnostics {
	return func(v interface{}, p cty.Path) diag.Diagnostics {
		value := v.(int)
		var diags diag.Diagnostics
		if value < lower_bound || value > upper_bound {
			diag := diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Invalid value of parameter: %v", value),
				Detail:   fmt.Sprintf("Invalid value: %v\nValid values: number between %v and %v (inclusive)", value, lower_bound, upper_bound),
			}
			diags = append(diags, diag)
		}
		return diags
	}
}
