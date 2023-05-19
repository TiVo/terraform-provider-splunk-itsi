package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/lestrrat-go/backoff/v2"
	"gopkg.in/yaml.v3"

	assert "github.com/stretchr/testify/assert"
	mock_models "github.com/tivo/terraform-provider-splunk-itsi/models"
)

var _KPI_BASE_SEARCH = "kpi_base_search"
var _KPI_THRESHOLD_TEMPLATE = "kpi_threshold_template"
var _SERVICE = "service"
var _SERVICE_RESOURCE_LABEL = "test_service_resource"
var _PATH_PREFIX = "/servicesNS/nobody/SA-ITOA/itoa_interface/"

var _DATA_FOLDER = "unit_test_data/resource_service/"

func testServiceProvider() *schema.Provider {
	return &schema.Provider{
		ConfigureContextFunc: func(c context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
			clientConfigStub := mock_models.ClientConfig{}
			clientConfigStub.RetryPolicy = backoff.Null()
			return clientConfigStub, nil
		},
		ResourcesMap: map[string]*schema.Resource{
			"test_service_resource": ResourceService(),
		},
	}
}

type TestServiceResourceValidationTestCase struct {
	Description   string
	Config        string
	ExpectedError string `yaml:"expected_error"`
}

func TestServiceResourceValidation(t *testing.T) {

	providerFactories := map[string]func() (*schema.Provider, error){
		"test": func() (*schema.Provider, error) { return testServiceProvider(), nil },
	}

	var resourceValidationDataProvider []TestServiceResourceValidationTestCase
	parseYaml(t, "validation_data_provider.yaml", &resourceValidationDataProvider)

	for _, test := range resourceValidationDataProvider {
		resource.UnitTest(t, resource.TestCase{
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config:      test.Config,
					ExpectError: regexp.MustCompile(test.ExpectedError),
				},
			},
		})
		mock_models.TearDown()
	}
}

type TestServiceResourceCreateTestCase struct {
	Description                string
	ResourceName               string            `yaml:"resource_name"`
	Config                     string            `yaml:"config"`
	InputBaseSearchId          string            `yaml:"input_base_search_id"`
	InputBaseSearch            string            `yaml:"input_base_search"`
	InputThresholdTemplateId   string            `yaml:"input_threshold_template_id"`
	InputThresholdTemplate     string            `yaml:"input_threshold_template"`
	ExpectedPostBody           string            `yaml:"expected_service_post_body"`
	ExpectedApiCallsCount      int               `yaml:"expected_api_calls_count"`
	ExpectedResourceAttributes map[string]string `yaml:"expected_resource_attributes"`
	ServiceIdToSet             string            `yaml:"service_id_to_set"`
}

