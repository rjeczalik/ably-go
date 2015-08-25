package ably

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	ProtocolJSON    = "json"
	ProtocolMsgPack = "msgpack"
)

var DefaultOptions = &ClientOptions{
	RestHost:          "rest.ably.io",
	RealtimeHost:      "realtime.ably.io",
	Protocol:          ProtocolMsgPack,
	TimeoutConnect:    15 * time.Second,
	TimeoutDisconnect: 30 * time.Second,
	TimeoutSuspended:  2 * time.Minute,
}

type AuthMethod uint8

const (
	AuthBasic AuthMethod = 1 + iota
	AuthToken
)

type AuthOptions struct {
	// AuthCallback is called in order to obtain a signed token request.
	//
	// This enables a client to obtain token requests from another entity,
	// so tokens can be renewed without the client requiring access to keys.
	//
	// The returned value of the functor is expected to be one of the following
	// types:
	//
	//   - string, which is then used as token string
	//   - *ably.TokenRequest, which is then used as an already signed request
	//   - *ably.TokenDetails, which is then used as a token
	//
	AuthCallback func(*TokenParams) (interface{}, error)

	// URL which is queried to obtain a signed token request.
	//
	// This enables a client to obtain token requests from another entity,
	// so tokens can be renewed without the client requiring access to keys.
	//
	// If AuthURL is non-empty and AuthCallback is nil, the Ably library
	// builds a req (*http.Request) which then is issued against the given AuthURL
	// in order to obtain authentication token. The response is expected to
	// carry a single token string in the payload when Content-Type header
	// is "text/plain" or JSON-encoded *ably.TokenDetails when the header
	// is "application/json".
	//
	// The req is built with the following values:
	//
	//   - req.Method is set to AuthMethod
	//   - req.URL.RawQuery is encoded from *TokenParams and AuthParams
	//   - req.Header is set to AuthHeaders
	//
	AuthURL string

	// Key obtained from the dashboard.
	Key string

	// Token is an authentication token issued for this application against
	// a specific key and TokenParams.
	Token string

	// TokenDetails is an authentication token issued for this application against
	// a specific key and TokenParams.
	TokenDetails *TokenDetails

	// AuthMethod specifies which method, GET or POST, is used to query AuthURL
	// for the token information (*ably.TokenRequest or *ablyTokenDetails).
	//
	// If empty, GET is used by default.
	AuthMethod string

	// AuthHeaders are HTTP request headers to be included in any request made
	// to the AuthURL.
	AuthHeaders http.Header

	// AuthParams are HTTP query parameters to be included in any requset made
	// to the AuthURL.
	AuthParams url.Values

	// UseQueryTime when set to true, the time queried from Ably servers will
	// be used to sign the TokenRequest instread of using local time.
	UseQueryTime bool
}

func (opts *AuthOptions) Set(override *AuthOptions) *AuthOptions {
	if override == nil {
		return opts
	}
	if override.AuthCallback != nil {
		opts.AuthCallback = override.AuthCallback
	}
	if override.AuthURL != "" {
		opts.AuthURL = override.AuthURL
	}
	if override.Key != "" {
		opts.Key = override.Key
	}
	if override.Token != "" {
		opts.Token = override.Token
	}
	if override.TokenDetails != nil {
		opts.TokenDetails = override.TokenDetails
	}
	if len(override.AuthHeaders) != 0 {
		opts.AuthHeaders = override.AuthHeaders
	}
	if len(override.AuthParams) != 0 {
		opts.AuthParams = override.AuthParams
	}
	return opts

}

func (opts *AuthOptions) authURLMethod() string {
	if opts.AuthMethod != "" {
		return opts.AuthMethod
	}
	return "GET"
}

func (opts *AuthOptions) merge(defaults *AuthOptions) {
	if opts.AuthCallback == nil {
		opts.AuthCallback = defaults.AuthCallback
	}
	if opts.AuthURL == "" {
		opts.AuthURL = defaults.AuthURL
	}
	if opts.Key == "" {
		opts.Key = defaults.Key
	}
	if opts.Token == "" {
		opts.Token = defaults.Token
	}
	if opts.TokenDetails == nil {
		opts.TokenDetails = defaults.TokenDetails
	}
	if len(opts.AuthHeaders) == 0 {
		opts.AuthHeaders = defaults.AuthHeaders
	}
	if len(opts.AuthParams) == 0 {
		opts.AuthParams = defaults.AuthParams
	}
}

// KeyName gives the key name parsed from the Key field.
func (opts *AuthOptions) KeyName() string {
	if i := strings.IndexRune(opts.Key, ':'); i != -1 {
		return opts.Key[:i]
	}
	return ""
}

// KeySecret gives the key secret parsed from the Key field.
func (opts *AuthOptions) KeySecret() string {
	if i := strings.IndexRune(opts.Key, ':'); i != -1 {
		return opts.Key[i+1:]
	}
	return ""
}

