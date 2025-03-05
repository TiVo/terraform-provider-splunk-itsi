package models

// Models and support for interacting (generically) with Splunk ITSI's
// ITOA object-related REST APIs.

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tivo/terraform-provider-splunk-itsi/util"
	"gopkg.in/yaml.v3"
)

const (
	resourceHashField      = "_tf_hash"
	asyncUpdateCheckPeriod = 15 * time.Second
	updateSuccessTimeout   = 100 * time.Second //If the update success cannot be confirmed within this period, the update will be considered as failed and retried.
)

var RestConfigs map[string]restConfig

type RawJson json.RawMessage

func (rj RawJson) ToInterfaceMap() (m map[string]any, err error) {
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
	RestInterface          string `yaml:"rest_interface"`
	ObjectType             string `yaml:"object_type"`
	RestKeyField           string `yaml:"rest_key_field"`
	TFIDField              string `yaml:"tfid_field"`
	MaxPageSize            int    `yaml:"max_page_size"`
	GenerateKey            bool   `yaml:"generate_key"`
	UnimplementedFiltering bool   `yaml:"unimplemented_filtering"`
}

type ItsiObj struct {
	Base
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

func NewItsiObj(clientConfig ClientConfig, key, id, objectType string) *ItsiObj {
	if _, ok := RestConfigs[objectType]; !ok {
		panic(fmt.Sprintf("invalid objectype %s!", objectType))
	}
	obj := &ItsiObj{
		restConfig: RestConfigs[objectType],
		RESTKey:    key,
		TFID:       id,
	}

	obj.Base = Base{
		Splunk:    clientConfig,
		RetryFunc: obj.handleRequestError,
	}
	return obj
}

func (obj *ItsiObj) Clone() *ItsiObj {
	obj_ := &ItsiObj{
		restConfig: obj.restConfig,
		RawJson:    obj.RawJson,
		Base:       obj.Base,
		RESTKey:    obj.RESTKey,
		TFID:       obj.TFID,
	}
	return obj_
}

func (obj *ItsiObj) urlBase() string {
	const restBaseFmt = "https://%[1]s:%[2]d/servicesNS/nobody/SA-ITOA/%[3]s/%[4]s"
	url := fmt.Sprintf(restBaseFmt, obj.Splunk.Host, obj.Splunk.Port, obj.RestInterface, obj.ObjectType)
	return url
}

func (obj *ItsiObj) urlBaseWithKey() string {
	const restKeyFmt = "https://%[1]s:%[2]d/servicesNS/nobody/SA-ITOA/%[3]s/%[4]s/%[5]s"
	url := fmt.Sprintf(restKeyFmt, obj.Splunk.Host, obj.Splunk.Port, obj.RestInterface, obj.ObjectType, obj.RESTKey)
	return url
}

func (obj *ItsiObj) handleConflictOnCreate(ctx context.Context) (responseBody []byte, err error) {
	obj_, err := obj.Read(ctx)
	if err != nil {
		return
	} else if obj_ == nil {
		err = fmt.Errorf("error while handling 409 Conflict response for create %s request", obj.ObjectType)
		return
	}

	if obj.RESTKey == obj_.RESTKey {
		responseBody = fmt.Appendf(nil, `{"%s": "%s"}`, obj.RestKeyField, obj.RESTKey)
	} else {
		err = fmt.Errorf("409 Conflict response for create %s request", obj.ObjectType)
	}

	return
}

func (obj *ItsiObj) handleRequestError(ctx context.Context, method string, statusCode int, responseBody []byte, requestErr error) (shouldRetry bool, newStatusCode int, newBody []byte, err error) {
	//Common unretriable errors
	//400: Bad Request
	//401: Unauthorized
	//403: Forbidden
	//404: Not Found
	//409: Conflict

	newStatusCode, newBody, err = statusCode, responseBody, requestErr

	switch {
	case method == http.MethodPost && statusCode == http.StatusConflict && obj.GenerateKey:
		if newBody, err = obj.handleConflictOnCreate(ctx); err != nil {
			newStatusCode = http.StatusOK
		}
	case method == http.MethodDelete && statusCode == http.StatusInternalServerError:
		shouldRetry, err = obj.exists(ctx)
	case statusCode == 400 || statusCode == 401 || statusCode == 403 || statusCode == 404 || statusCode == 409: //do not retry
	default:
		shouldRetry = true
	}

	return
}

func (obj *ItsiObj) GetPageSize() int {
	maxPageSize := obj.restConfig.MaxPageSize
	if maxPageSize == 0 {
		return -1
	}
	return maxPageSize
}

func (obj *ItsiObj) IsFilterSupported() bool {
	return !obj.restConfig.UnimplementedFiltering
}

func (obj *ItsiObj) PopulateRawJSON(ctx context.Context, body map[string]any) error {
	if obj.GenerateKey && obj.RESTKey == "" {
		key, err := GenerateResourceKey()
		if err != nil {
			return err
		}

		body[obj.RestKeyField] = key
		obj.RESTKey = key
	}
	//compute body hash
	by, err := json.Marshal(body)
	if err != nil {
		return err
	}
	obj.Hash = util.Sha256(by)
	body[resourceHashField] = obj.Hash
	// populate obj.RawJson
	by, err = json.Marshal(body)
	if err != nil {
		return err
	}
	return json.Unmarshal(by, &obj.RawJson)
}

func (obj *ItsiObj) Create(ctx context.Context) (*ItsiObj, error) {

	reqBody, err := json.Marshal(obj.RawJson)
	if err != nil {
		return nil, err
	}
	var respBody []byte
	_, respBody, err = obj.requestWithRetry(ctx, http.MethodPost, obj.urlBase(), reqBody)
	if err != nil {
		return nil, err
	}
	var r map[string]string
	err = json.Unmarshal(respBody, &r)
	if err != nil {
		return nil, err
	}
	obj.RESTKey = r[obj.restConfig.RestKeyField]
	obj.storeCache()
	return obj, nil
}

func (obj *ItsiObj) Read(ctx context.Context) (*ItsiObj, error) {
	if obj.RESTKey == "" {
		return nil, fmt.Errorf("could not Read %s resource: RESTKey was not provided", obj.ObjectType)
	}

	_, respBody, err := obj.requestWithRetry(ctx, http.MethodGet, obj.urlBaseWithKey(), nil)
	if err != nil || respBody == nil {
		return nil, err
	}

	var raw json.RawMessage
	err = json.Unmarshal(respBody, &raw)
	if err != nil {
		return nil, err
	}
	itsi_obj := obj.Clone()
	err = itsi_obj.Populate(raw)
	if err != nil {
		return nil, err
	}
	itsi_obj.storeCache()
	return itsi_obj, nil
}

func (obj *ItsiObj) Update(ctx context.Context) error {
	reqBody, err := json.Marshal(obj.RawJson)
	if err != nil {
		return err
	}

	_, _, err = obj.requestWithRetry(ctx, http.MethodPut, obj.urlBaseWithKey(), reqBody)
	if err != nil {
		return err
	}
	obj.storeCache()
	return nil
}

func (obj *ItsiObj) updateConfirm(ctx context.Context) (ok bool, diags diag.Diagnostics) {
	/*
		Sometimes PUT request may return 200 before an object is actually updated.
		To handle this case, once PUT returns 200 we'll be making follow up GET requests to check if the object hash matches the expected one.
		If we cannot confirm a successful update within the `updateSuccessTimeout` timeout, we'll cancel the context and return ok=false,
		to indicate that the update was not successful.
	*/

	updateDeadlineExceeded := fmt.Errorf("update deadline exceeded")

	ctx, cancel := context.WithTimeoutCause(ctx, updateSuccessTimeout, updateDeadlineExceeded)
	defer cancel()

	reqBody, err := json.Marshal(obj.RawJson)
	if err != nil {
		return
	}

	resultCh := make(chan error, 1)

	start := time.Now()
	go func() {
		_, _, err := obj.requestWithRetry(ctx, http.MethodPut, obj.urlBaseWithKey(), reqBody)
		if ctx.Err() == nil {
			resultCh <- err
		}
	}()

	ticker := time.NewTicker(asyncUpdateCheckPeriod)
	defer ticker.Stop()

	updateReqComplete := false
	postUpdateHashCheckMismatch := 0
	checkOriginHashFunc := func() (ok bool, err error) {
		originHash, err := obj.getOriginHash(ctx)
		if ctx.Err() != nil {
			return false, nil
		}
		if err != nil {
			return false, err
		}

		hashMatches := (originHash == obj.Hash && originHash != "")

		tflog.Warn(ctx,
			fmt.Sprintf("[Update %s %s] [checkOriginHash] update_request_completed=%v checksums_match=%v time_since_update_request=%s _tf_hash_expected=%s _tf_hash_actual=%s",
				obj.ObjectType, obj.RESTKey, updateReqComplete, hashMatches, time.Since(start).String(), obj.Hash, originHash),
		)

		if updateReqComplete && !hashMatches {
			postUpdateHashCheckMismatch++
			diags = diag.Diagnostics{}
			warnMsg := fmt.Sprintf(util.Dedent(`
				Update %s %s: The update request returned a 200 OK status, but we were unable to confirm its success after %d attempts.
				This might be due to the update taking longer to propagate or be fully applied.
			`), obj.ObjectType, obj.RESTKey, postUpdateHashCheckMismatch)
			diags.AddWarning("Transient Update Failure", warnMsg)
			tflog.Warn(ctx, warnMsg)
		}
		return hashMatches, nil
	}

	for {
		select {
		case err = <-resultCh:
			if err != nil {
				diags.AddError("PUT Request Error", err.Error())
				return false, diags
			}

			updateReqComplete = true
			if ok, err := checkOriginHashFunc(); err == nil {
				if ok {
					return true, diags
				}
			} else {
				diags.AddError("Origin Hash Check Error", err.Error())
				return false, diags
			}
		case <-ticker.C:
			if ok, err := checkOriginHashFunc(); err == nil {
				if ok {
					return true, diags
				}
			} else {
				diags.AddError("Origin Hash Check Error", err.Error())
				return false, diags
			}
		case <-ctx.Done():
			if context.Cause(ctx) == updateDeadlineExceeded {
				return false, diags
			}

			diags.AddError(ctx.Err().Error(), ctx.Err().Error())
			return false, diags
		}
	}

}

// Retries async updates until a successful update is confirmed by comparing
// the expected hash value against the hash values stored in remote state
func (obj *ItsiObj) updateAndWaitForState(ctx context.Context) (diags diag.Diagnostics) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ok := false
	start := time.Now()

	var i int
	for i = 0; !ok && !diags.HasError() && ctx.Err() == nil; i++ {
		var d diag.Diagnostics

		ok, d = obj.updateConfirm(ctx)

		if ctx.Err() != nil {
			break
		}

		if diags.Append(d...); diags.HasError() {
			return
		}

		if !ok {
			tflog.Warn(ctx, "Transient Update Failure: "+
				fmt.Sprintf(`%s %s: Unable to confirm the update's success after waiting for %s.`,
					obj.ObjectType, obj.RESTKey, time.Since(start).String()))
		}
	}

	if ok && i > 1 {
		diags.AddWarning("Update operation completed with transient failures",
			fmt.Sprintf(`%s %s was updated successfully after %s but encountered %d verification failure(s). This may indicate an issue with ITSI backend.`,
				obj.ObjectType, obj.RESTKey, time.Since(start).String(), i-1))
	}

	if ctx.Err() != nil {
		diags.AddError(ctx.Err().Error(), ctx.Err().Error())
	}

	return
}

