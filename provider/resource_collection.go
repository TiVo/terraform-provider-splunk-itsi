package provider

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

func validateStringIdentifier() schema.SchemaValidateFunc {
	return validation.All(
		validation.StringLenBetween(0, 255),
		validation.StringMatch(regexp.MustCompile(`^[a-zA-Z]`), "must begin with a letter"),
		validation.StringMatch(regexp.MustCompile(`^.[-_0-9a-zA-Z]+$`), "must contain only alphanumerics, hypens and underscore"))
}

func validateFieldTypes() schema.SchemaValidateFunc {
	arr := []string{"array", "number", "bool", "string", "cidr", "time"}
	return validation.StringInSlice(arr, true)
}

func ResourceCollection() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a KV store collection resource in Splunk.",
		CreateContext: collectionCreate,
		ReadContext:   collectionRead,
		UpdateContext: collectionUpdate,
		DeleteContext: collectionDelete,
		Importer: &schema.ResourceImporter{
			StateContext: collectionImport,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Name of the collection",
				ValidateFunc: validateStringIdentifier(),
			},
			"field_types": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateFieldTypes(),
				},
				Description: "Field type information",
			},
			"accelerations": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsJSON,
				},
				Description: "Field acceleration information (see Splunk docs for accelerated_fields in collections.conf)",
			},
		},
	}
}

func ResourceCollectionEntry() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a KV store collection record.",
		CreateContext: collectionEntryCreate,
		ReadContext:   collectionEntryRead,
		UpdateContext: collectionEntryUpdate,
		DeleteContext: collectionEntryDelete,
		Importer: &schema.ResourceImporter{
			StateContext: collectionEntryImport,
		},
		Schema: map[string]*schema.Schema{
			"collection_name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Name of the collection containing this entry",
				ValidateFunc: validateStringIdentifier(),
			},
			"key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Value of the key field of this collection entry",
				ForceNew:    true,
			},
			"data": {
				Type:     schema.TypeMap,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Collection entry data, fields and their values",
			},
		},
	}
}

func ResourceCollectionEntries() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages KV store collection records.",
		CreateContext: collectionEntriesCreate,
		ReadContext:   collectionEntriesRead,
		UpdateContext: collectionEntriesUpdate,
		DeleteContext: collectionEntriesDelete,
		Importer: &schema.ResourceImporter{
			StateContext: collectionEntriesImport,
		},
		Schema: map[string]*schema.Schema{
			"collection_name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Name of the collection containing this entry",
				ValidateFunc: validateStringIdentifier(),
			},
			"preserve_keys": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Should the _key field be preserved?",
			},
			"instance": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Computed instance ID of the resource, used w/ 'generation' to prevent row duplication in a given scope",
			},
			"generation": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Computed latest generation of changes",
			},
			"scope": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "default_scope",
				Description: "Scope of ownership of this collection entry",
			},
			"data": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeSet,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"name": {
								Type:        schema.TypeString,
								Required:    true,
								Description: "Name of the field",
							},
							"values": {
								Type:        schema.TypeList,
								Required:    true,
								Description: "Values of the field",
								Elem: &schema.Schema{
									Type: schema.TypeString,
								},
							},
						},
					},
				},
				Description: "Collection data for all entries",
			},
		},
	}
}

// ----------------------------------------------------------------------

// Helper functions in the spirit of diag.Errorf()...
func errorf(summary string) diag.Diagnostic {
	return diag.Diagnostic{Severity: diag.Error, Summary: summary}
}
func warnf(summary string) diag.Diagnostic {
	return diag.Diagnostic{Severity: diag.Warning, Summary: summary}
}

func unpackRow(in []interface{}) (out map[string]interface{}) {
	out = make(map[string]interface{})
	for _, f := range in {
		m := f.(map[string]interface{})

		multiValue, ok := m["values"].([]interface{})
		if ok && len(multiValue) > 0 {
			if len(multiValue) == 1 {
				out[m["name"].(string)] = multiValue[0]
			} else {
				out[m["name"].(string)] = multiValue
			}
		}
	}
	return
}

// ----------------------------------------------------------------------

func getCollectionName(d *schema.ResourceData) string {
	if name, ok := d.GetOk("name"); ok {
		return name.(string)
	}
	if name, ok := d.GetOk("collection_name"); ok {
		return name.(string)
	}
	panic("could not find collection name in resource")
}

