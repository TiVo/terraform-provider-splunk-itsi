package provider

import (
	"testing"
)

func TestDataSourceSplunkSearchSchema(t *testing.T) {
	testDataSourceSchema(t, new(dataSourceSplunkSearch))
}