func (obj *ItsiObj) UpdateAsync(ctx context.Context) (diags diag.Diagnostics) {
	diags = obj.updateAndWaitForState(ctx)
	if !diags.HasError() {
		obj.storeCache()
	}
	return
}

func (obj *ItsiObj) Delete(ctx context.Context) (diags diag.Diagnostics) {
	Cache.Remove(obj)
	var err error

	start := time.Now()
	var i int

	for i = 0; ; i++ {
		_, _, err = obj.requestWithRetry(ctx, http.MethodDelete, obj.urlBaseWithKey(), nil)
		if err != nil {
			diags.AddError(fmt.Sprintf("Failed to delete %s/%s", obj.ObjectType, obj.RESTKey), err.Error())
			return
		}

		exists := true
		if exists, err = obj.exists(ctx); err != nil {
			diags.AddError(fmt.Sprintf("Failed to check if %s/%s exists", obj.ObjectType, obj.RESTKey), err.Error())
			return
		}

		if !exists {
			//deletion successful
			break
		}

		tflog.Warn(ctx, "Transient Delete Failure: "+
			fmt.Sprintf(`%s %s still exists despite the respective DELETE request having succeeded.`,
				obj.ObjectType, obj.RESTKey))
	}

	if i > 1 {
		diags.AddWarning("Delete operation completed with transient failures",
			fmt.Sprintf(`%s %s was deleted successfully after %s but encountered %d verification failure(s). This may indicate an issue with ITSI backend.`,
				obj.ObjectType, obj.RESTKey, time.Since(start).String(), i-1))
	}

	return
}

