package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/splunk"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

var splunkSearchLimiter *util.Limiter
var splunkSearchClients models.IHttpClients

func init() {
	splunkSearchClients = models.InitHttpClients()
}

func InitSplunkSearchLimiter(concurrency int) {
	splunkSearchLimiter = util.NewLimiter(concurrency)
}

func DatasourceSplunkSearch() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to retrieve the results of a Splunk search.",
		ReadContext: splunkSearchRead,
		Timeouts: &schema.ResourceTimeout{
			Read: schema.DefaultTimeout(20 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"search": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"query": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The search language string to execute, taking results from the local and remote servers. See https://dev.splunk.com/enterprise/docs/devtools/customsearchcommands/",
							ValidateDiagFunc: func(v_ interface{}, p cty.Path) diag.Diagnostics {
								v := strings.TrimSpace(v_.(string))
								var diags diag.Diagnostics

								if strings.HasPrefix(v, "|") || strings.HasPrefix(v, "search") {
									return diags
								}
								diag := diag.Diagnostic{
									Severity: diag.Error,
									Summary:  "wrong query",
									Detail:   "The query must start either with 'search' or '|'.",
								}
								diags = append(diags, diag)
								return diags
							},
						},
						"splunk_user": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "nobody",
							Description: "The Splunk user in the context of which the search query should be performed",
						},
						"splunk_app": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "search",
							Description: "The Splunk app in which the search query should be performed.",
						},
						"earliest_time": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "-4h",
							Description: "Specify a time string. Sets the earliest (inclusive), respectively, time bounds for the search.",
						},
						"latest_time": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "now",
							Description: "Specify a time string. Sets the latest (exclusive), respectively, time bounds for the search.",
						},
						"allow_partial_results": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Indicates whether the search job can proceed to provide partial results if a search peer fails. When set to false, the search job fails if a search peer providing results for the search job fails.",
						},
						"timeout": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     60,
							Description: "HTTP timeout in seconds. 0 means no timeout.",
						},
					},
				},
			},
			"is_mv": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Multivalue mode. Indicates whether the search can return multivalue results.",
			},
			"mv_separator": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "\n",
				Description: "The separator string to be placed in between multivalue field elements.",
			},
			"join_fields": {
				Type:        schema.TypeSet,
				Optional:    true,
				Default:     nil,
				Description: "A set of strings, represents field names results will be FULL joined by.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"search_concurrency": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     2,
				Description: "Amount of searches that could be run in parallel per data source.",
			},
			"results": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeMap,
				},
				Description: "Represents search results. Format is a list of maps, where field names of the raw result are keys.",
			},
		},
	}
}

func splunkSearchRead(ctx context.Context, d *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(models.ClientConfig)
	ismv := d.Get("is_mv").(bool)
	searches := []SplunkSearch{}
	for _, e := range d.Get("search").(*schema.Set).List() {
		s := e.(map[string]interface{})
		searches = append(searches, SplunkSearch{
			Query:               s["query"].(string),
			User:                s["splunk_user"].(string),
			App:                 s["splunk_app"].(string),
			EarliestTime:        s["earliest_time"].(string),
			LatestTime:          s["latest_time"].(string),
			AllowPartialResults: s["allow_partial_results"].(bool),
			Timeout:             s["timeout"].(int),
		})
	}

	joinFields := []string{}
	for _, f := range d.Get("join_fields").(*schema.Set).List() {
		joinFields = append(joinFields, f.(string))
	}
	sort.Strings(joinFields)
	req := NewSplunkRequest(client, searches, d.Get("search_concurrency").(int), joinFields, ismv, d.Get("mv_separator").(string))

	results, diags := req.Run(ctx)
	d.SetId(req.ID())
	err := d.Set("results", results)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	return
}

type SplunkSearch struct {
	Query               string
	User                string
	App                 string
	EarliestTime        string
	LatestTime          string
	AllowPartialResults bool
	Timeout             int
}

type SplunkSearchResults struct {
	Results []map[string]splunk.Value
	Diags   diag.Diagnostics
}

type SplunkRequest struct {
	client      models.ClientConfig
	searches    []SplunkSearch
	joinFields  []string
	multivalue  bool
	mvseparator string
	limiter     *util.Limiter
}

func NewSplunkRequest(client models.ClientConfig, searches []SplunkSearch, concurrency int, joinFields []string, ismv bool, mvseparator string) *SplunkRequest {
	limiter := util.NewLimiter(concurrency)
	return &SplunkRequest{client, searches, joinFields, ismv, mvseparator, limiter}
}

