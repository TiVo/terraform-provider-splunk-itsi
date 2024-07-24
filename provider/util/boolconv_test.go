package util

import "testing"

func TestAtob(t *testing.T) {
	// Test cases
	testCases := []struct {
		input    any
		expected bool
	}{
		{input: "true", expected: true},
		{input: "false", expected: false},
		{input: "1", expected: true},
		{input: "0", expected: false},
		{input: 1, expected: true},
		{input: 0, expected: false},
		{input: 2.222222, expected: true},
		{input: 0.000000000000001, expected: false},
	}

	// Run test cases
	for _, tc := range testCases {
		result := Atob(tc.input)
		if result != tc.expected {
			t.Errorf("Atob(%v) = %v, expected %v", tc.input, result, tc.expected)
		}
	}
}
