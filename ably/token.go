package ably

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// Capability
type Capability map[string][]string

// ParseCapability
func ParseCapability(capability string) (c Capability, err error) {
	c = make(Capability)
	err = json.Unmarshal([]byte(capability), &c)
	return
}

// Encode
func (c Capability) Encode() string {
	if len(c) == 0 {
		return ""
	}
	p, err := json.Marshal((map[string][]string)(c))
	if err != nil {
		panic(err)
	}
	return string(p)
}

// TokenParams
type TokenParams struct {
	// KeyName represents a key name against which a token request is issued.
	KeyName string `json:"keyName" msgpack:"keyName"`

	// TTL is a requested time to live for the token. If the token request
	// is successful, the TTL of the returned token will be less than or equal
	// to this value depending on application settings and the attributes
	// of the issuing key.
	TTL int `json:"ttl" msgpack:"ttl"`

	// RawCapability represents encoded access rights of the token.
	RawCapability string `json:"capability" msgpack:"capability"`

	// ClientID represents a client, whom the token is generated for.
	ClientID string `json:"clientId" msgpack:"clientId"`

	// Timestamp of the token request. It's used, in conjunction with the nonce,
	// are used to prevent token requests from being replayed.
	Timestamp int64 `json:"timestamp" msgpack:"timestamp"`
}

// Capability
func (params *TokenParams) Capability() Capability {
	c, _ := ParseCapability(params.RawCapability)
	return c
}

// Query encodes the params to query params value. If a field of params is
// a zero-value, it's omitted. If params is zero-value, nil is returned.
func (params *TokenParams) Query() url.Values {
	var q url.Values
	if params.KeyName != "" {
		q.Set("keyName", params.KeyName)
	}
	if params.TTL != 0 {
		q.Set("ttl", strconv.Itoa(params.TTL))
	}
	if params.RawCapability != "" {
		q.Set("capability", params.RawCapability)
	}
	if params.ClientID != "" {
		q.Set("clientID", params.ClientID)
	}
	if params.Timestamp != 0 {
		q.Set("timestamp", strconv.FormatInt(params.Timestamp, 10))
	}
	if len(q) == 0 {
		return nil
	}
	return q
}

// TokenRequest
//
// NOTE: struct field inlining is not yet merged to the upstream at the moment
// of writing - https://github.com/vmihailenco/msgpack/pull/40.
type TokenRequest struct {
	TokenParams `msgpack:",inline"`

	Nonce string `json:"nonce" msgpack:"nonce"` // should be at least 16 characters long
	Mac   string `json:"mac" msgpack:"mac"`     // message authentication code for the request
}

func (req *TokenRequest) sign(secret []byte) {
	mac := hmac.New(sha256.New, secret)
	fmt.Fprintln(mac, req.KeyName)
	fmt.Fprintln(mac, req.TTL)
	fmt.Fprintln(mac, req.RawCapability)
	fmt.Fprintln(mac, req.ClientID)
	fmt.Fprintln(mac, req.Timestamp)
	fmt.Fprintln(mac, req.Nonce)
	req.Mac = base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// TokenDetails
type TokenDetails struct {
	// Token
	Token string `json:"token" msgpack:"token"`

	// KeyName
	KeyName string `json:"keyName" msgpack:"keyName"`

	// Expires
	Expires int64 `json:"expires" msgpack:"expires"`

	// Issued
	Issued int64 `json:"issued" msgpack:"issued"`

	// RawCapability
	RawCapability string `json:"capability" msgpack:"capability"`
}

// Capability
func (tok *TokenDetails) Capability() Capability {
	c, _ := ParseCapability(tok.RawCapability)
	return c
}

// Expired
func (tok *TokenDetails) Expired() bool {
	return tok.Expires != 0 && tok.Expires <= TimestampNow()
}

func newTokenDetails(token string) *TokenDetails {
	return &TokenDetails{
		Token: token,
	}
}
