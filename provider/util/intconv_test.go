package util

import (
	"fmt"
	"testing"
)

func TestAtoi(t *testing.T) {
	tests := []struct {
		input    any
		expected int
		err      error
	}{
		{input: "123", expected: 123, err: nil},
		{input: "0", expected: 0, err: nil},
		{input: "-456", expected: -456, err: nil},
		{input: "abc", expected: 0, err: fmt.Errorf("strconv.Atoi: parsing \"abc\": invalid syntax")},
		{input: "", expected: 0, err: nil},
		{input: true, expected: 1, err: nil},
		{input: false, expected: 0, err: nil},
		{input: 123, expected: 123, err: nil},
		{input: 123.456, expected: 123, err: nil},
	}

	for _, tt := range tests {
		result, err := Atoi(tt.input)
		if result != tt.expected {
			t.Errorf("Atoi(%v): expected %v, got %v", tt.input, tt.expected, result)
		}
		if err != nil && err.Error() != tt.err.Error() {
			t.Errorf("Atoi(%v): expected error %v, got %v", tt.input, tt.err, err)
		}
	}
}
