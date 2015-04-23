package ably

import (
	"errors"
	"fmt"
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

var errInvalidKey = errors.New("invalid key format")

var DefaultOptions = &ClientOptions{
	RestEndpoint:     "https://rest.ably.io",
	RealtimeEndpoint: "wss://realtime.ably.io:443",
	Protocol:         ProtocolJSON, // TODO: make it ProtocolMsgPack
}

type ClientOptions struct {
	RestEndpoint     string
	RealtimeEndpoint string
	Token            string
	Secret           string
	ClientID         string
	Protocol         string // either ProtocolJSON or ProtocolMsgPack
	UseTokenAuth     bool

	HTTPClient *http.Client
}

var (
	restTLS       = strings.NewReplacer("http://", "https://")
	restNoTLS     = strings.NewReplacer("https://", "http://")
	realtimeTLS   = strings.NewReplacer("ws://", "wss://", ":443", ":80")
	realtimeNoTLS = strings.NewReplacer("wss://", "ws://", ":80", ":443")
)

func (opts *ClientOptions) SetTLS(enabled bool) {
	if enabled {
		opts.RestEndpoint = restTLS.Replace(opts.RestEndpoint)
		opts.RealtimeEndpoint = realtimeTLS.Replace(opts.RealtimeEndpoint)
	} else {
		opts.RestEndpoint = restNoTLS.Replace(opts.RestEndpoint)
		opts.RealtimeEndpoint = realtimeNoTLS.Replace(opts.RealtimeEndpoint)
	}
}

func (opts *ClientOptions) SetKey(key string) error {
	s := strings.Split(key, ":")
	if len(s) != 2 {
		return errInvalidKey
	}
	opts.Token = s[0]
	opts.Secret = s[1]
	return nil
}

func (opts *ClientOptions) Key() string {
	if opts.Token == "" || opts.Secret == "" {
		return ""
	}
	return opts.Token + ":" + opts.Secret
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
	return ProtocolJSON // TODO: make it ProtocolMsgPack
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
