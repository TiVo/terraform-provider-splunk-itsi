package util

import (
	"reflect"
	"testing"
)

func TestReverseMap(t *testing.T) {
	// Test case 1: Empty map
	input1 := map[int]string{}
	expected1 := map[string]int{}
	output1 := ReverseMap(input1)
	if !reflect.DeepEqual(output1, expected1) {
		t.Errorf("ReverseMap(%v) = %v, expected %v", input1, output1, expected1)
	}

	// Test case 2: Map with unique keys and values
	input2 := map[string]int{
		"apple":  1,
		"banana": 2,
		"cherry": 3,
	}
	expected2 := map[int]string{
		1: "apple",
		2: "banana",
		3: "cherry",
	}
	output2 := ReverseMap(input2)
	if !reflect.DeepEqual(output2, expected2) {
		t.Errorf("ReverseMap(%v) = %v, expected %v", input2, output2, expected2)
	}

	// Test case 3: Map with duplicate values
	input3 := map[int]string{
		1: "apple",
		2: "banana",
		3: "apple",
	}
	// Since the input map has duplicate values, ReverseMap should panic
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		_ = ReverseMap(input3)
	}()
	if !panicked {
		t.Errorf("ReverseMap(%v) did not panic as expected", input3)
	}
}
