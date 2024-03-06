package provider

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

/*

	Higher-level (than models.CollectionAPI) API clients
	for manipulating Splunk collections, focusing on the interoperability with
	our terraform-plugin-framework collection data structures.

	collectionAPI:
		Base client for all collection API operations.

	collectionConfigAPI:
		Client for manipulating collection configuration.

	collectionDataAPI:
		Client for manipulating collection data.


                    ┌──────────────────────────┐
                    │      collectionAPI       │
                    └──────────────────────────┘
                                  │
              ┌───────────────────┴─────────────────┐
              │                                     │
              ▼                                     ▼
┌──────────────────────────┐          ┌──────────────────────────┐
│   collectionConfigAPI    │          │    collectionDataAPI     │
└──────────────────────────┘          └──────────────────────────┘

*/

// collectionAPI client

type collectionAPI struct {
	collectionIDModel
	client models.ClientConfig
}

func NewCollectionAPI(m collectionIDModel, c models.ClientConfig) *collectionAPI {
	return &collectionAPI{m, c}
}

func (api *collectionAPI) Model(objectType string) *models.CollectionApi {
	return models.NewCollection(
		api.client,
		api.Name.ValueString(),
		api.App.ValueString(),
		api.Owner.ValueString(),
		api.Name.ValueString(),
		objectType)
}

func (api *collectionAPI) CollectionExists(ctx context.Context, require bool) (exists bool, diags diag.Diagnostics) {
	collection := api.Model("collection_config_no_body")
	collection.SetCodeHandle(http.StatusNotFound, util.Ignore)
	collection_, err := collection.Read(ctx)
	if err != nil {
		diags.AddError(fmt.Sprintf("Unable to read %s collection config", api.Key()), err.Error())
		return
	}
	if exists = collection_ != nil; require && !exists {
		diags.AddError("collection not found",
			fmt.Sprintf("Collection %s does not exist", api.Key()))
	}
	return
}

func (api *collectionAPI) Query(ctx context.Context, query string, fields []string, limit int) (obj interface{}, diags diag.Diagnostics) {
	collection := api.Model("collection_data")
	var params, queryParams, fieldsParams string

	if strings.TrimSpace(query) != "" {
		queryParams = "query=" + url.QueryEscape(query)
	}
	paramsList := []string{queryParams}
	if len(fields) > 0 {
		urlEscapedFields := make([]string, len(fields))
		for i, field := range fields {
			urlEscapedFields[i] = url.QueryEscape(field)
		}
		fieldsParams = "fields=" + strings.Join(urlEscapedFields, ",")
		paramsList = append(paramsList, fieldsParams)
	}
	if limit > 0 {
		paramsList = append(paramsList, fmt.Sprintf("limit=%v", limit))
	}
	params = strings.Join(paramsList, "&")
	collection.Params = params

	var err error
	if collection, err = collection.Read(ctx); err != nil {
		diags.AddError(fmt.Sprintf("Unable to read %s collection data", api.Key()), err.Error())
		return
	}

	if obj, err = collection.Unmarshal(collection.Body); err != nil {
		diags.AddError(fmt.Sprintf("Unable to unmarshal %s collection data", api.Key()), err.Error())
	}

	return
}

type CollectionCondition int

const (
	CollectionExists CollectionCondition = iota
	CollectionDoesNotExist
)

func (api *collectionAPI) Wait(ctx context.Context, condition CollectionCondition) (diags diag.Diagnostics) {
	// Splunk's collection config APIs are not synchronous,
	// which means that we cannot assume that a collection exists (or is deleted) after
	// a create (or delete) request has been made. This function attempts to wait for a collection
	// until it exists (or does not exist).
	// The solution below is not ideal, because it does not guarantee that the collection config will be
	// replicated to all search heads in a distributed environment. But it's better than nothing.

	const checkInterval = time.Duration(15) * time.Second
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()
	var done bool

	for !done {
		select {
		case <-ctx.Done():
			diags.AddError("Timeout", "Timeout while waiting for collection")
			done = true
		case <-ticker.C:
			exists, diags := api.CollectionExists(ctx, false)
			if diags.HasError() {
				return diags
			}

			done = (condition == CollectionExists && exists) || (condition == CollectionDoesNotExist && !exists)
		}
	}

	return
}

