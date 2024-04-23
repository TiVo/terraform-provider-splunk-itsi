package provider

import (
	"testing"
)

func TestDataSourceKpiThresholdTemplateSchema(t *testing.T) {
	testDataSourceSchema(t, new(dataSourceKpiThresholdTemplate))
}
