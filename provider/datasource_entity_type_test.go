package provider

import (
	"testing"
)

func TestDataSourceEntityTypeSchema(t *testing.T) {
	testDataSourceSchema(t, new(dataSourceEntityType))
}
