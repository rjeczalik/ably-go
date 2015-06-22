package ably

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"encoding/json"

	"github.com/ably/ably-go/Godeps/_workspace/src/github.com/flynn/flynn/pkg/random"
)

var (
	errMissingKey          = errors.New("missing key")
	errInvalidKey          = errors.New("invalid key")
	errMismatchedKeys      = errors.New("mismatched keys")
	errUnsupportedType     = errors.New("unsupported Content-Type header in response from AuthURL")
	errMissingType         = errors.New("missing Content-Type header in response from AuthURL")
	errInvalidCallbackType = errors.New("invalid value type returned from AuthCallback")
)

// addParams copies each params from rhs to lhs and returns lhs.
//
// If param from rhs exists in lhs, it's omitted.
func addParams(lhs, rhs url.Values) url.Values {
	for key := range rhs {
		if lhs.Get(key) != "" {
			continue
		}
		lhs.Set(key, rhs.Get(key))
	}
	return lhs
}

// addHeaders copies each header from rhs to lhs and returns lhs.
//
// If header from rhs exists in lhs, it's omitted.
func addHeaders(lhs, rhs http.Header) http.Header {
	for key := range rhs {
		if lhs.Get(key) != "" {
			continue
		}
		lhs.Set(key, rhs.Get(key))
	}
	return lhs
}

// Auth
type Auth struct {
	useToken bool // when true token auth is used; basic auth otherwise
	options  ClientOptions
	client   *RestClient
	token    *TokenDetails
}

// CreateTokenRequest
func (a *Auth) CreateTokenRequest(opts *AuthOptions, params *TokenParams) (*TokenRequest, error) {
	if opts == nil {
		opts = &a.options.AuthOptions
	} else {
		opts.merge(&a.options.AuthOptions)
	}
	keyName, keySecret := opts.KeyName(), opts.KeySecret()
	req := &TokenRequest{}
	if params != nil {
		req.TokenParams = *params
	}
	if req.KeyName == "" {
		req.KeyName = keyName
	}
	if err := a.setDefaults(opts, req); err != nil {
		return nil, err
	}
	// Validate arguments.
	switch {
	case opts.Key == "":
		return nil, newError(40101, errMissingKey)
	case keyName == "" || keySecret == "":
		return nil, newError(40102, errInvalidKey)
	case req.KeyName != keyName:
		return nil, newError(40102, errMismatchedKeys)
	}
	req.sign([]byte(keySecret))
	return req, nil
}

// RequestToken
func (a *Auth) RequestToken(opts *AuthOptions, params *TokenParams) (*TokenDetails, error) {
	if opts == nil {
		opts = &a.options.AuthOptions
	} else {
		opts.merge(&a.options.AuthOptions)
	}
	var tokReq *TokenRequest
	switch {
	case opts.AuthCallback != nil:
		v, err := opts.AuthCallback(params)
		if err != nil {
			return nil, newError(40170, err)
		}
		switch v := v.(type) {
		case *TokenRequest:
			tokReq = v
		case *TokenDetails:
			return v, nil
		case string:
			return newTokenDetails(v), nil
		default:
			return nil, newError(40170, errInvalidCallbackType)
		}
	case opts.AuthURL != "":
		// TODO(rjeczalik): requestAuthURL should detect whether it received
		// *TokenRequest or *TokenDetails; current implementation assumes it's
		// the latter to cover 2/3 of the use-case.
		return a.requestAuthURL(opts, params)
	default:
		req, err := a.CreateTokenRequest(opts, params)
		if err != nil {
			return nil, err
		}
		tokReq = req
	}
	token := &TokenDetails{}
	_, err := a.client.Post("/keys/"+tokReq.KeyName+"/requestToken", tokReq, token)
	if err != nil {
		return nil, err
	}
	return token, nil
}

// Authorise
func (a *Auth) Authorise(opts *AuthOptions, params *TokenParams, force bool) (*TokenDetails, error) {
	if a.token != nil && !a.token.Expired() && !force {
		return a.token, nil
	}
	token, err := a.RequestToken(opts, params)
	if err != nil {
		a.token = nil
		return nil, err
	}
	a.token = token
	return token, nil
}

func (a *Auth) setDefaults(opts *AuthOptions, req *TokenRequest) error {
	if req.Nonce == "" {
		req.Nonce = random.String(32)
	}
	if req.RawCapability == "" {
		req.RawCapability = (Capability{"*": {"*"}}).Encode()
	}
	if req.TTL == 0 {
		req.TTL = 60 * 60 * 1000
	}
	if req.ClientID == "" {
		req.ClientID = a.options.ClientID
	}
	if req.Timestamp == 0 {
		if opts.UseQueryTime {
			t, err := a.client.Time()
			if err != nil {
				return newError(40100, err)
			}
			req.Timestamp = Timestamp(t)
		} else {
			req.Timestamp = TimestampNow()
		}
	}
	return nil
}

func (a *Auth) requestAuthURL(opts *AuthOptions, params *TokenParams) (*TokenDetails, error) {
	req, err := http.NewRequest(opts.authURLMethod(), opts.AuthURL, nil)
	if err != nil {
		return nil, newError(40000, err)
	}
	req.URL.RawQuery = addParams(params.Query(), opts.AuthParams).Encode()
	req.Header = addHeaders(req.Header, opts.AuthHeaders)
	resp, err := a.options.httpclient().Do(req)
	if err != nil {
		return nil, newError(40000, err)
	}
	if err = checkValidHTTPResponse(resp); err != nil {
		return nil, newError(40000, err)
	}
	defer resp.Body.Close()
	contentType := resp.Header.Get("Content-Type")
	if i := strings.IndexRune(contentType, ';'); i != -1 {
		contentType = contentType[:i]
	}
	switch contentType {
	case "text/plain":
		token, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, newError(40000, err)
		}
		return newTokenDetails(string(token)), nil
	case "application/json":
		var token TokenDetails
		err := json.NewDecoder(resp.Body).Decode(&token)
		if err != nil {
			return nil, newError(40000, err)
		}
		return &token, nil
	case "":
		return nil, newError(40000, errMissingType)
	default:
		return nil, newError(40000, errUnsupportedType)
	}
}
