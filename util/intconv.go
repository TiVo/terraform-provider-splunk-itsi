package util

import (
	"fmt"
	"strconv"
)

func Atoi(a any) (int, error) {
	if a == nil {
		return 0, nil
	}

	switch v := a.(type) {
	case int:
		return v, nil
	case float64:
		return int(v), nil
	case bool:
		return Btoi(v), nil
	case string:
		if v == "" {
			return 0, nil
		}
		return strconv.Atoi(v)

	default:
		return 0, fmt.Errorf("cannot convert %T to int", a)
	}
}
