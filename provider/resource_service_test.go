package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/lestrrat-go/backoff/v2"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
	"testing"

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
								response = ioutil.NopCloser(bytes.NewReader([]byte("{success}")))
								assert.Exactly(t, test.ExpectedApiCallsCount, actualApiCallCount, apiCallStackMsg)

							case method == "GET" && path == _KPI_BASE_SEARCH+"/"+test.InputBaseSearchId:
								response = ioutil.NopCloser(bytes.NewReader([]byte(test.InputBaseSearch)))

							case method == "GET" && path == _KPI_THRESHOLD_TEMPLATE+"/"+test.InputThresholdTemplateId:
								response = ioutil.NopCloser(bytes.NewReader([]byte(test.InputThresholdTemplate)))

							case method == "GET" && path == _SERVICE+"/"+test.ServiceIdToSet:
								var serviceAfterCreation map[string]interface{}
								json.Unmarshal([]byte(test.ExpectedPostBody), &serviceAfterCreation)
								serviceAfterCreation["_key"] = test.ServiceIdToSet
								newData, _ := json.Marshal(serviceAfterCreation)

								response = ioutil.NopCloser(bytes.NewReader(newData))

							case method == "POST" && path == _SERVICE:
								mockAnswer := fmt.Sprintf("{\"_key\" : \"%s\"}", test.ServiceIdToSet)
								response = ioutil.NopCloser(bytes.NewReader([]byte(mockAnswer)))

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
							response = ioutil.NopCloser(bytes.NewReader([]byte(mock_answer)))

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
							response = ioutil.NopCloser(bytes.NewReader([]byte(mock_answer)))

						case method == "POST" && strings.Contains(path, _SERVICE):
							mock_answer := fmt.Sprintf("{\"_key\" : \"%s\"}", serviceIdToSet)
							response = ioutil.NopCloser(bytes.NewReader([]byte(mock_answer)))

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

	return assert.Equal(t, expectedJSONAsInterface, actualJSONAsInterface, msgAndArgs...)
}
