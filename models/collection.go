package models

// Models and support for interacting with Splunk's collection-related
// REST APIs.

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gopkg.in/yaml.v3"
)

func init() {
	err := yaml.Unmarshal([]byte(collectionApiConfigs), &CollectionApiConfigs)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
}

type collectionApiConfig struct {
	Path                  string `yaml:"path"`
	ApiCollectionKeyInUrl bool   `yaml:"api_collection_key_in_url"`
	ApiKeyInUrl           bool   `yaml:"api_key_in_url"`
	PathExtension         string `yaml:"path_extension"`
	ContentType           string `yaml:"content_type"`
	BodyFormat            string `yaml:"body_format"`
	ApiDoNotSendBody      bool   `yaml:"api_do_not_send_body"`
	ApiIgnoreResponseBody bool   `yaml:"api_ignore_response_body"`
	RestKeyField          string `yaml:"rest_key_field"`
	TFIDField             string `yaml:"tfid_field"`
}

var CollectionApiConfigs map[string]collectionApiConfig

const (
	collectionApiConfigs = `
collection_config_keyless:
    path:                            storage/collections/config
    api_key_in_url:                  false
    body_format:                     XML
    api_ignore_response_body:        true
    rest_key_field:                  name
    tfid_field:                      name

collection_config_keyless_with_body:
    path:                            storage/collections/config
    api_key_in_url:                  false
    body_format:                     XML
    api_ignore_response_body:        false
    rest_key_field:                  name
    tfid_field:                      name

collection_config:
    path:                            storage/collections/config
    api_key_in_url:                  true
    body_format:                     XML
    api_do_not_send_body:            true
    rest_key_field:                  name
    tfid_field:                      name

collection_config_update:
    path:                            storage/collections/config
    api_key_in_url:                  true
    body_format:                     XML
    api_ignore_response_body:        true
    rest_key_field:                  name
    tfid_field:                      name

collection_config_no_body:
    path:                            storage/collections/config
    api_key_in_url:                  true
    body_format:                     XML
    api_do_not_send_body:            true
    api_ignore_response_body:        true
    rest_key_field:                  name
    tfid_field:                      name

collection_entry_keyless:
    path:                            storage/collections/data
    api_collection_key_in_url:       true
    api_key_in_url:                  false
    body_format:                     JSON
    rest_key_field:                  _key
    tfid_field:                      key

collection_entry:
    path:                            storage/collections/data
    api_collection_key_in_url:       true
    api_key_in_url:                  true
    body_format:                     JSON
    rest_key_field:                  _key
    tfid_field:                      key

collection_entry_no_body:
    path:                            storage/collections/data
    api_collection_key_in_url:       true
    api_key_in_url:                  true
    body_format:                     JSON
    api_ignore_response_body:        true
    rest_key_field:                  _key
    tfid_field:                      key

collection_data:
    path:                            storage/collections/data
    api_key_in_url:                  true
    body_format:                     JSON
    rest_key_field:                  name
    tfid_field:                      name

collection_batchsave:
    path:                            storage/collections/data
    api_key_in_url:                  true
    path_extension:                  batch_save
    body_format:                     JSON
    api_ignore_response_body:        true
    rest_key_field:                  name
    tfid_field:                      name

collection_batchfind:
    path:                            storage/collections/data
    api_key_in_url:                  true
    path_extension:                  batch_find
    body_format:                     JSON
    rest_key_field:                  name
    tfid_field:                      name
`
)

type CollectionApi struct {
	base       Base
	RESTKey    string // key used to collect this resource via the REST API
	apiConfig  collectionApiConfig
	Collection string                 // Collection name
	App        string                 // Collection App
	Owner      string                 // Collection owner
	Data       map[string]interface{} // Data for this object
	Params     string                 // URL query string, iff provided
	Body       []byte                 // Body used for this API call
}

func NewCollection(clientConfig ClientConfig, collection, app, owner, key, objectType string) *CollectionApi {
	if _, ok := CollectionApiConfigs[objectType]; !ok {
		panic(fmt.Sprintf("invalid objectype %s!", objectType))
	}

	c := &CollectionApi{
		apiConfig:  CollectionApiConfigs[objectType],
		Collection: collection,
		App:        app,
		RESTKey:    key,
		Owner:      owner,
	}
	c.base = Base{
		Splunk:    clientConfig,
		RetryFunc: c.handleRequestError,
	}

	if c.apiConfig.BodyFormat == "" {
		c.apiConfig.BodyFormat = "JSON"
	}
	return c
}

