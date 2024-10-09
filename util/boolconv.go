package util

import "strings"

func Atob(a any) bool {
	if a == nil {
		return false
	}
	switch v := a.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case float64:
		return (int(v) != 0)
	case string:
		return v != "" &&
			strings.TrimSpace(strings.ToLower(v)) != "false" &&
			strings.TrimSpace(strings.ToLower(v)) != "0"
	default:
		return false
	}
}

func Btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
