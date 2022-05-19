package splunk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type SplunkConnection struct {
	Username, Password, BaseURL, SplunkUser, SplunkApp string
	BearerToken                                        string

	sessionKey SessionKey
	HttpClient *http.Client
}

// SessionKey represents the JSON object returned from the Splunk authentication REST call
type SessionKey struct {
	Value string `json:"sessionKey"`
}

// Login connects to the Splunk server and retrieves a session key
func (conn SplunkConnection) Login() (key SessionKey, err error) {

	data := make(url.Values)
	data.Add("username", conn.Username)
	data.Add("password", conn.Password)
	data.Add("output_mode", "json")
	response, err := conn.httpPost(fmt.Sprintf("%s/services/auth/login", conn.BaseURL), &data)

	if err != nil {
		return SessionKey{}, err
	}

	bytes := []byte(response)
	err = json.Unmarshal(bytes, &key)
	if err != nil {
		return
	}

	conn.sessionKey = key
	if !conn.HasSessionKey() {
		err = fmt.Errorf("Login Failed. No session key received.")
	}
	return
}

func (conn SplunkConnection) HasSessionKey() bool {
	return len(conn.sessionKey.Value) > 0
}
