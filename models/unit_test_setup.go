//go:build test_setup
// +build test_setup

package models

import (
	"context"
	"errors"
	"fmt"
	"github.com/lestrrat-go/backoff/v2"
	"net/http"
)

type MockClients struct {
}

func (hc *MockClients) Get(config ClientConfig) IHttpClient {
	return &MockClient{}
}

type MockClient struct {
}

func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	if Do == nil {
		return nil, errors.New(fmt.Sprintf("Test setup misconfiguration: missed Do function implementation"))
	}
	return Do(req)
}

var ClientConfigStub ClientConfig
var ContextStub context.Context

func InitHttpClients() IHttpClients {
	return &MockClients{}
}

var Do func(req *http.Request) (*http.Response, error)

func TearDown() {
	Cache = NewCache(50)
	Do = nil
}

func init() {
	InitItsiApiLimiter(1)

	ClientConfigStub = ClientConfig{}
	ClientConfigStub.RetryPolicy = backoff.Null()

	ContextStub = context.Background()
}
