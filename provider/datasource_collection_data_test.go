package provider

import (
	"testing"
)

func TestDataSourceCollectionDataSchema(t *testing.T) {
	testDataSourceSchema(t, new(dataSourceCollectionData))
}
