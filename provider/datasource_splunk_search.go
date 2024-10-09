package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider/splunk"
	"github.com/tivo/terraform-provider-splunk-itsi/util"
)

const (
	searchDefaultEarliestTime = "-4h"
	searchDefaultLatestTime   = "now"
	searchDefaultUser         = "nobody"
	searchDefaultApp          = "search"
	searchDefaultTimeout      = 60
	searchDefaultConcurrency  = 2

	searchErrorIncompleteResults = "Splunk search returned incomplete results"
	searchErrorNoResults         = "Splunk search returned no results"
)

var (
	_ datasource.DataSource              = &dataSourceSplunkSearch{}
	_ datasource.DataSourceWithConfigure = &dataSourceSplunkSearch{}

	splunkSearchLimiter *util.Limiter
	splunkSearchClients models.IHttpClients
)

func init() {
	splunkSearchClients = models.InitHttpClients()
}

func InitSplunkSearchLimiter(concurrency int) {
	splunkSearchLimiter = util.NewLimiter(concurrency)
}

type searchModel struct {
	Query               types.String `tfsdk:"query"`
	SplunkUser          types.String `tfsdk:"splunk_user"`
	SplunkApp           types.String `tfsdk:"splunk_app"`
	EarliestTime        types.String `tfsdk:"earliest_time"`
	LatestTime          types.String `tfsdk:"latest_time"`
	AllowNoResults      types.Bool   `tfsdk:"allow_no_results"`
	AllowPartialResults types.Bool   `tfsdk:"allow_partial_results"`
	Timeout             types.Int64  `tfsdk:"timeout"`
}

type dataSourceSplunkSearch struct {
	client models.ClientConfig
}