//  collectionConfigAPI client

type collectionConfigAPI struct {
	*collectionAPI
	config collectionConfigModel
}

func NewCollectionConfigAPI(m collectionConfigModel, c models.ClientConfig) *collectionConfigAPI {
	return &collectionConfigAPI{NewCollectionAPI(m.CollectionIDModel(), c), m}
}

func (api *collectionConfigAPI) Model(ctx context.Context, objectType string) (model *models.CollectionApi, diags diag.Diagnostics) {
	model = api.collectionAPI.Model(objectType)

	fieldTypes := make(map[string]string)
	var accelerations []string

	if !api.config.FieldTypes.IsNull() {
		for field, fieldType := range api.config.FieldTypes.Elements() {
			fieldTypes[field] = fieldType.String()
		}
	}
	if !api.config.Accelerations.IsNull() {
		if diags.Append(api.config.Accelerations.ElementsAs(ctx, &accelerations, false)...); diags.HasError() {
			return
		}
	}

	model.Data = map[string]interface{}{
		"field_types":   fieldTypes,
		"accelerations": accelerations,
		"name":          api.config.Name.ValueString(),
		"app":           api.config.App.ValueString(),
		"owner":         api.config.Owner.ValueString(),
	}

	return
}

func collectionConfigModelFromAPIModel(ctx context.Context, c *models.CollectionApi) (model collectionConfigModel, diags diag.Diagnostics) {
	type Key struct {
		XmlName xml.Name
		Name    string `xml:"name,attr"`
		Value   string `xml:",chardata"`
	}
	type Dict struct {
		XmlName xml.Name
		Attrs   []xml.Attr `xml:",any,attr"`
		Keys    []Key      `xml:"http://dev.splunk.com/ns/rest key"`
	}
	type Content struct {
		XmlName xml.Name
		Type    string `xml:"type,attr"`
		Dicts   []Dict `xml:"http://dev.splunk.com/ns/rest dict"`
	}
	type Feed struct {
		XmlName xml.Name
		Attrs   []xml.Attr `xml:",any,attr"`
		Content []Content  `xml:"entry>content"`
	}

	// Unmarshal the XML body...
	var feed Feed
	if err := xml.Unmarshal(c.Body, &feed); err != nil {
		diags.AddError("Failed to unmarshal collection API response", err.Error())
		return
	}
	keys := feed.Content[0].Dicts[0].Keys
	tflog.Trace(ctx, "RSRC COLLECTION:   content",
		map[string]interface{}{"num": len(keys), "keys": fmt.Sprintf("%+v", keys)})

	// Iterate through the feed finding fieldTypes and
	// accelerations...
	fieldTypes := map[string]string{}
	accelerations := make([]string, 0, len(keys))
	for _, key := range keys {
		if strings.HasPrefix(key.Name, "field.") {
			k := strings.TrimPrefix(key.Name, "field.")
			fieldTypes[k] = strings.Replace(key.Value, "\"", "", -1)
		} else if strings.HasPrefix(key.Name, "accelerated_fields.accel_") {
			s := strings.TrimPrefix(key.Name, "accelerated_fields.accel_")
			if i, err := strconv.Atoi(s); err == nil && i < cap(keys) {
				accelerations = accelerations[0 : i+1]
				accelerations[i] = key.Value
			}
		}
	}

	c.Data["field_types"] = fieldTypes
	c.Data["accelerations"] = accelerations

	model.Name = types.StringValue(c.Data["name"].(string))
	model.App = types.StringValue(c.Data["app"].(string))
	model.Owner = types.StringValue(c.Data["owner"].(string))

	fieldTypesMap, diag := types.MapValueFrom(ctx, types.StringType, fieldTypes)
	diags.Append(diag...)
	model.FieldTypes = fieldTypesMap

	accelerationsList, diag := types.ListValueFrom(ctx, types.StringType, accelerations)
	diags.Append(diag...)
	model.Accelerations = accelerationsList

	tflog.Trace(ctx, "RSRC COLLECTION:   field types", map[string]interface{}{"field_types": fieldTypes, "len": len(fieldTypes)})
	tflog.Trace(ctx, "RSRC COLLECTION:   accelerations", map[string]interface{}{"accelerations": accelerations, "len": len(accelerations)})

	return
}