// Create a "collection API model" that can help us query APIs,
// passing no body in the query...
func collection(ctx context.Context, d *schema.ResourceData, object_type string, m interface{}) (config *models.CollectionApi, err error) {
	clientConfig := m.(models.ClientConfig)
	name := getCollectionName(d)
	c := models.NewCollection(clientConfig, name, name, object_type)

	data := make(map[string]interface{})
	data["name"] = name
	if field_types, ok := d.GetOk("field_types"); ok {
		if data["field_types"], err = unpackResourceMap[string](field_types.(map[string]interface{})); err != nil {
			return
		}
	} else {
		data["field_types"] = make(map[string]string)
	}
	if accelerations, ok := d.GetOk("accelerations"); ok {
		if data["accelerations"], err = unpackResourceList[string](accelerations.([]interface{})); err != nil {
			return
		}
	} else {
		data["accelerations"] = []string{}
	}
	c.Data = data

	tflog.Trace(ctx, "RSRC COLLECTION:     api model ("+object_type+")",
		map[string]interface{}{"key": c.RESTKey, "data": data})
	return c, nil
}

// Create a "collection API model" that can help us query APIs,
// passing no body in the query...
func collectionEntry(ctx context.Context, d *schema.ResourceData, object_type string, m interface{}) (config *models.CollectionApi, err error) {
	clientConfig := m.(models.ClientConfig)
	collection := d.Get("collection_name").(string)
	key := d.Get("key").(string)
	if key == "" {
		key = d.Id()
	}
	c := models.NewCollection(clientConfig, collection, key, object_type)

	data := make(map[string]interface{})
	data["collection"] = collection
	data["key"] = key

	dataMap, err := unpackResourceMap[string](d.Get("data").(map[string]interface{}))
	if err != nil {
		return
	}
	dataMap["_key"] = data["key"].(string)
	tflog.Trace(ctx, "RSRC COLLECTION:     data", map[string]interface{}{"map": dataMap})

	data["data"] = dataMap
	c.Data = data

	tflog.Trace(ctx, "RSRC COLLECTION:     api model ("+object_type+")",
		map[string]interface{}{"key": c.RESTKey, "data": data})
	return c, nil
}

// Create a "collection API model" that can help us query APIs,
// passing no body in the query...
func collectionEntries(ctx context.Context, d *schema.ResourceData, object_type string, m interface{}) (config *models.CollectionApi, err error) {
	clientConfig := m.(models.ClientConfig)
	name := d.Get("collection_name").(string)
	c := models.NewCollection(clientConfig, name, name, object_type)

	data := make(map[string]interface{})
	data["instance"] = d.Get("instance").(string)
	data["collection_name"] = d.Get("collection_name").(string)
	data["preserve_keys"] = d.Get("preserve_keys").(bool)
	data["generation"] = d.Get("generation").(int)
	data["scope"] = d.Get("scope").(string)
	c.Data = data

	tflog.Trace(ctx, "RSRC COLLECTION:     api model ("+object_type+")",
		map[string]interface{}{"key": c.RESTKey, "data": data})
	return c, nil
}

// Create a "collection API model" that can help us query APIs,
// passing a default body structure in the query...
func collectionEntriesDataBody(ctx context.Context, d *schema.ResourceData, object_type string, m interface{}) (config *models.CollectionApi, err error) {
	clientConfig := m.(models.ClientConfig)
	name := d.Get("collection_name").(string)
	c := models.NewCollection(clientConfig, name, name, object_type)

	data := make(map[string]interface{})
	instance := d.Get("instance").(string)
	data["instance"] = instance
	data["collection_name"] = d.Get("collection_name").(string)
	data["preserve_keys"] = d.Get("preserve_keys").(bool)
	gen := d.Get("generation").(int)
	data["generation"] = gen
	scope := d.Get("scope").(string)
	data["scope"] = scope

	dataRes := d.Get("data").([]interface{})
	dataList := make([]map[string]interface{}, 0, len(dataRes))

	tflog.Trace(ctx, "RSRC COLLECTION:     save", map[string]interface{}{"arr": dataRes})
	for i, row := range dataRes {
		rowMap := unpackRow(row.(*schema.Set).List())
		rowMap["_instance"] = instance
		// Add ordering information so that we can reorders
		// after rereading from Splunk...s
		rowMap["_idx"] = fmt.Sprintf("%d", i)
		// Add generational information so that we can delete
		// out of date entries efficiently...
		rowMap["_gen"] = gen
		// Add scope information so that we can manage a
		// subset of the entries...
		rowMap["_scope"] = scope
		tflog.Trace(ctx, "RSRC COLLECTION:       item", map[string]interface{}{"item": rowMap})
		dataList = append(dataList, rowMap)
	}
	data["data"] = dataList

	c.Data = data

	tflog.Trace(ctx, "RSRC COLLECTION:   api model ("+object_type+")",
		map[string]interface{}{"key": c.RESTKey, "err": err})
	return c, nil
}

