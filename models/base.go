package models

// Models and support for interacting (generically) with Splunk ITSI's
// ITOA object-related REST APIs.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/lestrrat-go/backoff/v2"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
	"gopkg.in/yaml.v3"
)

const (
	resourceHashField      = "_tf_hash"
	asyncUpdateCheckPeriod = 15 * time.Second
)

var RestConfigs map[string]restConfig

type RawJson json.RawMessage

func (rj RawJson) ToInterfaceMap() (m map[string]interface{}, err error) {
	var by []byte
	if by, err = json.RawMessage(rj).MarshalJSON(); err != nil {
		return
	}
	err = json.Unmarshal(by, &m)
	return
}

func (rj RawJson) MarshalJSON() ([]byte, error) {
	return json.RawMessage(rj).MarshalJSON()
}

func (rj *RawJson) UnmarshalJSON(data []byte) error {
	*rj = RawJson(append(make([]byte, 0, len(data)), data...))
	return nil
}

var GenerateResourceKey = func() (string, error) {
	return uuid.GenerateUUID()
}

func init() {
	err := yaml.Unmarshal([]byte(metadataConfig), &RestConfigs)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
}

type restConfig struct {
	RestInterface string `yaml:"rest_interface"`
	ObjectType    string `yaml:"object_type"`
	RestKeyField  string `yaml:"rest_key_field"`
	TFIDField     string `yaml:"tfid_field"`
	MaxPageSize   int    `yaml:"max_page_size"`
	GenerateKey   bool   `yaml:"generate_key"`
}

type Base struct {
	Splunk ClientConfig
	restConfig
	// key used to collect this resource via the REST API
	RESTKey string
	// Terraform Identifier
	TFID    string
	RawJson RawJson
	Fields  []string
	Hash    string
}

func init() {
	clients = InitHttpClients()
	err := yaml.Unmarshal([]byte(metadataConfig), &RestConfigs)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
}

func NewBase(clientConfig ClientConfig, key, id, objectType string) *Base {
	if _, ok := RestConfigs[objectType]; !ok {
		panic(fmt.Sprintf("invalid objectype %s!", objectType))
	}
	b := &Base{
		Splunk:     clientConfig,
		restConfig: RestConfigs[objectType],
		RESTKey:    key,
		TFID:       id,
	}
	return b
}

func (b *Base) Clone() *Base {
	b_ := &Base{
		restConfig: b.restConfig,
		RawJson:    b.RawJson,
		Splunk:     b.Splunk,
		RESTKey:    b.RESTKey,
		TFID:       b.TFID,
	}
	return b_
}

func (b *Base) urlBase() string {
	const restBaseFmt = "https://%[1]s:%[2]d/servicesNS/nobody/SA-ITOA/%[3]s/%[4]s"
	url := fmt.Sprintf(restBaseFmt, b.Splunk.Host, b.Splunk.Port, b.RestInterface, b.ObjectType)
	return url
}

func (b *Base) urlBaseWithKey() string {
	const restKeyFmt = "https://%[1]s:%[2]d/servicesNS/nobody/SA-ITOA/%[3]s/%[4]s/%[5]s"
	url := fmt.Sprintf(restKeyFmt, b.Splunk.Host, b.Splunk.Port, b.RestInterface, b.ObjectType, b.RESTKey)
	return url
}

func (b *Base) handleConflictOnCreate(ctx context.Context) (responseBody []byte, err error) {
	b_, err := b.Read(ctx)
	if err != nil {
		return
	} else if b_ == nil {
		err = fmt.Errorf("error while handling 409 Conflict response for create %s request", b.ObjectType)
		return
	}

	if b.RESTKey == b_.RESTKey {
		responseBody = []byte(fmt.Sprintf("{\"%s\": \"%s\"}", b.RestKeyField, b.RESTKey))
	} else {
		err = fmt.Errorf("409 Conflict response for create %s request", b.ObjectType)
	}

	return
}

func (b *Base) handleRequestError(ctx context.Context, method string, statusCode int, responseBody []byte, requestErr error) (shouldRetry bool, newStatusCode int, newBody []byte, err error) {
	//Common unretriable errors
	//400: Bad Request
	//401: Unauthorized
	//403: Forbidden
	//404: Not Found
	//409: Conflict

	newStatusCode, newBody, err = statusCode, responseBody, requestErr

	switch {
	case method == http.MethodPost && statusCode == http.StatusConflict && b.GenerateKey:
		if newBody, err = b.handleConflictOnCreate(ctx); err != nil {
			newStatusCode = http.StatusOK
		}
	case statusCode == 400 || statusCode == 401 || statusCode == 403 || statusCode == 404 || statusCode == 409: //do not retry
	default:
		shouldRetry = true
	}

	return
}