func (api *collectionConfigAPI) Read(ctx context.Context) (diags diag.Diagnostics) {
	model, d := api.Model(ctx, "collection_config")
	if diags.Append(d...); diags.HasError() {
		return
	}
	var err error
	if model, err = model.Read(ctx); err != nil {
		diags.AddError(fmt.Sprintf("Failed to read collection config for '%s' collection", api.collectionIDModel.Key()), err.Error())
		return
	}
	api.config, diags = collectionConfigModelFromAPIModel(ctx, model)
	return
}

func (api *collectionConfigAPI) Create(ctx context.Context) (diags diag.Diagnostics) {
	model, d := api.Model(ctx, "collection_config_keyless")
	if diags.Append(d...); diags.HasError() {
		return
	}

	body := url.Values{}
	body.Set("name", api.Name.ValueString())

	for k, v := range model.Data["field_types"].(map[string]string) {
		body.Set(fmt.Sprintf("field.%s", k), v)
	}
	for k, v := range model.Data["accelerations"].([]string) {
		body.Set(fmt.Sprintf("accelerated_fields.accel_%03d", k), v)
	}

	model.Body = []byte(body.Encode())

	if _, err := model.Create(ctx); err != nil {
		diags.AddError("Failed to create collection config", err.Error())
		return
	}

	diags.Append(api.Wait(ctx, CollectionExists)...)
	return
}

func (api *collectionConfigAPI) Update(ctx context.Context) (diags diag.Diagnostics) {
	model, d := api.Model(ctx, "collection_config_update")
	if diags.Append(d...); diags.HasError() {
		return
	}

	body := url.Values{}

	for k, v := range model.Data["field_types"].(map[string]string) {
		body.Set(fmt.Sprintf("field.%s", k), v)
	}
	for k, v := range model.Data["accelerations"].([]string) {
		body.Set(fmt.Sprintf("accelerated_fields.accel_%03d", k), v)
	}

	model.Body = []byte(body.Encode())

	if _, err := model.Update(ctx); err != nil {
		diags.AddError("Failed to update collection config", err.Error())
	}
	return
}

func (api *collectionConfigAPI) Delete(ctx context.Context) (diags diag.Diagnostics) {
	model, d := api.Model(ctx, "collection_config")
	if diags.Append(d...); diags.HasError() {
		return
	}

	exists, d := api.CollectionExists(ctx, true)
	if diags.Append(d...); !exists || diags.HasError() {
		return
	}

	if _, err := model.Delete(ctx); err != nil {
		diags.AddError("Failed to delete collection config", err.Error())
		return
	}

	diags.Append(api.Wait(ctx, CollectionDoesNotExist)...)
	return
}

// collectionDataAPI client

type collectionDataAPI struct {
	*collectionAPI
	collectionDataModel
}

func NewCollectionDataAPI(m collectionDataModel, c models.ClientConfig) *collectionDataAPI {
	return &collectionDataAPI{NewCollectionAPI(m.Collection, c), m}
}