func TestServiceResourceCreate(t *testing.T) {

	providerFactories := map[string]func() (*schema.Provider, error){
		"test": func() (*schema.Provider, error) { return testServiceProvider(), nil },
	}

	var serviceResourceCreateDataProvider []TestServiceResourceCreateTestCase
	parseYaml(t, "creation_data_provider.yaml", &serviceResourceCreateDataProvider)

	for _, test := range serviceResourceCreateDataProvider {
		t.Log("=== RUNNING ", t.Name(), ": TEST CASE ", test.Description)

		mock_models.GenerateResourceKey = func() (string, error) {
			return test.ServiceIdToSet, nil
		}

		GenerateUUID = func(internalID string) (string, error) {
			return internalID, nil
		}

		resource.UnitTest(t, resource.TestCase{
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: test.Config,
					Check: func(s *terraform.State) error {
						resourceName := _SERVICE_RESOURCE_LABEL + "." + test.ResourceName

						for attrKey, attrValue := range test.ExpectedResourceAttributes {
							if err := resource.TestCheckResourceAttr(resourceName, attrKey, attrValue)(s); err != nil {
								return fmt.Errorf("Check %s->%s error: %s", attrKey, attrValue, err)
							}
						}
						return nil
					},
					PreConfig: func() {
						actualApiCallCount := 0
						apiCallStackMsg := ""

						mock_models.Do = func(req *http.Request) (*http.Response, error) {
							var response io.ReadCloser
							var err error = nil
							var path string = strings.TrimPrefix(req.URL.Path, _PATH_PREFIX)

							apiCallStackMsg += fmt.Sprintf("%d. [%s] %s\n: %s\n", actualApiCallCount, req.Method,
								req.URL.Path, string(debug.Stack()))
							actualApiCallCount++

							switch method, body := req.Method, req.Body; {
							// for destroy after the end of the test plan
							case method == "DELETE" && path == _SERVICE+"/"+test.ServiceIdToSet:
								response = io.NopCloser(bytes.NewReader([]byte("{success}")))
								assert.Exactly(t, test.ExpectedApiCallsCount, actualApiCallCount, apiCallStackMsg)

							case method == "GET" && path == _KPI_BASE_SEARCH+"/"+test.InputBaseSearchId:
								response = io.NopCloser(bytes.NewReader([]byte(test.InputBaseSearch)))

							case method == "GET" && path == _KPI_THRESHOLD_TEMPLATE+"/"+test.InputThresholdTemplateId:
								response = io.NopCloser(bytes.NewReader([]byte(test.InputThresholdTemplate)))

							case method == "GET" && path == _SERVICE+"/"+test.ServiceIdToSet:
								var serviceAfterCreation map[string]interface{}
								json.Unmarshal([]byte(test.ExpectedPostBody), &serviceAfterCreation)
								serviceAfterCreation["_key"] = test.ServiceIdToSet
								newData, _ := json.Marshal(serviceAfterCreation)

								response = io.NopCloser(bytes.NewReader(newData))

							case method == "POST" && path == _SERVICE:
								mockAnswer := fmt.Sprintf("{\"_key\" : \"%s\"}", test.ServiceIdToSet)
								response = io.NopCloser(bytes.NewReader([]byte(mockAnswer)))

								actualServicePostBody := new(bytes.Buffer)
								actualServicePostBody.ReadFrom(body)

								assertServiceResourceJSONEq(t, test.ExpectedPostBody,
									actualServicePostBody.String(), "Service body mismatched")

							default:
								err = errors.New(fmt.Sprintf("Unexpected [%s] Call: %s %s", method, path, body))
							}
							return &http.Response{
								StatusCode: 200,
								Body:       response,
							}, err
						}
					},
				},
			},
		})
		mock_models.TearDown()
	}
}