func (b *Base) requestWithRetry(ctx context.Context, method string, url string, body []byte) (statusCode int, responseBody []byte, requestErr error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	bo := b.Splunk.RetryPolicy.Start(ctx)

	attempt := 1

	for backoff.Continue(bo) {
		start := time.Now()
		statusCode, responseBody, requestErr = b.request(ctx, method, url, body)
		tflog.Trace(ctx, fmt.Sprintf("%v %v (%v): %v %v [%s]", method, url, attempt, statusCode, http.StatusText(statusCode), time.Since(start).String()))
		if requestErr != nil {

			if shouldRetry, newStatus, newBody, err := b.handleRequestError(ctx, method, statusCode, responseBody, requestErr); !shouldRetry {
				if err == nil {
					statusCode = newStatus
					responseBody = newBody
				} else {
					tflog.Error(ctx, fmt.Sprintf("%v %v (%v) failed: %v", attempt, method, url, statusCode))
					responseBody = nil
				}

				requestErr = err
				return
			}

			if ctx.Err() == nil {
				attempt++
				continue
			}
		}

		break
	}

	if requestErr == nil {
		requestErr = ctx.Err()
	}

	if requestErr != nil {
		tflog.Error(ctx, fmt.Sprintf("%v %v (%v) failed: %s", method, url, attempt, requestErr.Error()))
	}
	return
}

func (b *Base) request(ctx context.Context, method string, url string, body []byte) (statusCode int, responseBody []byte, err error) {
	client := clients.Get(b.Splunk)
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
	if err != nil {
		return
	}
	if Verbose {
		err = logRequest(req)
		if err != nil {
			return
		}
	}

	if b.Splunk.BearerToken != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", b.Splunk.BearerToken))
	} else {
		req.SetBasicAuth(b.Splunk.User, b.Splunk.Password)
	}
	req.Header.Add("Content-Type", "application/json")

	itsiLimiter.Acquire()
	defer itsiLimiter.Release()

	resp, err := client.Do(req)
	if resp != nil {
		statusCode = resp.StatusCode
	}
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if Verbose {
		if err = logResponse(resp); err != nil {
			return
		}
	}
	responseBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("%v error: %v", method, resp.Status)
	}

	success := false

	switch method {
	case http.MethodDelete:
		if resp.StatusCode == 404 {
			// Ignore 404 errors for DELETE requests, when we are trying to delete a nonexistent resource
			success = true
			break
		}
		success = resp.StatusCode >= 200 && resp.StatusCode < 300
	case http.MethodGet:
		success = resp.StatusCode == 200 || resp.StatusCode == 404
		if statusCode != 200 {
			responseBody = nil
		}
	default:
		success = resp.StatusCode == 200
	}

	if !success {
		return resp.StatusCode, nil, fmt.Errorf("%v error: %v \n%s", method, resp.Status, responseBody)
	}

	return
}

func (b *Base) GetPageSize() int {
	maxPageSize := b.restConfig.MaxPageSize
	if maxPageSize == 0 {
		return -1
	}
	return maxPageSize
}

func (b *Base) PopulateRawJSON(ctx context.Context, body map[string]interface{}) error {
	if b.GenerateKey && b.RESTKey == "" {
		key, err := GenerateResourceKey()
		if err != nil {
			return err
		}

		body[b.RestKeyField] = key
		b.RESTKey = key
	}
	//compute body hash
	by, err := json.Marshal(body)
	if err != nil {
		return err
	}
	b.Hash = util.Sha256(by)
	body[resourceHashField] = b.Hash
	// populate b.RawJson
	by, err = json.Marshal(body)
	if err != nil {
		return err
	}
	return json.Unmarshal(by, &b.RawJson)
}

func (b *Base) Create(ctx context.Context) (*Base, error) {

	reqBody, err := json.Marshal(b.RawJson)
	if err != nil {
		return nil, err
	}
	var respBody []byte
	_, respBody, err = b.requestWithRetry(ctx, http.MethodPost, b.urlBase(), reqBody)
	if err != nil {
		return nil, err
	}
	var r map[string]string
	err = json.Unmarshal(respBody, &r)
	if err != nil {
		return nil, err
	}
	b.RESTKey = r[b.restConfig.RestKeyField]
	b.storeCache()
	return b, nil
}