func (obj *ItsiObj) storeCache() {
	Cache.Add(obj)
}

// Returns an object from cache if it's present, or makes the relevant API calls..
func (obj *ItsiObj) Find(ctx context.Context) (result *ItsiObj, err error) {
	if obj.RESTKey == "" && obj.TFID == "" {
		return nil, fmt.Errorf("could not Find %s resource: neither RESTKey nor TFID were provided", obj.ObjectType)
	}

	cacheMu.Lock()
	item, found := Cache.Get(obj)
	if !found || item == nil {
		//create a new cache item
		item = Cache.Reset(obj)
	} else {
		result = item.obj
	}
	cacheMu.Unlock()

	if result != nil {
		//return item found in the cache
		return
	}

	// Make the necessary API calls to retrieve the respective resource only ONCE per lifetime of the cache item
	// (even across the Find invocations for the same resource that might be running simultaneously).
	// When we invoke `.Do`, if there is an on-going simultaneous operation,
	// it will block until it has completed (and `item.obj` is populated).
	// Or if the operation has already completed once before, this call is a no-op and doesn't block.
	item.once.Do(func() {
		obj_ := obj.Clone()
		if obj_.TFIDField == "_key" && obj_.RESTKey == "" && obj_.TFID != "" {
			// If we don't know REST Key, but know TFID,
			// and that TFID is also _key field, we can infer REST Key and save 1 API call
			obj_.RESTKey = obj_.TFID
		}

		if obj_.RESTKey == "" {
			// Handle the situation when RESTKey is unknown.
			// This can happen only when we are importing resources or reading data sources
			if err = obj_.lookupRESTKey(ctx); err != nil {
				return
			} else if obj_.RESTKey == "" {
				//Here, REST Key is still unknown after lookup by TFID.
				//This should normally happen if we use `terraform import`, but provide the REST Key instead of the TF ID
				//At this point, our last chance to find the resouce, is to attempt `Read` using TFID for RESTKey
				obj_.RESTKey = obj_.TFID
				obj_.TFID = ""
			}
		}
		item.obj, err = obj_.Read(ctx)
	})

	if err == nil {
		//at this point `item.base` must be pointing to the data returned by the API for this resource
		//(even if that happened in a different `Find` goroutine).
		//Or it can be nil, if the resource was not found.
		result = item.obj
	} else {
		//API call to retrieve the resource has failed.
		//Reset the cache item (`Once` object in particular),
		//so that we can try again during further invocations of Find for the same resource.
		Cache.Reset(obj)
	}
	return
}

