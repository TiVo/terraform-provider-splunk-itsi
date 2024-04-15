package provider

import (
	"testing"
)

func TestDataSourceKPIBaseSearchSchema(t *testing.T) {
	testDataSourceSchema(t, new(dataSourceKpiBaseSearch))
}