func (b *Base) Read(ctx context.Context) (*Base, error) {
	if b.RESTKey == "" {
		return nil, fmt.Errorf("could not Read %s resource: RESTKey was not provided", b.ObjectType)
	}

	_, respBody, err := b.requestWithRetry(ctx, http.MethodGet, b.urlBaseWithKey(), nil)
	if err != nil || respBody == nil {
		return nil, err
	}

	var raw json.RawMessage
	err = json.Unmarshal(respBody, &raw)
	if err != nil {
		return nil, err
	}
	base := b.Clone()
	err = base.Populate(raw)
	if err != nil {
		return nil, err
	}
	base.storeCache()
	return base, nil
}

func (b *Base) Update(ctx context.Context) error {
	reqBody, err := json.Marshal(b.RawJson)
	if err != nil {
		return err
	}

	_, _, err = b.requestWithRetry(ctx, http.MethodPut, b.urlBaseWithKey(), reqBody)
	if err != nil {
		return err
	}
	b.storeCache()
	return nil
}

func (b *Base) updateAndWaitForState(ctx context.Context) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	reqBody, err := json.Marshal(b.RawJson)
	if err != nil {
		return err
	}

	resultCh := make(chan error, 1)
	go func() {
		defer close(resultCh)
		_, _, err := b.requestWithRetry(ctx, http.MethodPut, b.urlBaseWithKey(), reqBody)
		resultCh <- err
	}()

	ticker := time.NewTicker(asyncUpdateCheckPeriod)
	defer ticker.Stop()

	for {
		select {
		case err = <-resultCh:
			return
		case <-ticker.C:
			if originHash, err := b.getOriginHash(ctx); err == nil && originHash != "" {
				if originHash == b.Hash {
					return nil
				}
			} else if err != nil {
				return err
			}
			continue
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}
}

func (b *Base) UpdateAsync(ctx context.Context) error {
	err := b.updateAndWaitForState(ctx)
	if err != nil {
		return err
	}
	b.storeCache()
	return nil
}

func (b *Base) Delete(ctx context.Context) error {
	Cache.Remove(b)
	_, _, err := b.requestWithRetry(ctx, http.MethodDelete, b.urlBaseWithKey(), nil)
	return err
}

func (b *Base) storeCache() {
	Cache.Add(b)
}

// Returns an object from cache if it's present, or makes the relevant API calls..
func (b *Base) Find(ctx context.Context) (result *Base, err error) {
	if b.RESTKey == "" && b.TFID == "" {
		return nil, fmt.Errorf("could not Find %s resource: neither RESTKey nor TFID were provided", b.ObjectType)
	}

	cacheMu.Lock()
	item, found := Cache.Get(b)
	if !found || item == nil {
		//create a new cache item
		item = Cache.Reset(b)
	} else {
		result = item.base
	}
	cacheMu.Unlock()

	if result != nil {
		//return item found in the cache
		return
	}

	// Make the necessary API calls to retrieve the respective resource only ONCE per lifetime of the cache item
	// (even across the Find invocations for the same resource that might be running simultaneously).
	// When we invoke `.Do`, if there is an on-going simultaneous operation,
	// it will block until it has completed (and `item.base` is populated).
	// Or if the operation has already completed once before, this call is a no-op and doesn't block.
	item.once.Do(func() {
		b_ := b.Clone()
		if b_.TFIDField == "_key" && b_.RESTKey == "" && b_.TFID != "" {
			// If we don't know REST Key, but know TFID,
			// and that TFID is also _key field, we can infer REST Key and save 1 API call
			b_.RESTKey = b_.TFID
		}

		if b_.RESTKey == "" {
			// Handle the situation when RESTKey is unknown.
			// This can happen only when we are importing resources or reading data sources
			if err = b_.lookupRESTKey(ctx); err != nil {
				return
			} else if b_.RESTKey == "" {
				//Here, REST Key is still unknown after lookup by TFID.
				//This should normally happen if we use `terraform import`, but provide the REST Key instead of the TF ID
				//At this point, our last chance to find the resouce, is to attempt `Read` using TFID for RESTKey
				b_.RESTKey = b_.TFID
				b_.TFID = ""
			}
		}
		item.base, err = b_.Read(ctx)
		return
	})

	if err == nil {
		//at this point `item.base` must be pointing to the data returned by the API for this resource
		//(even if that happened in a different `Find` goroutine).
		//Or it can be nil, if the resource was not found.
		result = item.base
	} else {
		//API call to retrieve the resource has failed.
		//Reset the cache item (`Once` object in particular),
		//so that we can try again during further invocations of Find for the same resource.
		Cache.Reset(b)
	}
	return
}

