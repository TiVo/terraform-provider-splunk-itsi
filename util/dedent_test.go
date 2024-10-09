package util

import "testing"

func TestDedent(t *testing.T) {
	text := `
		This is a multiline
		text with indentation.
			It should be dedented.`

	expected := "\nThis is a multiline\ntext with indentation.\n\tIt should be dedented."

	result := Dedent(text)

	if result != expected {
		t.Errorf("Dedent() = %q, expected %q", result, expected)
	}
}