func (api *collectionDataAPI) Model(ctx context.Context, includeData bool) (model *models.CollectionApi, diags diag.Diagnostics) {
	data := map[string]interface{}{
		"collection_name": api.Collection.Name.ValueString(),
		"scope":           api.Scope.ValueString(),
		"generation":      api.Generation.ValueInt64(),
		"instance":        api.ID.ValueString(),
	}
	if includeData {
		model = api.collectionAPI.Model("collection_batchsave")
		var entries []collectionEntryModel
		if diags = api.Entries.ElementsAs(ctx, &entries, false); diags.HasError() {
			return
		}

		rows := make([]map[string]interface{}, len(entries))

		for i, entry := range entries {
			rowMap, diags_ := entry.Unpack()
			diags.Append(diags_...)
			rowMap["_instance"] = api.ID.ValueString()
			rowMap["_gen"] = api.Generation.ValueInt64()
			rowMap["_scope"] = api.Scope.ValueString()
			rowMap["_key"] = entry.ID.ValueString()
			rows[i] = rowMap
		}
		data["data"] = rows
		var err error
		model.Body, err = json.Marshal(rows)
		if err != nil {
			diags.AddError(fmt.Sprintf("Unable to marshal %s collection data", api.Key()), err.Error())
			return nil, diags
		}
	} else {
		model = api.collectionAPI.Model("collection_data")
	}
	model.Data = data
	return
}

func (api *collectionDataAPI) Save(ctx context.Context) (diags diag.Diagnostics) {
	if len(api.Entries.Elements()) == 0 {
		return
	}
	model, diags := api.Model(ctx, true)
	if diags.HasError() {
		return
	}
	_, err := model.Create(ctx)
	if err != nil {
		diags.AddError(fmt.Sprintf("Unable to save %s collection data", api.Key()), err.Error())
	}
	return
}

func (api *collectionDataAPI) deleteOldRows(ctx context.Context) (diags diag.Diagnostics) {
	model, diags := api.Model(ctx, false)
	if diags.HasError() {
		return
	}
	q := fmt.Sprintf(`{"$or":[{"_instance":null},{"_instance":{"$ne": "%s"}},{"_gen":null},{"_gen":{"$ne": %d}}]}`, api.ID.ValueString(), api.Generation.ValueInt64())
	q = fmt.Sprintf(`{"$and":[{"_scope":"%s"},%s]}`, api.Scope.ValueString(), q)
	model.Params = "query=" + url.QueryEscape(q)

	_, err := model.Delete(ctx)
	if err != nil {
		diags.AddError(fmt.Sprintf("Unable to delete %s collection data", api.Key()), err.Error())
	}
	return
}

func (api *collectionDataAPI) Read(ctx context.Context) (data []collectionEntryModel, diags diag.Diagnostics) {
	q := fmt.Sprintf(`{"_scope":"%s"}`, api.Scope.ValueString())
	obj, diags := api.Query(ctx, q, []string{}, 0)
	if diags.Append(diags...); diags.HasError() {
		return
	}

	arr, ok := obj.([]interface{})
	if !ok {
		diags.AddError(fmt.Sprintf("Unable to read %s collection data", api.Key()), "expected array body return type")
		return
	}

	data = make([]collectionEntryModel, len(arr))

	for i, item := range arr {
		var entry collectionEntryModel

		item_, ok := item.(map[string]interface{})
		if !ok {
			diags.AddError(fmt.Sprintf("Unable to read %s collection data", api.Key()), "expected map in array body return type")
		}
		row := map[string]interface{}{}

		for k, v := range item_ {
			if k == "_key" {
				entry.ID = types.StringValue(v.(string))
			} else if !strings.HasPrefix(k, "_") {
				switch val := v.(type) {
				case []interface{}:
					row[k] = val
				default:
					row[k] = []interface{}{val}
				}
			}
		}
		if diags.Append(entry.Pack(row)...); diags.HasError() {
			return
		}
		data[i] = entry
	}

	return
}

func (api *collectionDataAPI) Delete(ctx context.Context) (diags diag.Diagnostics) {
	model, diags_ := api.Model(ctx, false)
	if diags.Append(diags_...); diags.HasError() {
		return
	}
	model.Params = "query=" + url.QueryEscape(fmt.Sprintf(`{"_scope":"%s"}`, api.Scope.ValueString()))
	if _, err := model.Delete(ctx); err != nil {
		diags.AddError(fmt.Sprintf("Unable to delete %s collection data", api.Key()), err.Error())
	}
	return
}