func (b *Base) Dump(ctx context.Context, offset, count int) ([]*Base, error) {

	params := url.Values{}
	params.Add("sort_key", b.restConfig.RestKeyField)
	params.Add("sort_dir", "asc")
	if count > 0 && offset >= 0 {
		params.Add("count", strconv.Itoa(count))
		params.Add("offset", strconv.Itoa(offset))
	}

	log.Printf("Requesting %s with params %s\n", b.restConfig.ObjectType, params.Encode())
	_, respBody, err := b.requestWithRetry(ctx, http.MethodGet, fmt.Sprintf("%s?%s", b.urlBase(), params.Encode()), nil)
	if err != nil || respBody == nil {
		return nil, err
	}

	var raw []json.RawMessage
	err = json.Unmarshal(respBody, &raw)
	if err != nil {
		return nil, err
	}
	res := []*Base{}
	for _, r := range raw {
		b_ := b.Clone()
		err = b_.Populate(r)
		if err != nil {
			return nil, err
		}
		res = append(res, b_)
	}
	return res, err
}

func (b *Base) Populate(raw []byte) error {
	err := json.Unmarshal(raw, &b.RawJson)
	if err != nil {
		return err
	}
	var fieldsMap map[string]*json.RawMessage
	err = json.Unmarshal(raw, &fieldsMap)
	if err != nil {
		return err
	}
	key := b.RestKeyField
	if _, ok := fieldsMap[key]; !ok {
		return fmt.Errorf("missing %s RESTKey field for %s", key, b.ObjectType)
	}
	keyBytes, err := fieldsMap[key].MarshalJSON()
	if err != nil {
		return err
	}
	err = json.Unmarshal(keyBytes, &b.RESTKey)
	if err != nil {
		return err
	}
	id := b.TFIDField
	if _, ok := fieldsMap[id]; !ok {
		return fmt.Errorf("missing %s TFID field for %s", id, b.ObjectType)
	}
	idBytes, err := fieldsMap[id].MarshalJSON()
	if err != nil {
		return err
	}
	err = json.Unmarshal(idBytes, &b.TFID)
	if err != nil {
		return err
	}
	b.Fields = []string{}
	for field := range fieldsMap {
		b.Fields = append(b.Fields, field)
	}

	sort.Strings(b.Fields)

	if Cache != nil {
		Cache.restKey.Update(b.RestInterface, b.ObjectType, b.TFID, b.RESTKey)
	}

	return nil
}

func (b *Base) lookupRESTKey(ctx context.Context) error {
	params := url.Values{}
	params.Add("limit", "2")
	params.Add("filter", fmt.Sprintf("{\"%s\":\"%s\"}", b.TFIDField, b.TFID))
	params.Add("fields", strings.Join([]string{b.TFIDField, b.RestKeyField}, ","))

	_, respBody, err := b.requestWithRetry(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s?%s", b.urlBase(), params.Encode()),
		nil)

	if err != nil || respBody == nil {
		return err
	}

	var raw []json.RawMessage
	err = json.Unmarshal(respBody, &raw)
	if err != nil {
		return err
	}

	if len(raw) > 1 {
		return fmt.Errorf("failed to lookup RESTKey for %s TFID %s: TFID is not unique", b.ObjectType, b.TFID)
	}

	for _, r := range raw {
		b_ := b.Clone()
		if err = b_.Populate(r); err != nil {
			return err
		}
		Cache.restKey.Update(b.RestInterface, b.ObjectType, b.TFID, b.RESTKey)
		b.RESTKey = b_.RESTKey
		return nil
	}

	return nil
}

func (b *Base) getOriginHash(ctx context.Context) (string, error) {
	params := url.Values{}
	params.Add("filter", fmt.Sprintf("{\"%s\":\"%s\"}", b.RestKeyField, b.RESTKey))
	params.Add("fields", strings.Join([]string{b.RestKeyField, b.TFIDField, resourceHashField}, ","))

	_, respBody, err := b.requestWithRetry(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s?%s", b.urlBase(), params.Encode()),
		nil)

	if err != nil || respBody == nil {
		return "", err
	}

	var raw []json.RawMessage
	if err = json.Unmarshal(respBody, &raw); err != nil {
		return "", err
	}

	if len(raw) > 1 {
		return "", fmt.Errorf("failed to lookup hash for %s %s: object is not unique", b.ObjectType, b.RESTKey)
	}

	for _, r := range raw {
		b_ := b.Clone()
		if err = b_.Populate(r); err != nil {
			return "", err
		}

		m, err := b_.RawJson.ToInterfaceMap()
		if err != nil {
			return "", err
		}

		if iHash, ok := m[resourceHashField]; ok {
			if hash, ok_ := iHash.(string); ok_ {
				return hash, nil
			}
		}
	}

	return "", nil
}
