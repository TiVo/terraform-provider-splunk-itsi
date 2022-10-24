package models

// Models and support for interacting with Splunk's collection-related
// REST APIs.

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/lestrrat-go/backoff/v2"
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

collection_config:
    path:                            storage/collections/config
    api_key_in_url:                  true
    body_format:                     XML
    api_do_not_send_body:            true
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
    tfid_field:                      _key

collection_entry:
    path:                            storage/collections/data
    api_collection_key_in_url:       true
    api_key_in_url:                  true
    body_format:                     JSON
    rest_key_field:                  _key
    tfid_field:                      _key

collection_entry_no_body:
    path:                            storage/collections/data
    api_collection_key_in_url:       true
    api_key_in_url:                  true
    body_format:                     JSON
    api_ignore_response_body:        true
    rest_key_field:                  _key
    tfid_field:                      _key

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
`
)

type CollectionApi struct {
	splunk    ClientConfig
	apiConfig collectionApiConfig

	RESTKey    string                 // key used to collect this resource via the REST API
	Collection string                 // Collection name
	Data       map[string]interface{} // Data for this object
	Params     string                 // URL query string, iff provided
	Body       []byte                 // Body used for this API call
}

func NewCollection(clientConfig ClientConfig, collection, key, objectType string) *CollectionApi {
	if _, ok := CollectionApiConfigs[objectType]; !ok {
		panic(fmt.Sprintf("invalid objectype %s!", objectType))
	}
	c := &CollectionApi{
		splunk:     clientConfig,
		apiConfig:  CollectionApiConfigs[objectType],
		RESTKey:    key,
		Collection: collection,
	}
	if c.apiConfig.BodyFormat == "" {
		c.apiConfig.BodyFormat = "JSON"
	}
	return c
}

func (c *CollectionApi) url() (u string) {
	const f = "https://%[1]s:%[2]d/servicesNS/nobody/itsi/%[3]s"
	u = fmt.Sprintf(f, c.splunk.Host, c.splunk.Port, c.apiConfig.Path)
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

func (c *CollectionApi) shouldRetry(method string, statusCode int, err error) bool {
	//Common unretriable errors
	//400: Bad Request
	//401: Unauthorized
	//403: Forbidden
	//404: Not Found
	//409: Conflict
	if statusCode == 400 || statusCode == 401 || statusCode == 403 ||
		statusCode == 404 || statusCode == 409 {
		return false
	}
	return true
}

func (c *CollectionApi) requestWithRetry(ctx context.Context, method string, url string, body []byte) (statusCode int, responseBody []byte, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	co := c.splunk.RetryPolicy.Start(ctx)

	attempt := 1
	for backoff.Continue(co) {
		start := time.Now()
		statusCode, responseBody, err = c.request(ctx, method, url, body)
		tflog.Trace(ctx, fmt.Sprintf("%v %v (%v): %v %v [%s]", method, url,
			attempt, statusCode, http.StatusText(statusCode),
			time.Since(start).String()))
		if err != nil {
			if !c.shouldRetry(method, statusCode, err) {
				tflog.Error(ctx, fmt.Sprintf("%v %v (%v) failed: %v",
					attempt, method, url, statusCode))
				responseBody = nil
				return
			}
			attempt++
			continue
		}
		break
	}

	if err == nil {
		err = ctx.Err()
	}

	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("%v %v (%v) failed: %s", method, url,
			attempt, err.Error()))
	}
	return
}

func (c *CollectionApi) request(ctx context.Context, method string, u string, body []byte) (statusCode int, responseBody []byte, err error) {
	client := clients.Get(c.splunk)
	req, err := http.NewRequest(method, u, bytes.NewBuffer(body))
	if err != nil {
		return
	}
	if Verbose {
		err = logRequest(req)
		if err != nil {
			return
		}
	}

	if c.splunk.BearerToken != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.splunk.BearerToken))
	} else {
		req.SetBasicAuth(c.splunk.User, c.splunk.Password)
	}
	if c.apiConfig.ContentType != "" {
		req.Header.Add("Content-Type", c.apiConfig.ContentType)
	} else if c.apiConfig.BodyFormat == "JSON" {
		req.Header.Add("Content-Type", "application/json")
	}

	itsiLimiter.Acquire()
	defer itsiLimiter.Release()

	tflog.Debug(ctx, "COLLECTION: Created a request",
		map[string]interface{}{"key": c.RESTKey, "method": method, "url": u})

	tflog.Trace(ctx, "COLLECTION:   Request body",
		map[string]interface{}{"key": c.RESTKey, "c": c, "body": string(body)})

	resp, err := client.Do(req)
	if resp != nil {
		statusCode = resp.StatusCode
	}
	if err != nil {
		tflog.Trace(ctx, "COLLECTION:   Request fail", map[string]interface{}{"key": c.RESTKey, "err": err})
		return
	}
	defer resp.Body.Close()
	tflog.Debug(ctx, "COLLECTION:   Response err code: "+resp.Status,
		map[string]interface{}{"key": c.RESTKey, "content-length": resp.ContentLength})
	if Verbose {
		if err = logResponse(resp); err != nil {
			return
		}
	}

	responseBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		tflog.Trace(ctx, "COLLECTION:   Response read err",
			map[string]interface{}{"key": c.RESTKey, "err": err})
		return statusCode, nil, fmt.Errorf("%v error: %v", method, resp.Status)
	}

	success := false
	switch method {
	case http.MethodDelete:
		if resp.StatusCode == 404 {
			// Ignore 404 errors for DELETE requests, when
			// we are trying to delete a nonexistent
			// resource...
			success = true
			break
		}
		success = resp.StatusCode >= 200 && resp.StatusCode < 300
	default:
		success = resp.StatusCode >= 200 && resp.StatusCode < 300
	}

	if !success {
		return statusCode, nil, fmt.Errorf("%v error: %v \n%s", method, resp.Status, responseBody)
	}
	return
}

func (c *CollectionApi) Create(ctx context.Context) (*CollectionApi, error) {
	tflog.Trace(ctx, "COLLECTION CREATE: Create", map[string]interface{}{"c": c})
	reqBody, err := c.body()
	if err != nil {
		return nil, err
	}

	_, _, err = c.requestWithRetry(ctx, http.MethodPost, c.url(), reqBody)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *CollectionApi) Read(ctx context.Context) (*CollectionApi, error) {
	tflog.Trace(ctx, "COLLECTION READ: Read", map[string]interface{}{"c": c})

	_, respBody, err := c.requestWithRetry(ctx, http.MethodGet, c.url(), nil)
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

	_, _, err = c.requestWithRetry(ctx, http.MethodPost, c.url(), reqBody)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *CollectionApi) Delete(ctx context.Context) (*CollectionApi, error) {
	tflog.Trace(ctx, "COLLECTION DELETE: Delete", map[string]interface{}{"c": c})

	_, _, err := c.requestWithRetry(ctx, http.MethodDelete, c.url(), nil)
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
