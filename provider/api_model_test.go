package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestTfmodelID(t *testing.T) {
	type TestTFModel struct {
		ID types.String `tfsdk:"id"`
	}
	m := &TestTFModel{ID: types.StringValue("123")}
	result := tfmodelID(m)
	if result != &m.ID {
		t.Errorf("tfmodelID(%v) returned unexpected result: %#v", m, result)
	}
}

func TestTfmodelID_MissingIDTag(t *testing.T) {
	type wrongTFModel struct{}
	m := &wrongTFModel{}
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("tfmodelID should have panicked")
		}
	}()
	tfmodelID(m)
}