type ClientOptions struct {
	AuthOptions

	RestHost     string // optional; overwrite endpoint hostname for REST client
	RealtimeHost string // optional; overwrite endpoint hostname for Realtime client
	Environment  string // optional; prefixes both hostname with the environment string
	ClientID     string // optional; required for managing realtime presence of the current client
	Protocol     string // optional; either ProtocolJSON or ProtocolMsgPack
	Recover      string // optional; used to recover client state

	UseBinaryProtocol bool // when true uses msgpack for network serialization protocol
	UseTokenAuth      bool // when true REST and realtime client will use token authentication

	NoTLS      bool // when true REST and realtime client won't use TLS
	NoConnect  bool // when true realtime client will not attempt to connect automatically
	NoEcho     bool // when true published messages will not be echoed back
	NoQueueing bool // when true drops messages published during regaining connection

	TimeoutConnect    time.Duration // time period after which connect request is failed
	TimeoutDisconnect time.Duration // time period after which disconnect request is failed
	TimeoutSuspended  time.Duration // time period after which no more reconnection attempts are performed

	// Dial specifies the dial function for creating message connections used
	// by RealtimeClient.
	//
	// If Dial is nil, the default websocket connection is used.
	Dial func(protocol string, u *url.URL) (MsgConn, error)

	// Listener if set, will be automatically registered with On method for every
	// realtime connection and realtime channel created by realtime client.
	// The listener will receive events for all state transitions.
	Listener chan<- State

	// HTTPClient specifies the client used for HTTP communication by RestClient.
	//
	// If HTTPClient is nil, the http.DefaultClient is used.
	HTTPClient *http.Client
}

func NewClientOptions(key string) *ClientOptions {
	return &ClientOptions{
		AuthOptions: AuthOptions{
			Key: key,
		},
	}
}

func (opts *ClientOptions) timeoutConnect() time.Duration {
	if opts.TimeoutConnect != 0 {
		return opts.TimeoutConnect
	}
	return DefaultOptions.TimeoutConnect
}

func (opts *ClientOptions) timeoutDisconnect() time.Duration {
	if opts.TimeoutDisconnect != 0 {
		return opts.TimeoutDisconnect
	}
	return DefaultOptions.TimeoutDisconnect
}

func (opts *ClientOptions) timeoutSuspended() time.Duration {
	if opts.TimeoutSuspended != 0 {
		return opts.TimeoutSuspended
	}
	return DefaultOptions.TimeoutSuspended
}

func (opts *ClientOptions) restURL() string {
	host := opts.RestHost
	if host == "" {
		host = DefaultOptions.RestHost
		if opts.Environment != "" {
			host = opts.Environment + "-" + host
		}
	}
	if opts.NoTLS {
		return "http://" + host
	}
	return "https://" + host
}

func (opts *ClientOptions) realtimeURL() string {
	host := opts.RealtimeHost
	if host == "" {
		host = DefaultOptions.RealtimeHost
		if opts.Environment != "" {
			host = opts.Environment + "-" + host
		}
	}
	if opts.NoTLS {
		return "ws://" + net.JoinHostPort(host, "80")
	}
	return "wss://" + net.JoinHostPort(host, "443")
}

func (opts *ClientOptions) httpclient() *http.Client {
	if opts.HTTPClient != nil {
		return opts.HTTPClient
	}
	return http.DefaultClient
}

func (opts *ClientOptions) protocol() string {
	if opts.Protocol != "" {
		return opts.Protocol
	}
	return DefaultOptions.Protocol
}

// Timestamp returns the given time as a timestamp in milliseconds since epoch.
func Timestamp(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

// TimestampNow returns current time as a timestamp in milliseconds since epoch.
func TimestampNow() int64 {
	return Timestamp(time.Now())
}

// This needs to use a timestamp in millisecond
// Use the previous function to generate them from a time.Time struct.
type ScopeParams struct {
	Start int64
	End   int64
	Unit  string
}

func (s *ScopeParams) EncodeValues(out *url.Values) error {
	if s.Start != 0 && s.End != 0 && s.Start > s.End {
		return fmt.Errorf("start must be before end")
	}
	if s.Start != 0 {
		out.Set("start", strconv.FormatInt(s.Start, 10))
	}
	if s.End != 0 {
		out.Set("end", strconv.FormatInt(s.End, 10))
	}
	if s.Unit != "" {
		out.Set("unit", s.Unit)
	}
	return nil
}

type PaginateParams struct {
	Limit     int
	Direction string
	ScopeParams
}

func (p *PaginateParams) EncodeValues(out *url.Values) error {
	if p.Limit < 0 {
		out.Set("limit", strconv.Itoa(100))
	} else if p.Limit != 0 {
		out.Set("limit", strconv.Itoa(p.Limit))
	}
	switch p.Direction {
	case "":
		break
	case "backwards", "forwards":
		out.Set("direction", p.Direction)
		break
	default:
		return fmt.Errorf("Invalid value for direction: %s", p.Direction)
	}
	p.ScopeParams.EncodeValues(out)
	return nil
}
