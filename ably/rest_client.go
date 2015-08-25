package ably

import (
	"bytes"
	_ "crypto/sha512"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/ably/ably-go/ably/proto"

	"github.com/ably/ably-go/Godeps/_workspace/src/gopkg.in/vmihailenco/msgpack.v2"
)

var (
	msgType     = reflect.TypeOf((*[]*proto.Message)(nil)).Elem()
	statType    = reflect.TypeOf((*[]*proto.Stats)(nil)).Elem()
	presMsgType = reflect.TypeOf((*[]*proto.PresenceMessage)(nil)).Elem()
)

var protoMIME = map[string]string{
	ProtocolJSON:    "application/json",
	ProtocolMsgPack: "application/x-msgpack",
}

func query(fn func(string, interface{}) (*http.Response, error)) QueryFunc {
	return func(path string) (*http.Response, error) {
		return fn(path, nil)
	}
}

type RestClient struct {
	Auth     *Auth
	Protocol string
	Host     string

	chansMtx sync.Mutex
	chans    map[string]*RestChannel
}

func NewRestClient(options *ClientOptions) (*RestClient, error) {
	keyName, keySecret := options.KeyName(), options.KeySecret()
	if keyName == "" || keySecret == "" {
		return nil, newError(40005, errInvalidKey)
	}
	c := &RestClient{
		Protocol: options.protocol(),
		Host:     options.restURL(),
		chans:    make(map[string]*RestChannel),
	}
	c.Auth = &Auth{
		options: *options,
		client:  c,
	}
	return c, nil
}

func (c *RestClient) Time() (time.Time, error) {
	times := []int64{}
	_, err := c.Get("/time", &times)
	if err != nil {
		return time.Time{}, err
	}
	if len(times) != 1 {
		return time.Time{}, newErrorf(50000, "expected 1 timestamp, got %d", len(times))
	}
	return time.Unix(times[0]/1000, times[0]%1000), nil
}

func (c *RestClient) Channel(name string) *RestChannel {
	c.chansMtx.Lock()
	defer c.chansMtx.Unlock()
	if ch, ok := c.chans[name]; ok {
		return ch
	}
	ch := newRestChannel(name, c)
	c.chans[name] = ch
	return ch
}

func (c *RestClient) handleResp(v interface{}, resp *http.Response, err error) (*http.Response, error) {
	if err != nil {
		return nil, newError(50000, err)
	}
	if err = checkValidHTTPResponse(resp); err != nil {
		return nil, err
	}
	if v == nil {
		return resp, nil
	}
	defer resp.Body.Close()
	proto := c.Auth.options.protocol()
	typ, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	if typ != protoMIME[proto] {
		return nil, newErrorf(40000, "unrecognized Content-Type: %q", typ)
	}
	if err := decode(typ, resp.Body, v); err != nil {
		return nil, err
	}
	return resp, nil
}

// Stats gives the channel's metrics according to the given parameters.
// The returned result can be inspected for the statistics via the Stats()
// method.
func (c *RestClient) Stats(params *PaginateParams) (*PaginatedResult, error) {
	return newPaginatedResult(statType, "/stats", params, query(c.Get))
}

func (c *RestClient) Get(path string, out interface{}) (*http.Response, error) {
	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Auth.options.httpclient().Do(req)
	return c.handleResp(out, resp, err)
}

func (c *RestClient) Post(path string, in, out interface{}) (*http.Response, error) {
	req, err := c.newRequest("POST", path, in)
	if err != nil {
		return nil, err
	}
	resp, err := c.Auth.options.httpclient().Do(req)
	return c.handleResp(out, resp, err)
}

func (c *RestClient) newRequest(method, path string, in interface{}) (*http.Request, error) {
	var body io.Reader
	var typ = protoMIME[c.Auth.options.protocol()]
	if in != nil {
		p, err := encode(typ, in)
		if err != nil {
			return nil, newError(ErrCodeProtocol, err)
		}
		body = bytes.NewReader(p)
	}
	req, err := http.NewRequest(method, c.Auth.options.restURL()+path, body)
	if err != nil {
		return nil, newError(50000, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", typ)
	}
	req.Header.Set("Accept", typ)
	req.SetBasicAuth(c.Auth.options.KeyName(), c.Auth.options.KeySecret())
	return req, nil
}

func encode(typ string, in interface{}) ([]byte, error) {
	switch typ {
	case "application/json":
		return json.Marshal(in)
	case "application/x-msgpack":
		return msgpack.Marshal(in)
	default:
		return nil, newErrorf(40000, "unrecognized Content-Type: %q", typ)
	}
}

func decode(typ string, r io.Reader, out interface{}) error {
	switch typ {
	case "application/json":
		return json.NewDecoder(r).Decode(out)
	case "application/x-msgpack":
		return msgpack.NewDecoder(r).Decode(out)
	default:
		return newErrorf(40000, "unrecognized Content-Type: %q", typ)
	}
}
