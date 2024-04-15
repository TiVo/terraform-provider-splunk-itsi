package provider

import (
	"testing"
)

func TestDataSourceCollectionSchema(t *testing.T) {
	testDataSourceSchema(t, new(dataSourceCollection))
}
