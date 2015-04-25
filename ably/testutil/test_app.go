package testutil

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/ably/ably-go/ably"
)

type testAppNamespace struct {
	ID        string `json:"id"`
	Persisted bool   `json:"persisted"`
}

type testAppKey struct {
	ID         string `json:"id,omitempty"`
	Value      string `json:"value,omitempty"`
	Capability string `json:"capability,omitempty"`
}

type testAppPresenceConfig struct {
	ClientID string `json:"clientId"`
	Data     string `json:"data"`
}

type testAppChannels struct {
	Name     string                  `json:"name"`
	Presence []testAppPresenceConfig `json:"presence"`
}

type testAppConfig struct {
	ApiID      string             `json:"appId,omitempty"`
	Keys       []testAppKey       `json:"keys"`
	Namespaces []testAppNamespace `json:"namespaces"`
	Channels   []testAppChannels  `json:"channels"`
}

type App struct {
	Config testAppConfig
	opts   *ably.ClientOptions
}

func (t *App) Options() *ably.ClientOptions {
	opts := *t.opts
	if opts.HTTPClient != nil {
		cli := *opts.HTTPClient
		opts.HTTPClient = &cli
	}
	return &opts
}

func (t *App) AppKeyValue() string {
	return t.Config.Keys[0].Value
}

func (t *App) AppKeyId() string {
	return fmt.Sprintf("%s.%s", t.Config.ApiID, t.Config.Keys[0].ID)
}

func (t *App) ok(status int) bool {
	return status == http.StatusOK || status == http.StatusCreated || status == http.StatusNoContent
}

func (t *App) populate(res *http.Response) error {
	if err := json.NewDecoder(res.Body).Decode(&t.Config); err != nil {
		return err
	}

	t.opts.Token = t.AppKeyId()
	t.opts.Secret = t.AppKeyValue()

	return nil
}

func (t *App) httpclient() *http.Client {
	if t.Options != nil && t.opts.HTTPClient != nil {
		return t.opts.HTTPClient
	}
	return http.DefaultClient
}

func (t *App) Create() (*http.Response, error) {
	buf, err := json.Marshal(t.Config)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", t.opts.RestEndpoint+"/apps", bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	res, err := t.httpclient().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if !t.ok(res.StatusCode) {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return res, fmt.Errorf("Unexpected status code %d. Body couldn't be parsed: %s", res.StatusCode, err)
		} else {
			return res, fmt.Errorf("Unexpected status code %d: %s", res.StatusCode, body)
		}
	}

	err = t.populate(res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (t *App) Delete() (*http.Response, error) {
	req, err := http.NewRequest("DELETE", t.opts.RestEndpoint+"/apps/"+t.Config.ApiID, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(t.AppKeyId(), t.AppKeyValue())
	res, err := t.httpclient().Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if !t.ok(res.StatusCode) {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return res, fmt.Errorf("Unexpected status code %d. Body couldn't be parsed: %s", res.StatusCode, err)
		} else {
			return res, fmt.Errorf("Unexpected status code %d: %s", res.StatusCode, body)
		}
	}

	return res, nil
}

var timeout = 10 * time.Second

func NewApp() *App {
	var client *http.Client

	// Don't verify hostname - for use with proxies for testing purposes.
	if os.Getenv("HTTP_PROXY") != "" {
		client = &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				Dial: (&net.Dialer{
					Timeout:   timeout,
					KeepAlive: timeout,
				}).Dial,
				TLSHandshakeTimeout: timeout,
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			},
		}
	}

	environment := "sandbox"
	if s := os.Getenv("ABLY_ENV"); s != "" {
		environment = s
	}

	return &App{
		opts: &ably.ClientOptions{
			RealtimeEndpoint: fmt.Sprintf("wss://%s-realtime.ably.io:443", environment),
			RestEndpoint:     fmt.Sprintf("https://%s-rest.ably.io", environment),
			HTTPClient:       client,
		},
		Config: testAppConfig{
			Keys: []testAppKey{
				{},
			},
			Namespaces: []testAppNamespace{
				{ID: "persisted", Persisted: true},
			},
			Channels: []testAppChannels{
				{
					Name: "persisted:presence_fixtures",
					Presence: []testAppPresenceConfig{
						{ClientID: "client_bool", Data: "true"},
						{ClientID: "client_int", Data: "true"},
						{ClientID: "client_string", Data: "true"},
						{ClientID: "client_json", Data: "{ \"test\" => 'This is a JSONObject clientData payload'}"},
					},
				},
			},
		},
	}
}