// ----------------------------------------------------------------------
// Basic API Calls
// ----------------------------------------------------------------------

func collectionApiExists(ctx context.Context, d *schema.ResourceData, m interface{}) (api *models.CollectionApi, err error) {
	tflog.Info(ctx, "RSRC COLLECTION:   exists ("+getCollectionName(d)+")")
	// To check if a collection exists, we use...
	//   storage/collections/config/{collection} -- GET
	api, err = collection(ctx, d, "collection_config_no_body", m)
	if err != nil {
		return nil, err
	}
	return api.Read(ctx, true)
}

// ----------------------------------------------------------------------

func collectionApiCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (api *models.CollectionApi, err error) {
	tflog.Info(ctx, "RSRC COLLECTION:   create ("+d.Get("name").(string)+")")
	// To create a collection, we use...
	//   storage/collections/config -- POST (body: name=<collection>)
	api, _ = collection(ctx, d, "collection_config_keyless", m)

	body := url.Values{}
	body.Set("name", api.RESTKey)
	api.Body = []byte(body.Encode())

	return api.Create(ctx)
}

func collectionApiRead(ctx context.Context, d *schema.ResourceData, m interface{}) (api *models.CollectionApi, err error) {
	tflog.Info(ctx, "RSRC COLLECTION:   read ("+d.Get("name").(string)+")")
	// To read a collection, we use...
	//   storage/collections/config/{collection} -- GET
	api, _ = collection(ctx, d, "collection_config", m)
	return api.Read(ctx)
}

func collectionApiUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (api *models.CollectionApi, err error) {
	tflog.Info(ctx, "RSRC COLLECTION:   update ("+d.Get("name").(string)+")")
	// To update a collection, we use...
	//   storage/collections/config/{collection} -- POST
	api, _ = collection(ctx, d, "collection_config", m)

	var qft []string
	for k, v := range api.Data["field_types"].(map[string]string) {
		qft = append(qft, fmt.Sprintf("field.%[1]s=%[2]s", k, url.QueryEscape(v)))
	}
	var qaf []string
	for i, s := range api.Data["accelerations"].([]string) {
		qaf = append(qaf, fmt.Sprintf("accelerated_fields.accel_%03d=%s", i, url.QueryEscape(s)))
	}
	api.Params = strings.Join(qft, "&") + "&" + strings.Join(qaf, "&")

	return api.Update(ctx)
}

func collectionApiDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (api *models.CollectionApi, err error) {
	tflog.Info(ctx, "RSRC COLLECTION:   delete ("+d.Get("name").(string)+")")
	// To delete a collection, we use...
	//   storage/collections/config/{collection} -- DELETE
	api, _ = collection(ctx, d, "collection_config", m)
	return api.Delete(ctx)
}

// ----------------------------------------------------------------------

func collectionApiEntryExists(ctx context.Context, d *schema.ResourceData, m interface{}) (api *models.CollectionApi, err error) {
	tflog.Info(ctx, "RSRC COLLECTION:   entry exists ("+d.Get("key").(string)+")")
	if d.Get("key").(string) == "" {
		return
	}
	// To check if a collection entry exists, we use...
	//   storage/collections/config/{collection}/{key} -- GET
	api, _ = collectionEntry(ctx, d, "collection_entry_no_body", m)
	return api.Read(ctx, true)
}

// ----------------------------------------------------------------------

func collectionApiEntryCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (api *models.CollectionApi, err error) {
	tflog.Info(ctx, "RSRC COLLECTION:   entry create ("+d.Get("key").(string)+")")
	// To create a collection entry, we use...
	//   storage/collections/data/{collection} -- POST
	api, _ = collectionEntry(ctx, d, "collection_entry_keyless", m)
	if api.Body, err = api.Marshal(api.Data["data"]); err != nil {
		return nil, err
	}
	return api.Create(ctx)
}

func collectionApiEntryRead(ctx context.Context, d *schema.ResourceData, m interface{}) (api *models.CollectionApi, err error) {
	tflog.Info(ctx, "RSRC COLLECTION:   entry read ("+d.Get("key").(string)+")")
	// To read a collection entry, we use...
	//   storage/collections/data/{collection}/{key} -- GET
	api, _ = collectionEntry(ctx, d, "collection_entry", m)
	return api.Read(ctx)
}

func collectionApiEntryUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (api *models.CollectionApi, err error) {
	tflog.Info(ctx, "RSRC COLLECTION:   entry update ("+d.Get("key").(string)+")")
	// To create a collection, we use...
	//   storage/collections/data/{collection}/{key} -- POST
	api, _ = collectionEntry(ctx, d, "collection_entry", m)
	if api.Body, err = api.Marshal(api.Data["data"]); err != nil {
		return nil, err
	}
	return api.Update(ctx)
}

func collectionApiEntryDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (api *models.CollectionApi, err error) {
	tflog.Info(ctx, "RSRC COLLECTION:   entry delete ("+d.Get("key").(string)+")")
	// To delete a collection, we use...
	//   storage/collections/config/{collection} -- DELETE
	api, _ = collectionEntry(ctx, d, "collection_entry", m)
	return api.Delete(ctx)
}

// ----------------------------------------------------------------------

func collectionApiEntriesRead(ctx context.Context, d *schema.ResourceData, m interface{}) (api *models.CollectionApi, err error) {
	tflog.Info(ctx, "RSRC COLLECTION:   entries read ("+d.Get("collection_name").(string)+")")
	// To read the contents of a collection, we use...
	//   storage/collections/data/{collection} -- GET
	api, _ = collectionEntries(ctx, d, "collection_data", m)

	scope := api.Data["scope"].(string)
	q := fmt.Sprintf("{\"_scope\":\"%s\"}", scope)
	api.Params = "query=" + url.QueryEscape(q)

	return api.Read(ctx)
}

func collectionApiEntriesSave(ctx context.Context, d *schema.ResourceData, m interface{}) (api *models.CollectionApi, err error) {
	tflog.Info(ctx, "RSRC COLLECTION:   entries save ("+d.Get("collection_name").(string)+")")
	// To update entries in a collection, we use...
	//   storage/collections/data/{collection}/batch_save -- POST (body: <row data>)

	instance := d.Get("instance")
	if instance == "" {
		if instance, err = uuid.GenerateUUID(); err != nil {
			return
		}
		if err = d.Set("instance", instance); err != nil {
			return
		}
	}

	// Increment the generation counter...
	gen := d.Get("generation").(int) + 1
	if err := d.Set("generation", gen); err != nil {
		return nil, err
	}

	api, _ = collectionEntriesDataBody(ctx, d, "collection_batchsave", m)
	if api.Body, err = api.Marshal(api.Data["data"]); err != nil {
		return nil, err
	}
	return api.Create(ctx)
}

func collectionApiEntriesDeleteAllRows(ctx context.Context, d *schema.ResourceData, m interface{}) (api *models.CollectionApi, err error) {
	tflog.Info(ctx, "RSRC COLLECTION:   entries delete all rows ("+d.Get("collection_name").(string)+")")
	// To delete all entries in a collection, we use...
	//   storage/collections/data/{collection} -- DELETE
	api, _ = collectionEntries(ctx, d, "collection_data", m)

	scope := api.Data["scope"].(string)
	q := fmt.Sprintf("{\"_scope\":\"%s\"}", scope)
	api.Params = "query=" + url.QueryEscape(q)

	return api.Delete(ctx)
}

func collectionApiEntriesDeleteOldRows(ctx context.Context, d *schema.ResourceData, m interface{}) (api *models.CollectionApi, err error) {
	tflog.Info(ctx, "RSRC COLLECTION:   entries delete old rows ("+d.Get("collection_name").(string)+")")
	// To delete entries not matching our keys in a collection, we use...
	//   storage/collections/data/{collection} -- DELETE
	api, _ = collectionEntries(ctx, d, "collection_data", m)
	instance := api.Data["instance"].(string)
	gen := api.Data["generation"].(int)
	scope := api.Data["scope"].(string)
	q := fmt.Sprintf("{\"$or\":[{\"_instance\":null},{\"_instance\":{\"$ne\": \"%s\"}},{\"_gen\":null},{\"_gen\":{\"$ne\": %d}}]}", instance, gen)
	q = fmt.Sprintf("{\"$and\":[{\"_scope\":\"%s\"},%s]}", scope, q)
	api.Params = "query=" + url.QueryEscape(q)

	return api.Delete(ctx)
}

// ----------------------------------------------------------------------
// Collection Resource
// ----------------------------------------------------------------------

func collectionCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	tflog.Info(ctx, "RSRC COLLECTION: CREATE", map[string]interface{}{"d": fmt.Sprintf("%+v", d)})

	existing, err := collectionApiExists(ctx, d, m)
	if err != nil {
		diags = append(diags, warnf("Unable to check existence of collection: "+err.Error()))
		return
	}
	if existing == nil {
		// Create the collection...
		if _, err := collectionApiCreate(ctx, d, m); err != nil {
			diags = append(diags, warnf("Unable to create collection: "+err.Error()))
			return
		}
	}

	// Update the collection configuration...
	if _, err := collectionApiUpdate(ctx, d, m); err != nil {
		diags = append(diags, warnf("Unable to update collection: "+err.Error()))
		return
	}

	// Now read everything back to see what we made...
	c, err := collectionApiRead(ctx, d, m)
	if err != nil {
		diags = append(diags, errorf("Unable to read back collection: "+err.Error()))
		return
	}
	if c == nil {
		diags = append(diags, errorf("Unable to create object!"))
		return
	}
	if err := populateCollectionResourceData(ctx, c, d); err != nil {
		diags = append(diags, warnf("Unable to populate resource: "+err.Error()))
	}
	return
}

func collectionRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	tflog.Info(ctx, "RSRC COLLECTION: READ", map[string]interface{}{"d": fmt.Sprintf("%+v", d)})

	c, err := collectionApiRead(ctx, d, m)
	if err != nil {
		diags = append(diags, errorf("Unable to read/dump collection model: "+err.Error()))
		return
	}
	if c == nil || c.Data == nil {
		d.SetId("")
		return nil
	}
	if err := populateCollectionResourceData(ctx, c, d); err != nil {
		diags = append(diags, errorf("Unable to populate resource: "+err.Error()))
	}
	return
}

func collectionUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	tflog.Info(ctx, "RSRC COLLECTION: UPDATE", map[string]interface{}{"d": fmt.Sprintf("%+v", d)})

	// If this collection doesn't exist, create it...
	existing, err := collectionApiExists(ctx, d, m)
	if err != nil {
		diags = append(diags, warnf("Unable to read/dump collection model: "+err.Error()))
	}
	if existing == nil {
		// Create the collection...
		if _, err := collectionApiCreate(ctx, d, m); err != nil {
			diags = append(diags, warnf("Unable to create collection: "+err.Error()))
			return
		}
	}

	// Update the collection configuration...
	if _, err := collectionApiUpdate(ctx, d, m); err != nil {
		diags = append(diags, warnf("Unable to update collection: "+err.Error()))
		return
	}

	// Now read everything back to see what we made...
	c, err := collectionApiRead(ctx, d, m)
	if err != nil {
		diags = append(diags, errorf("Unable to read back collection: "+err.Error()))
		return
	}
	if c == nil {
		diags = append(diags, errorf("Unable to update object, it is missing!"))
		return
	}
	if err := populateCollectionResourceData(ctx, c, d); err != nil {
		diags = append(diags, warnf("Unable to populate resource: "+err.Error()))
	}
	return
}

func collectionDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	tflog.Info(ctx, "RSRC COLLECTION: DELETE", map[string]interface{}{"d": fmt.Sprintf("%+v", d)})

	if _, err := collectionApiDelete(ctx, d, m); err != nil {
		diags = append(diags, errorf("Unable to delete collection: "+err.Error()))
	}
	return
}

func collectionImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	tflog.Info(ctx, "RSRC COLLECTION: IMPORT", map[string]interface{}{"d": fmt.Sprintf("%+v", d)})
	c, err := collectionApiRead(ctx, d, m)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}

	if err = populateCollectionResourceData(ctx, c, d); err != nil {
		return nil, err
	}
	if d.Id() == "" {
		return nil, nil
	}
	return []*schema.ResourceData{d}, nil
}

func populateCollectionResourceData(ctx context.Context, c *models.CollectionApi, d *schema.ResourceData) (err error) {
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
		return err
	}
	keys := feed.Content[0].Dicts[0].Keys
	tflog.Trace(ctx, "RSRC COLLECTION:   content",
		map[string]interface{}{"num": len(keys), "keys": fmt.Sprintf("%+v", keys)})

	// Iterate through the feed finding field_types and
	// accelerations...
	field_types := map[string]string{}
	accelerations := make([]string, 0, len(keys))
	for _, key := range keys {
		if strings.HasPrefix(key.Name, "field.") {
			k := strings.TrimPrefix(key.Name, "field.")
			field_types[k] = key.Value
		} else if strings.HasPrefix(key.Name, "accelerated_fields.accel_") {
			s := strings.TrimPrefix(key.Name, "accelerated_fields.accel_")
			if i, err := strconv.Atoi(s); err == nil && i < cap(keys) {
				accelerations = accelerations[0 : i+1]
				accelerations[i] = key.Value
			}
		}
	}

	c.Data["field_types"] = field_types
	c.Data["accelerations"] = accelerations
	tflog.Trace(ctx, "RSRC COLLECTION:   field types", map[string]interface{}{"field_types": field_types})
	tflog.Trace(ctx, "RSRC COLLECTION:   accelerations", map[string]interface{}{"accelerations": accelerations})

	if len(field_types) > 0 {
		if err = d.Set("field_types", c.Data["field_types"]); err != nil {
			return err
		}
	}
	if len(accelerations) > 0 {
		if err = d.Set("accelerations", c.Data["accelerations"]); err != nil {
			return err
		}
	}
	if err = d.Set("name", c.Data["name"]); err != nil {
		return err
	}
	tflog.Debug(ctx, "RSRC COLLECTION:   populate", map[string]interface{}{"data": c.Data})

	d.SetId(c.RESTKey)
	return nil
}

