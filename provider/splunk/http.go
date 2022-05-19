package splunk

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

/*
 * HTTP helper methods
 */

func (conn SplunkConnection) client() *http.Client {
	if conn.HttpClient != nil {
		return conn.HttpClient
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	return client
}

func (conn SplunkConnection) httpGet(url string, data *url.Values) (string, error) {
	if response, err := conn.httpCall(url, "GET", data); err != nil {
		return "", err
	} else {
		body, _ := ioutil.ReadAll(response.Body)
		response.Body.Close()
		return string(body), nil
	}
}

func (conn SplunkConnection) httpPost(url string, data *url.Values) (string, error) {
	if response, err := conn.httpCall(url, "POST", data); err != nil {
		return "", err
	} else {
		body, _ := ioutil.ReadAll(response.Body)
		response.Body.Close()
		return string(body), nil
	}
}

func (conn SplunkConnection) httpCall(url string, method string, data *url.Values) (*http.Response, error) {
	return conn.httpCallWithContext(context.Background(), url, method, data)
}

func (conn SplunkConnection) httpCallWithContext(ctx context.Context, url string, method string, data *url.Values) (*http.Response, error) {
	var payload io.Reader
	if data != nil {
		payload = bytes.NewBufferString(data.Encode())
	}

	request, err := http.NewRequestWithContext(ctx, method, url, payload)
	conn.addAuthHeader(request)
	response, err := conn.client().Do(request)

	if err != nil {
		return nil, err
	}
	return response, err
}

func (conn SplunkConnection) addAuthHeader(request *http.Request) {
	if conn.BearerToken != "" {
		request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", conn.BearerToken))
	} else if conn.sessionKey.Value != "" {
		request.Header.Add("Authorization", fmt.Sprintf("Splunk %s", conn.sessionKey))
	} else {
		request.SetBasicAuth(conn.Username, conn.Password)
	}
}
