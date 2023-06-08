package splunk

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/lestrrat-go/backoff/v2"
)

const (
	bufferDefaultSize = 32 * 1024   // 32KB
	bufferMaxSize     = 1024 * 1024 // 1MB

)

type Value interface{}

type Row struct {
	Preview bool             `json:"preview"`
	Offset  int              `json:"offset"`
	Result  map[string]Value `json:"result"`
	LastRow bool             `json:"lastrow"`
}

type Rows []Row

func NewRow() (ret Row) {
	ret = Row{}
	ret.Result = make(map[string]Value)
	return
}

func parseLine(line string) (r Row, err error) {
	fbuf := bytes.NewBufferString(line)
	dec := json.NewDecoder(fbuf)
	r = NewRow()
	return r, dec.Decode(&r)
}

func (conn SplunkConnection) Search(ctx context.Context, boPolicy backoff.Policy, searchString string, params ...map[string]string) (rows []Row, events []string, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	data := make(url.Values)
	data.Add("search", searchString)
	data.Add("output_mode", "json")
	data.Add("preview", "false")

	for _, m := range params {
		for k, v := range m {
			data.Add(k, v)
		}
	}

	var responseBody []byte
	bo := boPolicy.Start(ctx)
	attempt := 1

	for backoff.Continue(bo) {
		url := fmt.Sprintf("%s/servicesNS/%s/%s/search/jobs/export", conn.BaseURL, conn.SplunkUser, conn.SplunkApp)
		start := time.Now()

		statusCode, err := func() (statusCode int, err error) {
			response, err := conn.httpCallWithContext(ctx, url, http.MethodPost, &data)
			if err != nil {
				return
			}
			defer response.Body.Close()
			responseBody, err = io.ReadAll(response.Body)
			statusCode = response.StatusCode
			if statusCode != 200 {
				err = fmt.Errorf("splunk search error: %v", response.Status)
			}
			return
		}()
		status := http.StatusText(statusCode)
		tflog.Trace(ctx, fmt.Sprintf("POST %v (%v): %v %v [%s]", url, attempt, statusCode, status, time.Since(start).String()), map[string]interface{}{"splunk_search": searchString, "params": params})
		if err != nil {
			if statusCode == 400 || statusCode == 401 || statusCode == 403 || statusCode == 404 {
				return nil, nil, fmt.Errorf("splunk search error: %v", status)
			}
			attempt++
			continue
		}
		break
	}

	scanner := bufio.NewScanner(bytes.NewReader(responseBody))
	scanner.Buffer(make([]byte, bufferDefaultSize), bufferMaxSize)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}
		var r Row
		if r, err = parseLine(line); err == nil {
			rows = append(rows, r)
			events = append(events, line)
		} else {
			return
		}

	}
	if err = scanner.Err(); err != nil {
		return
	}
	err = ctx.Err()
	return
}

func (conn SplunkConnection) SearchStream(searchString string, params ...map[string]string) (events chan *Row, err error) {
	data := make(url.Values)
	data.Add("search", searchString)
	data.Add("output_mode", "json")

	for _, m := range params {
		for k, v := range m {
			data.Add(k, v)
		}
	}

	response, err := conn.httpCall(fmt.Sprintf("%s/servicesNS/nobody/search/search/jobs/export", conn.BaseURL), "POST", &data)
	if err != nil {
		return nil, err
	}

	events = make(chan *Row, 50)

	go func() { // Using closures here (events,response)
		scanner := bufio.NewScanner(response.Body)
		defer response.Body.Close()

		for scanner.Scan() {
			line := scanner.Text()
			if row, err := parseLine(line); err != nil {
				fmt.Printf("Could not decode line: '%s' %s\n", line, err)
			} else {
				events <- &row
			}
		}
		events <- nil //Signal EOF
	}()

	return events, err
}
