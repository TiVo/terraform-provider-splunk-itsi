package models

import (
	"github.com/lestrrat-go/backoff/v2"
	"net/http"
)

type ClientConfig struct {
	BearerToken string
	User        string
	Password    string
	Host        string
	Port        int
	SkipTLS     bool
	Concurrency int
	Timeout     int
	RetryPolicy backoff.Policy
}

type IHttpClients interface {
	Get(config ClientConfig) IHttpClient
}

type IHttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}
