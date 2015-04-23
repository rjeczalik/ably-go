package ably

import (
	"bytes"
	_ "crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/ably/ably-go/ably/proto"

	"github.com/ably/ably-go/Godeps/_workspace/src/gopkg.in/vmihailenco/msgpack.v2"
)

var (
	msgType     = reflect.TypeOf((*[]*proto.Message)(nil)).Elem()
	statType    = reflect.TypeOf((*[]*proto.Stat)(nil)).Elem()
	presMsgType = reflect.TypeOf((*[]*proto.PresenceMessage)(nil)).Elem()
)

func query(fn func(string, interface{}) (*http.Response, error)) QueryFunc {
	return func(path string) (*http.Response, error) {
		return fn(path, nil)
	}
}

type RestClient struct {
	Options *ClientOptions

	mtx   sync.Mutex // protects chans
	chans map[string]*RestChannel
	auth  *Auth
}

func NewRestClient(key string) *RestClient {
	var opts ClientOptions
	if err := opts.SetKey(key); err != nil {
		panic("ably: NewRestClient using " + err.Error())
	}
	return &RestClient{Options: &opts}
}

func (c *RestClient) options() *ClientOptions {
	if c.Options != nil {
		return c.Options
	}
	return DefaultOptions
}

func (c *RestClient) lazychans() map[string]*RestChannel {
	if c.chans == nil {
		c.chans = make(map[string]*RestChannel)
	}
	return c.chans
}

func (c *RestClient) Auth() *Auth {
	if c.auth == nil {
		c.auth = &Auth{
			options: c.Options,
			client:  c,
		}
	}
	return c.auth
}

func (c *RestClient) Time() (time.Time, error) {
	times := []int64{}
	_, err := c.Get("/time", &times)
	if err != nil {
		return time.Time{}, err
	}
	if len(times) != 1 {
		return time.Time{}, fmt.Errorf("expected 1 timestamp, got %d", len(times))
	}
	return time.Unix(times[0]/1000, times[0]%1000), nil
}

func (c *RestClient) Channel(name string) *RestChannel {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if ch, ok := c.lazychans()[name]; ok {
		return ch
	}
	ch := newRestChannel(name, c)
	c.chans[name] = ch
	return ch
}

// Stats gives the channel's metrics according to the given parameters.
// The returned resource can be inspected for the statistics via the Stats()
// method.
func (c *RestClient) Stats(params *PaginateParams) (*PaginatedResource, error) {
	return newPaginatedResource(statType, "/stats", params, query(c.Get))
}

func (c *RestClient) Get(path string, out interface{}) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.options().RestEndpoint+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	// TODO(rjeczalik): add support for token auth
	req.SetBasicAuth(c.options().Token, c.options().Secret)
	res, err := c.options().httpclient().Do(req)
	if err != nil {
		return nil, err
	}
	if !c.ok(res.StatusCode) {
		return res, NewRestHttpError(res, fmt.Sprintf("Unexpected status code %d", res.StatusCode))
	}
	if out != nil {
		defer res.Body.Close()
		return res, json.NewDecoder(res.Body).Decode(out)
	}
	return res, nil
}

func (c *RestClient) Post(path string, in, out interface{}) (*http.Response, error) {
	buf, err := c.marshalMessages(in)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", c.options().RestEndpoint+path, bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	// TODO(rjeczalik): add support for token auth
	req.SetBasicAuth(c.options().Token, c.options().Secret)
	res, err := c.options().httpclient().Do(req)
	if err != nil {
		return nil, err
	}
	if !c.ok(res.StatusCode) {
		return res, NewRestHttpError(res, fmt.Sprintf("Unexpected status code %d", res.StatusCode))
	}
	if out != nil {
		defer res.Body.Close()
		return res, json.NewDecoder(res.Body).Decode(out)
	}
	return res, nil
}

func (c *RestClient) ok(status int) bool {
	return status == http.StatusOK || status == http.StatusCreated
}

func (c *RestClient) marshalMessages(in interface{}) ([]byte, error) {
	switch proto := c.options().protocol(); proto {
	case ProtocolJSON:
		return json.Marshal(in)
	case ProtocolMsgPack:
		return msgpack.Marshal(in)
	default:
		return nil, errors.New(`invalid protocol: "` + proto + `"`)
	}
}