type dataSourceSplunkSearchModel struct {
	Search            types.Set    `tfsdk:"search"`
	JoinFields        types.Set    `tfsdk:"join_fields"`
	SearchConcurrency types.Int64  `tfsdk:"search_concurrency"`
	Results           types.String `tfsdk:"results"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func NewDataSourceSplunkSearch() datasource.DataSource {
	return &dataSourceSplunkSearch{}
}

func (d *dataSourceSplunkSearch) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	configureDataSourceClient(ctx, datasourceNameSplunkSearch, req, &d.client, resp)
}

func (d *dataSourceSplunkSearch) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	configureDataSourceMetadata(req, resp, datasourceNameSplunkSearch)
}

func (d *dataSourceSplunkSearch) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to retrieve the results of a Splunk search.",
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx),
			"search": schema.SetNestedBlock{
				MarkdownDescription: "Search to be executed",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"query": schema.StringAttribute{
							MarkdownDescription: "The search language string to execute, taking results from the local and remote servers. See https://dev.splunk.com/enterprise/docs/devtools/customsearchcommands/",
							Required:            true,
							Validators:          validateSearchQuery(),
						},
						"splunk_user": schema.StringAttribute{
							MarkdownDescription: "The Splunk user in the context of which the search query should be performed",
							Optional:            true,
						},
						"splunk_app": schema.StringAttribute{
							MarkdownDescription: "The Splunk app in which the search query should be performed.",
							Optional:            true,
						},
						"earliest_time": schema.StringAttribute{
							MarkdownDescription: "Specify a time string. Sets the earliest (inclusive), respectively, time bounds for the search.",
							Optional:            true,
						},
						"latest_time": schema.StringAttribute{
							MarkdownDescription: "Specify a time string. Sets the latest (exclusive), respectively, time bounds for the search.",
							Optional:            true,
						},
						"allow_no_results": schema.BoolAttribute{
							MarkdownDescription: "Indicates whether the search is allowed to return no results. When set to false, the search job fails if no results are returned.",
							Optional:            true,
						},
						"allow_partial_results": schema.BoolAttribute{
							MarkdownDescription: "Indicates whether the search job can proceed to provide partial results if a search peer fails. When set to false, the search job fails if a search peer providing results for the search job fails.",
							Optional:            true,
						},
						"timeout": schema.Int64Attribute{
							MarkdownDescription: "HTTP timeout in seconds. 0 means no timeout.",
							Optional:            true,
							DeprecationMessage:  "This attribute is deprecated and will be removed in a future release. Use the `timeouts` block instead.",
						},
					},
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			"join_fields": schema.SetAttribute{
				MarkdownDescription: "A set of strings, represents field names results will be FULL joined by.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"search_concurrency": schema.Int64Attribute{
				MarkdownDescription: "Number of searches that could be run in parallel per data source.",
				Optional:            true,
			},
			"results": schema.StringAttribute{
				MarkdownDescription: "Results of the search encoded as a JSON string. The data structure is a list of maps, where field names are keys.",
				Computed:            true,
			},
		},
	}
}

func validateSearchQuery() []validator.String {
	return []validator.String{
		stringvalidator.RegexMatches(regexp.MustCompile(`^\s*(search|\|)`), "Search query must start with 'search' or '|'"),
	}
}

func (d *dataSourceSplunkSearch) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read splunk_search datasource")

	var state dataSourceSplunkSearchModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading splunk_search datasource", map[string]interface{}{"state": state})

	readTimeout, diags := state.Timeouts.Read(ctx, tftimeout.Read)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	if state.SearchConcurrency.IsNull() {
		state.SearchConcurrency = types.Int64Value(searchDefaultConcurrency)
	}

	var searchModels []searchModel
	if resp.Diagnostics.Append(state.Search.ElementsAs(ctx, &searchModels, false)...); resp.Diagnostics.HasError() {
		return
	}

	var searches []SplunkSearch
	for _, search := range searchModels {
		if search.EarliestTime.IsNull() {
			search.EarliestTime = types.StringValue(searchDefaultEarliestTime)
		}

		if search.LatestTime.IsNull() {
			search.LatestTime = types.StringValue(searchDefaultLatestTime)
		}

		if search.SplunkApp.IsNull() {
			search.SplunkApp = types.StringValue(searchDefaultApp)
		}

		if search.SplunkUser.IsNull() {
			search.SplunkUser = types.StringValue(searchDefaultUser)
		}

		if search.Timeout.IsNull() {
			search.Timeout = types.Int64Value(searchDefaultTimeout)
		}

		timeoutSeconds := int(min(search.Timeout.ValueInt64(), int64(readTimeout.Seconds())))

		searches = append(searches, SplunkSearch{
			Query:               search.Query.ValueString(),
			AllowNoResults:      search.AllowNoResults.ValueBool(),
			AllowPartialResults: search.AllowPartialResults.ValueBool(),
			EarliestTime:        search.EarliestTime.ValueString(),
			LatestTime:          search.LatestTime.ValueString(),
			App:                 search.SplunkApp.ValueString(),
			User:                search.SplunkUser.ValueString(),
			Timeout:             timeoutSeconds,
		})
	}

	var joinFields []string
	if resp.Diagnostics.Append(state.JoinFields.ElementsAs(ctx, &joinFields, false)...); resp.Diagnostics.HasError() {
		return
	}

	sort.Strings(joinFields)

	splunkreq := NewSplunkRequest(d.client, searches, int(state.SearchConcurrency.ValueInt64()), joinFields, true, " ")

	results, diags := splunkreq.Run(ctx)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	jsonResults, err := json.Marshal(results)
	if err != nil {
		resp.Diagnostics.AddError("Unable to marshal JSON for Splunk search results", err.Error())
		return
	}

	state.Results = types.StringValue(string(jsonResults))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading splunk_search datasource", map[string]any{"success": true})

}

type SplunkSearch struct {
	Query               string
	User                string
	App                 string
	EarliestTime        string
	LatestTime          string
	AllowPartialResults bool
	AllowNoResults      bool
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
						diags.AddError("Missing join field", fmt.Sprintf("Missing field %s expected in the Splunk query result, as a join field.", f))
					}
				}
				// look for a search result with the same key and merge them together
				if keyJoinedRow, ok := keyJoinedResultsMap[key]; ok {
					for k, v := range searchResultRow {
						if v_, exists := keyJoinedRow[k]; exists {
							if v != v_ {
								diags.AddError("Splunk search result values overlap", fmt.Sprintf("Splunk search result values overlap on field %s: values %v and %v.\nMake sure your splunk searches do not share the same field names.", k, v, v_))
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
			diags.AddError("Couldn't login to Splunk", err.Error())
			return
		}
	}

	rows, _, err := conn.Search(ctx, sr.client.RetryPolicy, s.Query, params)
	if err != nil {
		diags.AddError("Splunk search failed", err.Error())
		return
	}

	if len(rows) == 0 {
		if s.AllowNoResults {
			diags.AddWarning(searchErrorNoResults, searchErrorNoResults)
		} else {
			diags.AddError(searchErrorNoResults, searchErrorNoResults)
		}
	} else if !rows[len(rows)-1].LastRow {
		if s.AllowPartialResults {
			diags.AddWarning(searchErrorIncompleteResults, searchErrorIncompleteResults)
		} else {
			diags.AddError(searchErrorIncompleteResults, searchErrorIncompleteResults)
		}
	}

	if diags.HasError() {
		return nil, diags
	}

	results = make([]map[string]splunk.Value, len(rows))
	for i, r := range rows {
		resultRow := make(map[string]splunk.Value)
		for k, v := range r.Result {
			resultRow[k] = v
		}

		results[i] = resultRow
	}

	return
}

func (sr *SplunkRequest) ID() string {
	b, _ := json.Marshal(sr.searches)
	return util.Sha256(b)
}