func TestServiceCreatePopulateComputed(t *testing.T) {
	t.Log("=== RUNNING ", t.Name(), ": TEST CASE verifying shkpi & id attributes are set")

	serviceIdToSet := "test_service_create_populate_computed"
	serviceResourceName := _SERVICE_RESOURCE_LABEL + ".service_create_test"

	providerFactories := map[string]func() (*schema.Provider, error){
		"test": func() (*schema.Provider, error) { return testServiceProvider(), nil },
	}

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
                        resource "test_service_resource" "service_create_test" {
                            title    = "TEST"
                            description = "Terraform unit test"
							enabled = false
                        }
                    `,
				PreConfig: func() {
					mock_models.Do = func(req *http.Request) (*http.Response, error) {
						var response io.ReadCloser
						var err error = nil
						var mock_answer = ""

						switch path, method, body := req.URL.Path, req.Method, req.Body; {
						// for destroy after the end of the test plan
						case method == "DELETE" && strings.Contains(path, _SERVICE+"/"+serviceIdToSet):
							mock_answer = "{success}"
							response = io.NopCloser(bytes.NewReader([]byte(mock_answer)))

						case method == "GET" && strings.Contains(path, _SERVICE+"/"+serviceIdToSet):
							mock_answer = `
                                {
                                   "_key":"test_service_create_populate_computed",
                                   "description":"Terraform unit test",
                                   "enabled":0,
                                   "entity_rules":[],
                                   "kpis":[
                                      {
                                        "title": "ServiceHealthScore",
                                        "unit": "",
                                        "gap_severity_value": "-1",
                                        "fill_gaps": "null_value",
                                        "search_alert_earliest": "15",
                                        "type": "service_health",
                                        "_owner": "nobody",
                                        "adaptive_thresholds_is_enabled": false,
                                        "source": "",
                                        "urgency": "11",
                                        "datamodel_filter": [],
                                        "alert_lag": "30",
                                        "kpi_base_search": "",
                                        "base_search": "get_full_itsi_summary_service_health_events(test)",
                                        "search_alert": "",
                                        "search": "get_full_itsi_summary_service_health_events(test)",
                                        "_key": "SHKPI-000000000000"
                                      }
                                   ],
                                   "object_type":"service",
                                   "sec_grp":"default_itsi_security_group",
                                   "service_tags":{
                                      "tags":[]
                                   },
                                   "services_depends_on":[],
                                   "title":"TEST"
                                }
                            `
							response = io.NopCloser(bytes.NewReader([]byte(mock_answer)))

						case method == "POST" && strings.Contains(path, _SERVICE):
							mock_answer := fmt.Sprintf("{\"_key\" : \"%s\"}", serviceIdToSet)
							response = io.NopCloser(bytes.NewReader([]byte(mock_answer)))

						default:
							err = errors.New(fmt.Sprintf("Unexpected [%s] Call: %s %s", method, path, body))
						}
						return &http.Response{
							StatusCode: 200,
							Body:       response,
						}, err
					}
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(serviceResourceName, "id", serviceIdToSet),
					resource.TestCheckResourceAttr(serviceResourceName, "shkpi_id", "SHKPI-000000000000"),
				),
			},
		},
	})
}

type TestServiceUpdateDataProvider struct {
	Description                  string
	ResourceName                 string            `yaml:"resource_name"`
	CreateConfig                 string            `yaml:"create_config"`
	UpdateConfig                 string            `yaml:"update_config"`
	InputDependenciesResponses   map[string]string `yaml:"input_dependencies_responses"`
	InputGetServiceBody          string            `yaml:"input_get_service_body"`
	ExpectedUpdateServicePutBody string            `yaml:"expected_put_service_body"`
	ServiceIdToSet               string            `yaml:"service_id_to_set"`
}

func TestServiceUpdate(t *testing.T) {
	providerFactories := map[string]func() (*schema.Provider, error){
		"test": func() (*schema.Provider, error) { return testServiceProvider(), nil },
	}

	GenerateUUID = func(internalID string) (string, error) {
		return internalID, nil
	}

	var resourceUpdateDataProvider []TestServiceUpdateDataProvider
	parseYaml(t, "update_data_provider.yaml", &resourceUpdateDataProvider)

	for _, test := range resourceUpdateDataProvider {
		t.Log("=== RUNNING ", t.Name(), ": TEST CASE ", test.Description)

		resource.UnitTest(t, resource.TestCase{
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: test.CreateConfig,
					PreConfig: func() {
						mock_models.Do = func(req *http.Request) (*http.Response, error) {
							var err error = nil
							var response io.ReadCloser

							path := strings.TrimPrefix(req.URL.Path, _PATH_PREFIX)
							inputDependencyBody, isDependency := test.InputDependenciesResponses[path]

							switch method, body := req.Method, req.Body; {
							case isDependency:
								assert.Exactly(t, "GET", method, "dependency expected to be readonly")
								response = io.NopCloser(bytes.NewReader([]byte(inputDependencyBody)))

							case method == "GET" && path == _SERVICE+"/"+test.ServiceIdToSet:
								response = io.NopCloser(bytes.NewReader([]byte(test.InputGetServiceBody)))

							case method == "POST":
								mock_answer := fmt.Sprintf("{\"_key\" : \"%s\"}", test.ServiceIdToSet)
								response = io.NopCloser(bytes.NewReader([]byte(mock_answer)))

							case method == "DELETE" && strings.Contains(path, _SERVICE+"/"+test.ServiceIdToSet):
								response = io.NopCloser(bytes.NewReader([]byte("{success}")))

							default:
								err = errors.New(fmt.Sprintf("Unexpected [%s] Call: %s %s", method, path, body))
							}
							return &http.Response{
								StatusCode: 200,
								Body:       response,
							}, err
						}
					},
				},
				{
					Config: test.UpdateConfig,
					PreConfig: func() {
						inputCreatedServiceBody := test.InputGetServiceBody
						mock_models.Do = func(req *http.Request) (*http.Response, error) {
							var err error = nil
							var response io.ReadCloser

							path := strings.TrimPrefix(req.URL.Path, _PATH_PREFIX)
							inputDependencyBody, isDependency := test.InputDependenciesResponses[path]

							switch method, body := req.Method, req.Body; {
							case isDependency:
								assert.Exactly(t, "GET", method, "dependency expected to be readonly")
								response = io.NopCloser(bytes.NewReader([]byte(inputDependencyBody)))

							case method == "GET" && path == _SERVICE+"/"+test.ServiceIdToSet:
								response = io.NopCloser(bytes.NewReader([]byte(inputCreatedServiceBody)))

							case method == "DELETE" && strings.Contains(path, _SERVICE+"/"+test.ServiceIdToSet):
								response = io.NopCloser(bytes.NewReader([]byte("{success}")))

							case method == "POST" && path == _SERVICE:
								mock_answer := fmt.Sprintf("{\"_key\" : \"%s\"}", test.ServiceIdToSet)
								response = io.NopCloser(bytes.NewReader([]byte(mock_answer)))

							case method == "PUT" && path == _SERVICE+"/"+test.ServiceIdToSet:
								mock_answer := fmt.Sprintf("{\"_key\" : \"%s\"}", test.ServiceIdToSet)
								response = io.NopCloser(bytes.NewReader([]byte(mock_answer)))

								actualServiceBody := new(bytes.Buffer)
								actualServiceBody.ReadFrom(body)

								assertServiceResourceJSONEq(t, test.ExpectedUpdateServicePutBody,
									actualServiceBody.String(), "expected body mismatched")
								inputCreatedServiceBody = actualServiceBody.String()
							default:
								err = errors.New(fmt.Sprintf("Unexpected [%s] Call: %s %s", method, path, body))
							}
							return &http.Response{
								StatusCode: 200,
								Body:       response,
							}, err
						}
					},
				},
			},
		})
		mock_models.TearDown()
	}
}

type TestCacheHitTestCase struct {
	Description               string
	Config                    string `yaml:"config"`
	CachedBaseSearchId        string `yaml:"cached_base_search_id"`
	CachedBaseSearch          string `yaml:"cached_base_search"`
	CachedThresholdTemplateId string `yaml:"cached_threshold_template_id"`
	CachedThresholdTemplate   string `yaml:"cached_threshold_template"`
	InputGetBody              string `yaml:"input_service_get_body"`
	ServiceIdToSet            string `yaml:"service_id_to_set"`
}

func TestCacheHit(t *testing.T) {
	providerFactories := map[string]func() (*schema.Provider, error){
		"test": func() (*schema.Provider, error) { return testServiceProvider(), nil },
	}

	GenerateUUID = func(internalID string) (string, error) {
		return internalID, nil
	}

	var resourceCacheHitDataProvider []TestCacheHitTestCase
	parseYaml(t, "cache_hit_data_provider.yaml", &resourceCacheHitDataProvider)

	for _, test := range resourceCacheHitDataProvider {
		t.Log("=== RUNNING ", t.Name(), ": TEST CASE ", test.Description)
		serviceIdToSet := test.ServiceIdToSet

		// set up cached objects
		kpiThresholdTemplateBase := mock_models.NewBase(mock_models.ClientConfigStub,
			test.CachedThresholdTemplateId, "test_kpi_threshold_template", "kpi_threshold_template")
		mock_models.Do = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(test.CachedThresholdTemplate))),
			}, nil
		}
		kpiThresholdTemplateBase.Read(mock_models.ContextStub)

		kpiBaseSearchBase := mock_models.NewBase(mock_models.ClientConfigStub,
			test.CachedBaseSearchId, "test_kpi_base_search", "kpi_base_search")
		mock_models.Do = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(test.CachedBaseSearch))),
			}, nil
		}
		kpiBaseSearchBase.Read(mock_models.ContextStub)

		resource.UnitTest(t, resource.TestCase{
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: test.Config,
					PreConfig: func() {
						mock_models.Do = func(req *http.Request) (*http.Response, error) {
							// verified that no threshold_template or kpi_base_search requested
							var response io.ReadCloser
							var err error = nil
							var path string = strings.TrimPrefix(req.URL.Path, _PATH_PREFIX)

							switch method, body := req.Method, req.Body; {
							case method == "DELETE" && path == _SERVICE+"/"+serviceIdToSet:
								response = io.NopCloser(bytes.NewReader([]byte("{success}")))
							case method == "POST" && path == _SERVICE:
								mock_answer := fmt.Sprintf("{\"_key\" : \"%s\"}", serviceIdToSet)
								response = io.NopCloser(bytes.NewReader([]byte(mock_answer)))

							case method == "GET" && path == _SERVICE+"/"+serviceIdToSet:
								response = io.NopCloser(bytes.NewReader([]byte(test.InputGetBody)))
							default:
								err = errors.New(fmt.Sprintf("Unexpected [%s] Call: %s %s", method, path, body))
							}
							return &http.Response{
								StatusCode: 200,
								Body:       response,
							}, err
						}
					},
				},
			},
		})
		mock_models.TearDown()
	}
}

type TestSharedDependenciesConcurrentTestCase struct {
	Description         string
	Config              string
	BaseSearchId        string `yaml:"base_search_id"`
	BaseSearch          string `yaml:"base_search"`
	ThresholdTemplateId string `yaml:"threshold_template_id"`
	ThresholdTemplate   string `yaml:"threshold_template"`
}

func TestSharedDependenciesConcurrentCallCheck(t *testing.T) {
	providerFactories := map[string]func() (*schema.Provider, error){
		"test": func() (*schema.Provider, error) { return testServiceProvider(), nil },
	}

	GenerateUUID = func(internalID string) (string, error) {
		return internalID, nil
	}

	mock_models.InitItsiApiLimiter(10)
	defer mock_models.InitItsiApiLimiter(1)
	var mu sync.Mutex
	var resourceSharedDependenciesConcurrentProvider []TestSharedDependenciesConcurrentTestCase
	parseYaml(t, "concurrent_cache_hit_data_provider.yaml", &resourceSharedDependenciesConcurrentProvider)
	for _, test := range resourceSharedDependenciesConcurrentProvider {
		t.Log("=== RUNNING ", t.Name(), ": TEST CASE ", test.Description)
		mockAnswers := map[string]string{}

		actualThresholdTemplateApiCalls := 0
		actualKpiBaseSearchApiCalls := 0

		resource.UnitTest(t, resource.TestCase{
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: test.Config,
					PreConfig: func() {
						mock_models.Do = func(req *http.Request) (*http.Response, error) {
							var response io.ReadCloser
							var err error = nil
							var path string = strings.TrimPrefix(req.URL.Path, _PATH_PREFIX)
							switch method, body := req.Method, req.Body; {
							case method == "DELETE":
								assert.Contains(t, path, _SERVICE, "Unexpected  API call: ", method, path)
								response = io.NopCloser(bytes.NewReader([]byte("{success}")))
							case method == "POST" && path == _SERVICE:

								// Intention of this test is to check only amount of dependency calls which are concurrent.
								// So here the future mocked GET answers is populated from POST, just to emulate successful flow.
								var postBodyJsonInterface map[string]interface{}
								buf := new(bytes.Buffer)
								buf.ReadFrom(body)
								postServiceBody := buf.String()

								if err = json.Unmarshal([]byte(postServiceBody), &postBodyJsonInterface); err != nil {
									return nil, err
								}

								// mock that service-returned _key == title
								serviceIdToSet, _ := postBodyJsonInterface["title"].(string)
								postBodyJsonInterface["_key"] = serviceIdToSet

								postBodyJsonBytes, _ := json.Marshal(postBodyJsonInterface)
								mu.Lock()
								mockAnswers[_SERVICE+"/"+serviceIdToSet] = string(postBodyJsonBytes)
								mu.Unlock()
								mockedPostServerAnswer := fmt.Sprintf("{\"_key\" : \"%s\"}", serviceIdToSet)
								response = io.NopCloser(bytes.NewReader([]byte(mockedPostServerAnswer)))

							case method == "GET" && mockAnswers[path] != "":
								response = io.NopCloser(bytes.NewReader([]byte(mockAnswers[path])))

							case method == "GET" && path == _KPI_BASE_SEARCH+"/"+test.BaseSearchId:
								actualKpiBaseSearchApiCalls++
								assert.Equal(t, 1, actualKpiBaseSearchApiCalls, "Extra kpi base search call")

								response = io.NopCloser(bytes.NewReader([]byte(test.BaseSearch)))
							case method == "GET" && path == _KPI_THRESHOLD_TEMPLATE+"/"+test.ThresholdTemplateId:
								actualThresholdTemplateApiCalls++
								assert.Equal(t, 1, actualThresholdTemplateApiCalls, "Extra threshold_template search call")

								response = io.NopCloser(bytes.NewReader([]byte(test.ThresholdTemplate)))

							default:
								err = errors.New(fmt.Sprintf("Unexpected [%s] Call: %s %s", method, path, body))
							}
							return &http.Response{
								StatusCode: 200,
								Body:       response,
							}, err
						}
					},
				},
			},
		})
		mock_models.TearDown()
	}
}

func parseYaml(t *testing.T, fileName string, testStruct interface{}) {
	t.Helper()

	yamlInput, err := os.ReadFile(_DATA_FOLDER + fileName)
	assert.NoError(t, err)

	err = yaml.Unmarshal(yamlInput, testStruct)
	assert.NoError(t, err)
}

func assertServiceResourceJSONEq(t *testing.T, expected string, actual string, msgAndArgs ...interface{}) bool {
	t.Helper()

	var expectedJSONAsInterface, actualJSONAsInterface map[string]interface{}

	if err := json.Unmarshal([]byte(expected), &expectedJSONAsInterface); err != nil {
		return assert.Fail(t, fmt.Sprintf("Expected value ('%s') is not valid json.\nJSON parsing error: '%s'",
			expected, err.Error()), msgAndArgs...)
	}

	if err := json.Unmarshal([]byte(actual), &actualJSONAsInterface); err != nil {
		return assert.Fail(t, fmt.Sprintf("Input ('%s') needs to be valid json.\nJSON parsing error: '%s'",
			actual, err.Error()), msgAndArgs...)
	}

	if kpis, ok := expectedJSONAsInterface["kpis"].([]interface{}); ok {
		sort.Slice(kpis, func(i, j int) bool {
			return kpis[i].(map[string]interface{})["_key"].(string) > kpis[j].(map[string]interface{})["_key"].(string)
		})
	}
	if kpis, ok := actualJSONAsInterface["kpis"].([]interface{}); ok {
		sort.Slice(kpis, func(i, j int) bool {
			return kpis[i].(map[string]interface{})["_key"].(string) > kpis[j].(map[string]interface{})["_key"].(string)
		})
	}
	delete(actualJSONAsInterface, "_tf_hash")
	delete(expectedJSONAsInterface, "_tf_hash")
	return assert.Equal(t, expectedJSONAsInterface, actualJSONAsInterface, msgAndArgs...)
}