type Parameters struct {
	Offset int
	Count  int
	Fields []string
	Filter string
}

func (obj *ItsiObj) Dump(ctx context.Context, queryParams *Parameters) ([]*ItsiObj, error) {

	params := url.Values{}
	params.Add("sort_key", obj.restConfig.RestKeyField)
	params.Add("sort_dir", "asc")
	if queryParams.Count > 0 && queryParams.Offset >= 0 {
		params.Add("count", strconv.Itoa(queryParams.Count))
		params.Add("offset", strconv.Itoa(queryParams.Offset))
	}
	if len(queryParams.Fields) > 0 {
		params.Add("fields", strings.Join(queryParams.Fields, ","))
	}
	if queryParams.Filter != "" {
		params.Add("filter", queryParams.Filter)
	}

	log.Printf("Requesting %s with params %s\n", obj.restConfig.ObjectType, params.Encode())
	_, respBody, err := obj.requestWithRetry(ctx, http.MethodGet, fmt.Sprintf("%s?%s", obj.urlBase(), params.Encode()), nil)
	if err != nil || respBody == nil {
		return nil, err
	}

	var raw []json.RawMessage
	err = json.Unmarshal(respBody, &raw)
	if err != nil {
		return nil, err
	}
	res := []*ItsiObj{}
	for _, r := range raw {
		obj_ := obj.Clone()
		err = obj_.Populate(r)
		if err != nil {
			return nil, err
		}
		res = append(res, obj_)
	}
	return res, err
}

func (obj *ItsiObj) Iter(ctx context.Context, queryParams *Parameters) iter.Seq2[*ItsiObj, error] {
	return func(yield func(*ItsiObj, error) bool) {
		filter := ""
		offset := 0
		limit := obj.GetPageSize()
		if queryParams != nil {
			filter = queryParams.Filter
			if queryParams.Count > 0 {
				limit = queryParams.Count
			}
		}

		for ; offset >= 0; offset += limit {
			items, err := obj.Dump(ctx, &Parameters{Offset: offset, Count: limit, Filter: filter})
			if err != nil {
				yield(nil, err)
				return
			}

			for _, item := range items {
				if !yield(item, nil) {
					return
				}
			}

			if len(items) < limit {
				return
			}
		}
	}
}