func (c *CollectionApi) url() (u string) {
	const f = "https://%[1]s:%[2]d/servicesNS/%[3]s/%[4]s/%[5]s"
	u = fmt.Sprintf(f, c.base.Splunk.Host, c.base.Splunk.Port, c.Owner, c.App, c.apiConfig.Path)
	if c.apiConfig.ApiCollectionKeyInUrl {
		u = fmt.Sprintf("%[1]s/%[2]s", u, c.Collection)
	}
	if c.apiConfig.ApiKeyInUrl {
		u = fmt.Sprintf("%[1]s/%[2]s", u, c.RESTKey)
	}
	if c.apiConfig.PathExtension != "" {
		u = u + "/" + c.apiConfig.PathExtension
	}
	if c.Params != "" {
		u = u + "?" + c.Params
	}
	return
}

func (c *CollectionApi) body() (body []byte, err error) {
	if c.apiConfig.ApiDoNotSendBody {
		return nil, nil
	}
	if c.Body != nil {
		body = c.Body
	} else if c.Data != nil {
		body, err = c.Marshal(c.Data)
	}
	return
}

func (c *CollectionApi) handleRequestError(ctx context.Context, method string, statusCode int, responseBody []byte, requestErr error) (shouldRetry bool, newStatusCode int, newBody []byte, err error) {
	newStatusCode, newBody, err = statusCode, responseBody, requestErr

	switch {
	case statusCode == 400 || statusCode == 401 || statusCode == 403 || statusCode == 404 || statusCode == 409: //do not retry
	default:
		shouldRetry = true
	}

	return
}

func (c *CollectionApi) Create(ctx context.Context) (*CollectionApi, error) {
	tflog.Trace(ctx, "COLLECTION CREATE: Create", map[string]interface{}{"c": c})
	reqBody, err := c.body()
	if err != nil {
		return nil, err
	}

	_, respBody, err := c.base.requestWithRetry(ctx, http.MethodPost, c.url(), reqBody)
	if err != nil {
		return nil, err
	}
	if c.apiConfig.Path == "storage/collections/data" && c.apiConfig.PathExtension != "batch_save" {
		data := make(map[string]string)
		if err = json.Unmarshal(respBody, &data); err == nil {
			if key, ok := data["_key"]; ok {
				c.RESTKey = key
			}
		}
	}

	return c, nil
}

func (c *CollectionApi) Read(ctx context.Context) (*CollectionApi, error) {
	tflog.Trace(ctx, "COLLECTION READ: Read", map[string]interface{}{"c": c})

	var method string
	var body []byte = nil
	var err error
	if c.apiConfig.PathExtension == "batch_find" {
		method = http.MethodPost
		body, err = c.body()
		if err != nil {
			return nil, err
		}
	} else {
		method = http.MethodGet
	}

	_, respBody, err := c.base.requestWithRetry(ctx, method, c.url(), body)
	if err != nil {
		return nil, err
	}
	if respBody == nil {
		return nil, nil
	}

	if !c.apiConfig.ApiIgnoreResponseBody {
		c.Body = respBody
	}
	return c, nil
}

func (c *CollectionApi) Update(ctx context.Context) (*CollectionApi, error) {
	tflog.Trace(ctx, "COLLECTION UPDATE: Update", map[string]interface{}{"c": c})

	reqBody, err := c.body()
	if err != nil {
		return nil, err
	}

	_, _, err = c.base.requestWithRetry(ctx, http.MethodPost, c.url(), reqBody)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *CollectionApi) Delete(ctx context.Context) (*CollectionApi, error) {
	tflog.Trace(ctx, "COLLECTION DELETE: Delete", map[string]interface{}{"c": c})

	_, _, err := c.base.requestWithRetry(ctx, http.MethodDelete, c.url(), nil)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *CollectionApi) Marshal(obj interface{}) (bytes []byte, err error) {
	switch c.apiConfig.BodyFormat {
	case "JSON":
		bytes, err = json.Marshal(obj)
	case "XML":
		bytes, err = xml.Marshal(obj)
	default:
		err = fmt.Errorf("unknown REST body format")
	}
	return
}

func (c *CollectionApi) Unmarshal(bytes []byte) (res interface{}, err error) {
	switch c.apiConfig.BodyFormat {
	case "JSON":
		err = json.Unmarshal(bytes, &res)
	case "XML":
		err = xml.Unmarshal(bytes, &res)
	default:
		err = fmt.Errorf("unknown REST body format")
	}
	return
}
