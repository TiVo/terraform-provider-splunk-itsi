//go:build !test_setup
// +build !test_setup

package models

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"
)

func InitHttpClients() IHttpClients {
	return &HttpClients{}
}

type HttpClients struct {
	clientsByConfig map[ClientConfig]*http.Client
	mu              sync.Mutex
}

func (hc *HttpClients) Get(config ClientConfig) IHttpClient {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if hc.clientsByConfig == nil {
		hc.clientsByConfig = make(map[ClientConfig]*http.Client)
	}

	if client, ok := hc.clientsByConfig[config]; ok {
		return client
	}

	tr := (http.DefaultTransport.(*http.Transport)).Clone()
	tr.Dial = func(network, addr string) (net.Conn, error) {
		return net.DialTimeout(network, addr, time.Duration(5*time.Second))
	}
	if config.Concurrency > 0 {
		tr.MaxIdleConns = config.Concurrency
		tr.MaxConnsPerHost = config.Concurrency
	}
	tr.MaxIdleConnsPerHost = tr.MaxIdleConns

	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: (config.Host == "localhost" || config.SkipTLS)}
	client := &http.Client{Transport: tr, Timeout: time.Duration(time.Duration(config.Timeout) * time.Second)}
	hc.clientsByConfig[config] = client

	return client
}