func (sr *SplunkRequest) Run(ctx context.Context) (results []map[string]splunk.Value, diags diag.Diagnostics) {
	resultsCh := make(chan SplunkSearchResults, len(sr.searches))
	wg := sync.WaitGroup{}
	wg.Add(len(sr.searches))

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	for _, s := range sr.searches {
		go func(s SplunkSearch) {
			defer wg.Done()
			r, d := sr.Search(ctx, s)
			resultsCh <- SplunkSearchResults{r, d}
		}(s)
	}

	if len(sr.joinFields) == 0 {
		for searchResult := range resultsCh {
			//just appennding search results as there are no join fields...
			results = append(results, searchResult.Results...)
			diags = append(diags, searchResult.Diags...)
		}
	} else {
		keyJoinedResultsMap := map[string]map[string]splunk.Value{}
		resultsOrder := []string{}
		for searchResult := range resultsCh {
			diags = append(diags, searchResult.Diags...)
			for _, searchResultRow := range searchResult.Results {
				// build the key for the search result based on the join fields.
				key := ""
				for _, f := range sr.joinFields {
					if keyToJoin, ok := searchResultRow[f]; ok {
						key = fmt.Sprintf("%s-%v", key, keyToJoin)
					} else {
						diags = append(diags, diag.Errorf("Missing field %s expected in the Splunk query result, as a join field.", f)...)
					}
				}
				// look for a search result with the same key and merge them together
				if keyJoinedRow, ok := keyJoinedResultsMap[key]; ok {
					for k, v := range searchResultRow {
						if v_, exists := keyJoinedRow[k]; exists {
							if v != v_ {
								diags = append(diags, diag.Errorf("Splunk search result values overlap on field %s: values %v and %v.\nMake sure your splunk searches do not share the same field names.", k, v, v_)...)
							}
						}
						keyJoinedRow[k] = v
					}
					keyJoinedResultsMap[key] = keyJoinedRow
				} else {
					keyJoinedResultsMap[key] = searchResultRow
					resultsOrder = append(resultsOrder, key)
				}
			}
		}
		for _, key := range resultsOrder {
			results = append(results, keyJoinedResultsMap[key])
		}
	}

	if diags.HasError() {
		return nil, diags
	}

	return
}

func (sr *SplunkRequest) Search(ctx context.Context, s SplunkSearch) (results []map[string]splunk.Value, diags diag.Diagnostics) {
	splunkSearchLimiter.Acquire()
	sr.limiter.Acquire()

	defer splunkSearchLimiter.Release()
	defer sr.limiter.Release()

	client := sr.client
	client.Timeout = s.Timeout

	conn := splunk.SplunkConnection{
		BearerToken: sr.client.BearerToken,
		Username:    sr.client.User,
		Password:    sr.client.Password,
		BaseURL:     fmt.Sprintf("https://%s:%v", sr.client.Host, sr.client.Port),
		SplunkApp:   s.App,
		SplunkUser:  s.User,

		HttpClient: splunkSearchClients.Get(client).(*http.Client),
	}

	params := map[string]string{
		"earliest_time":         s.EarliestTime,
		"latest_time":           s.LatestTime,
		"allow_partial_results": strconv.FormatBool(s.AllowPartialResults),
	}

	if sr.client.BearerToken == "" {
		_, err := conn.Login()
		if err != nil {
			return nil, append(diags, diag.Errorf("Couldn't login to splunk: %s", err)...)
		}
	}

	rows, _, err := conn.Search(ctx, sr.client.RetryPolicy, s.Query, params)
	if err != nil {
		return nil, append(diags, diag.Errorf("Splunk search failed: %s", err)...)
	}

	if len(rows) == 0 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Splunk search returned no results",
		})
	} else if !rows[len(rows)-1].LastRow {
		if s.AllowPartialResults {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Splunk search returned incomplete results",
			})
		} else {
			return nil, append(diags, diag.Errorf("Splunk search returned incomplete results.")...)
		}
	}

	results = make([]map[string]splunk.Value, len(rows))
	for i, r := range rows {
		resultRow := make(map[string]splunk.Value)
		for k, v := range r.Result {
			values, ismv := v.([]interface{})

			switch {
			case ismv && !sr.multivalue:
				return nil, append(diags, diag.Errorf("Splunk search returned multivalue results on field %s, but multivalue mode is disabled.", k)...)
			case ismv && sr.multivalue:
				valuesStr, err := unpackResourceList[string](values)
				if err != nil {
					return nil, append(diags, diag.FromErr(err)...)
				}
				resultRow[k] = strings.Join(valuesStr, sr.mvseparator)
			default:
				resultRow[k] = v
			}
		}

		results[i] = resultRow
	}

	return
}

func (sr *SplunkRequest) ID() string {
	b, _ := json.Marshal(sr.searches)
	return util.Sha256(b)
}
