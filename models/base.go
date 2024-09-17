package models

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/lestrrat-go/backoff/v2"
)

type Base struct {
	Splunk    ClientConfig
	RetryFunc RetryFunc
}

type RetryFunc func(ctx context.Context, method string, statusCode int, responseBody []byte, requestErr error) (shouldRetry bool, newStatusCode int, newBody []byte, err error)

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

			if shouldRetry, newStatus, newBody, err := b.RetryFunc(ctx, method, statusCode, responseBody, requestErr); !shouldRetry {
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
		success = resp.StatusCode >= 200 && resp.StatusCode < 300
	}

	if !success {
		return resp.StatusCode, nil, fmt.Errorf("%v error: %v \n%s", method, resp.Status, responseBody)
	}

	return
}