func collectionCheck(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	collection := getCollectionName(d)
	tflog.Info(ctx, "RSRC COLLECTION: CHECK", map[string]interface{}{"collection": collection})

	existing, err := collectionApiExists(ctx, d, m)
	if err != nil {
		diags = append(diags, warnf("Unable to read collection entry model: "+err.Error()))
		return
	}
	if existing == nil {
		s := fmt.Sprintf("Referenced collection (%s) does not exist; you should reference a collection resource or create it manually", collection)
		diags = append(diags, warnf(s))
	}
	return
}

// ----------------------------------------------------------------------
// Collection Entry Resource
// ----------------------------------------------------------------------

func collectionEntryCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	tflog.Info(ctx, "RSRC COLLECTION: ENTRY CREATE", map[string]interface{}{"d": fmt.Sprintf("%+v", d)})
	if diags = collectionCheck(ctx, d, m); diags != nil {
		return
	}

	var c *models.CollectionApi
	existing, err := collectionApiEntryExists(ctx, d, m)
	if err != nil {
		diags = append(diags, errorf("Unable to read collection entry model: "+err.Error()))
	}
	if existing == nil {
		if c, err = collectionApiEntryCreate(ctx, d, m); err != nil {
			diags = append(diags, errorf("Unable to create entry: "+err.Error()))
		}
	} else {
		if c, err = collectionApiEntryUpdate(ctx, d, m); err != nil {
			diags = append(diags, errorf("Unable to update entry: "+err.Error()))
		}
	}

	if err := populateCollectionEntryResourceData(ctx, c, d); err != nil {
		diags = append(diags, errorf("Unable to populate entry resource: "+err.Error()))
	}
	return
}

func collectionEntryRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	tflog.Info(ctx, "RSRC COLLECTION: ENTRY READ", map[string]interface{}{"d": fmt.Sprintf("%+v", d)})
	if diags = collectionCheck(ctx, d, m); diags != nil {
		return
	}

	c, err := collectionApiEntryRead(ctx, d, m)
	if err != nil {
		diags = append(diags, errorf("Unable to read/dump entry model: "+err.Error()))
		return
	}
	if c == nil || c.Data == nil {
		d.SetId("")
		return nil
	}
	if err := populateCollectionEntryResourceData(ctx, c, d); err != nil {
		diags = append(diags, errorf("Unable to populate entry resource: "+err.Error()))
	}
	return
}

func collectionEntryUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	tflog.Info(ctx, "RSRC COLLECTION: ENTRY UPDATE", map[string]interface{}{"d": fmt.Sprintf("%+v", d)})
	if diags = collectionCheck(ctx, d, m); diags != nil {
		return
	}

	existing, err := collectionApiEntryExists(ctx, d, m)
	if err != nil {
		diags = append(diags, errorf("Unable to read collection entry model: "+err.Error()))
	}
	if existing == nil {
		if _, err := collectionApiEntryCreate(ctx, d, m); err != nil {
			diags = append(diags, errorf("Unable to create entry: "+err.Error()))
		}
	} else {
		if _, err := collectionApiEntryUpdate(ctx, d, m); err != nil {
			diags = append(diags, errorf("Unable to update entry: "+err.Error()))
		}
	}

	// Now read everything back to see what we made...
	c, err := collectionApiEntryRead(ctx, d, m)
	if err != nil {
		diags = append(diags, errorf("Unable to read back entry: "+err.Error()))
		return
	}
	if c == nil {
		diags = append(diags, errorf("Unable to update object, it is missing!"))
		return
	}
	if err := populateCollectionEntryResourceData(ctx, c, d); err != nil {
		diags = append(diags, errorf("Unable to populate entry resource: "+err.Error()))
	}
	return
}

func collectionEntryDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	tflog.Info(ctx, "RSRC COLLECTION: ENTRY DELETE", map[string]interface{}{"d": fmt.Sprintf("%+v", d)})
	if diags = collectionCheck(ctx, d, m); diags != nil {
		return
	}

	if _, err := collectionApiEntryDelete(ctx, d, m); err != nil {
		diags = append(diags, errorf("Unable to delete entry: "+err.Error()))
	}
	return
}

func collectionEntryImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	tflog.Info(ctx, "RSRC COLLECTION: ENTRY IMPORT", map[string]interface{}{"d": fmt.Sprintf("%+v", d)})

	c, err := collectionApiEntryRead(ctx, d, m)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}

	if err = populateCollectionEntryResourceData(ctx, c, d); err != nil {
		return nil, err
	}
	if d.Id() == "" {
		return nil, nil
	}
	return []*schema.ResourceData{d}, nil
}

func populateCollectionEntryResourceData(ctx context.Context, c *models.CollectionApi, d *schema.ResourceData) (err error) {
	var obj interface{}
	if obj, err = c.Unmarshal(c.Body); err != nil {
		return err
	}
	dataRes, ok := obj.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected map body return type: %s", string(c.Body))
	}

	tflog.Trace(ctx, "RSRC COLLECTION:   data", map[string]interface{}{"map": dataRes})
	var key string
	dataMap := make(map[string]string)
	for k, v := range dataRes {
		// Do not include internal fields...
		if k[0] != '_' {
			dataMap[k] = fmt.Sprint(v)
		}
		if k == "_key" {
			key = fmt.Sprint(v)
		}
	}

	if c.Data == nil {
		c.Data = make(map[string]interface{})
	}

	c.Data["key"] = key
	c.Data["data"] = dataMap

	if err = d.Set("data", c.Data["data"]); err != nil {
		return err
	}
	tflog.Debug(ctx, "RSRC COLLECTION:   populate", map[string]interface{}{"data": c.Data})

	d.SetId(c.RESTKey)
	return nil
}

// ----------------------------------------------------------------------
// Collection Entries Resource
// ----------------------------------------------------------------------

func collectionEntriesCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	tflog.Info(ctx, "RSRC COLLECTION: ENTRIES CREATE", map[string]interface{}{"d": fmt.Sprintf("%+v", d)})
	if diags = collectionCheck(ctx, d, m); diags != nil {
		return
	}

	// Next push the data into the collection...
	if _, err := collectionApiEntriesSave(ctx, d, m); err != nil {
		diags = append(diags, warnf("Unable to populate collection: "+err.Error()))
	}

	// Delete all of the excess entries...
	if _, err := collectionApiEntriesDeleteOldRows(ctx, d, m); err != nil {
		diags = append(diags, errorf("Unable to delete excess rows in collection: "+err.Error()))
	}

	// Now read everything back to see what we made...
	c, err := collectionApiEntriesRead(ctx, d, m)
	if err != nil {
		diags = append(diags, errorf("Unable to read back collection: "+err.Error()))
		return
	}
	if c == nil {
		diags = append(diags, errorf("Unable to create object!"))
		return
	}
	if err := populateCollectionEntriesResourceData(ctx, c, d); err != nil {
		diags = append(diags, warnf("Unable to populate resource: "+err.Error()))
	}
	return
}

func collectionEntriesRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	tflog.Info(ctx, "RSRC COLLECTION: ENTRIES READ", map[string]interface{}{"d": fmt.Sprintf("%+v", d)})
	if diags = collectionCheck(ctx, d, m); diags != nil {
		return
	}

	c, err := collectionApiEntriesRead(ctx, d, m)
	if err != nil {
		diags = append(diags, errorf("Unable to read/dump collection model: "+err.Error()))
		return
	}
	if c == nil || c.Data == nil {
		d.SetId("")
		return nil
	}
	if err := populateCollectionEntriesResourceData(ctx, c, d); err != nil {
		diags = append(diags, errorf("Unable to populate resource: "+err.Error()))
	}
	return
}

func collectionEntriesUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	tflog.Info(ctx, "RSRC COLLECTION: ENTRIES UPDATE", map[string]interface{}{"d": fmt.Sprintf("%+v", d)})
	if diags = collectionCheck(ctx, d, m); diags != nil {
		return
	}

	// Next push the data into the collection...
	if _, err := collectionApiEntriesSave(ctx, d, m); err != nil {
		diags = append(diags, errorf("Unable to populate collection: "+err.Error()))
	}

	// Delete all of the excess entries...
	if _, err := collectionApiEntriesDeleteOldRows(ctx, d, m); err != nil {
		diags = append(diags, errorf("Unable to delete excess rows in collection: "+err.Error()))
	}

	// Now read everything back to see what we made...
	c, err := collectionApiEntriesRead(ctx, d, m)
	if err != nil {
		diags = append(diags, errorf("Unable to read back collection: "+err.Error()))
		return
	}
	if c == nil {
		diags = append(diags, errorf("Unable to update object, it is missing!"))
		return
	}
	if err := populateCollectionEntriesResourceData(ctx, c, d); err != nil {
		diags = append(diags, warnf("Unable to populate resource: "+err.Error()))
	}
	return
}

func collectionEntriesDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	tflog.Info(ctx, "RSRC COLLECTION: ENTRIES DELETE", map[string]interface{}{"d": fmt.Sprintf("%+v", d)})
	if diags = collectionCheck(ctx, d, m); diags != nil {
		return
	}

	if _, err := collectionApiEntriesDeleteAllRows(ctx, d, m); err != nil {
		diags = append(diags, errorf("Unable to delete collection: "+err.Error()))
	}
	return
}

func collectionEntriesImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	tflog.Info(ctx, "RSRC COLLECTION: ENTRIES IMPORT", map[string]interface{}{"d": fmt.Sprintf("%+v", d)})

	c, err := collectionApiEntriesRead(ctx, d, m)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}

	if err = populateCollectionEntriesResourceData(ctx, c, d); err != nil {
		return nil, err
	}
	if d.Id() == "" {
		return nil, nil
	}
	return []*schema.ResourceData{d}, nil
}

func populateCollectionEntriesResourceData(ctx context.Context, c *models.CollectionApi, d *schema.ResourceData) (err error) {
	var obj interface{}
	if obj, err = c.Unmarshal(c.Body); err != nil {
		return err
	}
	arr, ok := obj.([]interface{})
	if !ok {
		return fmt.Errorf("expected array body return type")
	}

	tflog.Trace(ctx, "RSRC COLLECTION:   populate", map[string]interface{}{"arr": arr})
	preserve_keys, _ := c.Data["preserve_keys"].(bool)
	data := make([][]map[string]interface{}, 0, len(arr))

	for _, item := range arr {
		item_, ok := item.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected map in array body return type")
		}
		row := []map[string]interface{}{}
		for k, v := range item_ {
			// Perhaps do not include internal fields...
			if k[0] != '_' || k == "_idx" || (k == "_key" && preserve_keys) {
				m := map[string]interface{}{"name": k}
				if singleValue, ok := v.(string); ok {
					m["values"] = []string{singleValue}
				} else if multiValue, ok := v.([]interface{}); ok {
					m["values"] = multiValue
				} else {
					return fmt.Errorf("invalid collection value %#v", v)
				}

				row = append(row, m)
			}
		}
		tflog.Trace(ctx, "RSRC COLLECTION:     item", map[string]interface{}{"item": item_, "row": row})
		data = append(data, row)
	}

	if c.Data == nil {
		c.Data = make(map[string]interface{})
	}
	c.Data["data"] = data

	if err = d.Set("collection_name", c.Data["collection_name"]); err != nil {
		return err
	}
	if err = d.Set("preserve_keys", c.Data["preserve_keys"]); err != nil {
		return err
	}
	if err = d.Set("generation", c.Data["generation"]); err != nil {
		return err
	}
	if err = d.Set("scope", c.Data["scope"]); err != nil {
		return err
	}
	tflog.Debug(ctx, "RSRC COLLECTION:   populate", map[string]interface{}{"data": data})

	// Reorder the data by our saved index ordering...
	rowIndex, idxPos := make([]int, len(data)), make([]int, len(data))
	for i, row := range data {
		for j, item := range row {
			if item["name"] == "_idx" {
				idx, err := strconv.Atoi(item["values"].([]string)[0])
				if err != nil {
					return err
				}
				rowIndex[i] = idx
				idxPos[i] = j
				break
			}
		}
	}

	//Now remove the artificial "_idx" field...
	for i := 0; i < len(data); i++ {
		data[i] = append(data[i][:idxPos[i]], data[i][idxPos[i]+1:]...)
	}

	sort.SliceStable(data, func(i, j int) bool {
		return rowIndex[i] < rowIndex[j]
	})

	if err = d.Set("data", data); err != nil {
		return err
	}

	d.SetId(c.RESTKey)
	return nil
}
