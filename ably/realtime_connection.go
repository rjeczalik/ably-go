// +build ignore

package ably

import (
	"fmt"
	"net/url"
	"strconv"
	"sync"

	"github.com/ably/ably-go/ably/proto"

	"github.com/ably/ably-go/Godeps/_workspace/src/code.google.com/p/go.net/websocket"
)

type ConnState int

const (
	ConnStateInitialized ConnState = iota
	ConnStateConnecting
	ConnStateConnected
	ConnStateDisconnected
	ConnStateSuspended
	ConnStateClosed
	ConnStateFailed
)

type ConnListener func()

type Conn struct {
	ClientOptions
	ID  string
	Ch  chan *proto.ProtocolMessage
	Err chan error

	stateChan   chan ConnState
	ws          *websocket.Conn
	state       ConnState
	mtx         sync.RWMutex
	stateMtx    sync.RWMutex
	listeners   map[ConnState][]ConnListener
	listenerMtx sync.RWMutex
}

func NewConn(options ClientOptions) *Conn {
	c := &Conn{
		ClientOptions: options,
		Ch:            make(chan *proto.ProtocolMessage),
		Err:           make(chan error),
		state:         ConnStateInitialized,
		stateChan:     make(chan ConnState),
	}
	go c.watchConnectionState()
	return c
}

func (c *Conn) State() ConnState {
	c.stateMtx.Lock()
	defer c.stateMtx.Unlock()
	return c.state
}

func (c *Conn) isActive() bool {
	c.stateMtx.RLock()
	defer c.stateMtx.RUnlock()

	switch c.state {
	case ConnStateConnecting,
		ConnStateConnected:
		return true
	default:
		return false
	}
}

func (c *Conn) send(msg *proto.ProtocolMessage) error {
	return websocket.JSON.Send(c.ws, msg)
}

func (c *Conn) Close() {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.setState(ConnStateFailed)
	c.setConnectionID("")
	c.ws.Close()
}

func (c *Conn) websocketUrl(token *Token) (*url.URL, error) {
	u, err := url.Parse(c.ClientOptions.RealtimeEndpoint + "?access_token=" + token.Token + "&binary=false&timestamp=" + strconv.FormatInt(TimestampNow(), 10))
	if err != nil {
		return nil, err
	}
	return u, err
}

func (c *Conn) Connect() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.isActive() {
		return nil
	}

	c.setState(ConnStateConnecting)

	restRealtimeClient := NewRestClient(c.ClientOptions)
	token, err := restRealtimeClient.Auth.RequestToken(nil)
	if err != nil {
		return fmt.Errorf("Error fetching token: %s", err)
	}

	u, err := c.websocketUrl(token)
	if err != nil {
		return err
	}

	return c.dial(u)
}

func (c *Conn) dial(u *url.URL) error {
	ws, err := websocket.Dial(u.String(), "", "https://"+u.Host)
	if err != nil {
		return err
	}
	c.ws = ws
	go c.read()
	return nil
}

func (c *Conn) read() {
	for {
		msg := &proto.ProtocolMessage{}
		err := websocket.JSON.Receive(c.ws, &msg)
		if err != nil {
			c.Close()
			c.Err <- fmt.Errorf("Failed to get websocket frame: %s", err)
			return
		}
		c.handle(msg)
	}
}

func (c *Conn) watchConnectionState() {
	for {
		select {
		case connState := <-c.stateChan:
			c.trigger(connState)
		}
	}
}

func (c *Conn) trigger(connState ConnState) {
	c.listenerMtx.RLock()
	for _, fn := range c.listeners[connState] {
		go fn()
	}
	c.listenerMtx.RUnlock()
}

func (c *Conn) On(connState ConnState, listener ConnListener) {
	c.listenerMtx.Lock()
	defer c.listenerMtx.Unlock()

	if c.listeners == nil {
		c.listeners = make(map[ConnState][]ConnListener)
	}

	if c.listeners[connState] == nil {
		c.listeners[connState] = []ConnListener{}
	}

	c.listeners[connState] = append(c.listeners[connState], listener)
}

func (c *Conn) setState(state ConnState) {
	c.stateMtx.Lock()
	defer c.stateMtx.Unlock()

	c.state = state
	c.stateChan <- state
}

func (c *Conn) setConnectionID(id string) {
	c.stateMtx.Lock()
	defer c.stateMtx.Unlock()

	c.ID = id
}

func (c *Conn) handle(msg *proto.ProtocolMessage) {
	switch msg.Action {
	case proto.ActionHeartbeat, proto.ActionAck, proto.ActionNack:
		return
	case proto.ActionError:
		if msg.Channel != "" {
			c.Ch <- msg
			return
		}
		c.Close()
		c.Err <- msg.Error
		return
	case proto.ActionConnected:
		c.setState(ConnStateConnected)
		c.setConnectionID(msg.ConnectionId)
		return
	case proto.ActionDisconnected:
		c.setState(ConnStateDisconnected)
		c.setConnectionID("")
		return
	default:
		c.Ch <- msg
	}
}