func (obj *ItsiObj) Populate(raw []byte) error {
	err := json.Unmarshal(raw, &obj.RawJson)
	if err != nil {
		return err
	}
	var fieldsMap map[string]*json.RawMessage
	err = json.Unmarshal(raw, &fieldsMap)
	if err != nil {
		return err
	}
	key := obj.RestKeyField
	if _, ok := fieldsMap[key]; !ok {
		return fmt.Errorf("missing %s RESTKey field for %s", key, obj.ObjectType)
	}
	keyBytes, err := fieldsMap[key].MarshalJSON()
	if err != nil {
		return err
	}
	err = json.Unmarshal(keyBytes, &obj.RESTKey)
	if err != nil {
		return err
	}
	id := obj.TFIDField
	if _, ok := fieldsMap[id]; !ok {
		return fmt.Errorf("missing %s TFID field for %s", id, obj.ObjectType)
	}
	idBytes, err := fieldsMap[id].MarshalJSON()
	if err != nil {
		return err
	}
	err = json.Unmarshal(idBytes, &obj.TFID)
	if err != nil {
		return err
	}
	obj.Fields = []string{}
	for field := range fieldsMap {
		obj.Fields = append(obj.Fields, field)
	}

	sort.Strings(obj.Fields)

	if Cache != nil {
		Cache.restKey.Update(obj.RestInterface, obj.ObjectType, obj.TFID, obj.RESTKey)
	}

	return nil
}

func (obj *ItsiObj) lookupRESTKey(ctx context.Context) error {
	params := url.Values{}
	params.Add("limit", "2")
	params.Add("filter", fmt.Sprintf(`{"%s":"%s"}`, obj.TFIDField, obj.TFID))
	params.Add("fields", strings.Join([]string{obj.TFIDField, obj.RestKeyField}, ","))

	_, respBody, err := obj.requestWithRetry(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s?%s", obj.urlBase(), params.Encode()),
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
		return fmt.Errorf("failed to lookup RESTKey for %s TFID %s: TFID is not unique", obj.ObjectType, obj.TFID)
	}

	for _, r := range raw {
		obj_ := obj.Clone()
		if err = obj_.Populate(r); err != nil {
			return err
		}
		Cache.restKey.Update(obj.RestInterface, obj.ObjectType, obj.TFID, obj.RESTKey)
		obj.RESTKey = obj_.RESTKey
	}

	return nil
}

func (obj *ItsiObj) exists(ctx context.Context) (ok bool, err error) {
	params := url.Values{}
	params.Add("filter", fmt.Sprintf(`{"%s":"%s"}`, obj.RestKeyField, obj.RESTKey))
	params.Add("fields", obj.RestKeyField)

	_, respBody, err := obj.requestWithRetry(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s?%s", obj.urlBase(), params.Encode()),
		nil)

	if err != nil {
		return
	}
	if respBody == nil {
		return false, fmt.Errorf("unexpected response while checking if an object exists")
	}

	var raw []json.RawMessage
	err = json.Unmarshal(respBody, &raw)
	ok = len(raw) > 0
	return
}

func (obj *ItsiObj) getOriginHash(ctx context.Context) (string, error) {
	params := url.Values{}
	params.Add("filter", fmt.Sprintf(`{"%s":"%s"}`, obj.RestKeyField, obj.RESTKey))
	params.Add("fields", strings.Join([]string{obj.RestKeyField, obj.TFIDField, resourceHashField}, ","))

	_, respBody, err := obj.requestWithRetry(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s?%s", obj.urlBase(), params.Encode()),
		nil)

	if err != nil || respBody == nil {
		return "", err
	}

	var raw []json.RawMessage
	if err = json.Unmarshal(respBody, &raw); err != nil {
		return "", err
	}

	if len(raw) > 1 {
		return "", fmt.Errorf("failed to lookup hash for %s %s: object is not unique", obj.ObjectType, obj.RESTKey)
	}

	for _, r := range raw {
		obj_ := obj.Clone()
		if err = obj_.Populate(r); err != nil {
			return "", err
		}

		m, err := obj_.RawJson.ToInterfaceMap()
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
