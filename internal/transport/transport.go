package transport

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redgoat650/barnacle-net/internal/inflight"
	"github.com/redgoat650/barnacle-net/internal/message"
)

type Transport struct {
	incomingCmds chan *message.Command
	inflight     *inflight.Inflight
	conn         *websocket.Conn
	wMu          *sync.Mutex

	stopping bool
	stopMu   *sync.RWMutex
}

func NewTransportConn(server, path string) (*Transport, error) {
	// Dial the websocket at host/path.
	URL := url.URL{Scheme: "ws", Host: server, Path: path}
	c, _, err := websocket.DefaultDialer.Dial(URL.String(), nil)
	if err != nil {
		return nil, err
	}

	t := &Transport{
		incomingCmds: make(chan *message.Command, 5),
		inflight:     inflight.NewInflight(),
		conn:         c,
		wMu:          new(sync.Mutex),
		stopMu:       new(sync.RWMutex),
	}

	go t.listen()

	return t, nil
}

func NewTransportForConn(c *websocket.Conn) *Transport {
	t := &Transport{
		incomingCmds: make(chan *message.Command, 5),
		inflight:     inflight.NewInflight(),
		conn:         c,
		wMu:          new(sync.Mutex),
		stopMu:       new(sync.RWMutex),
	}

	go t.listen()

	return t
}

func (t *Transport) Stopping() bool {
	t.stopMu.RLock()
	defer t.stopMu.RUnlock()

	return t.stopping
}

func (t *Transport) shutdown() {
	t.stopMu.Lock()
	defer t.stopMu.Unlock()

	t.stopping = true
}

func (t *Transport) GracefullyClose() error {
	// Mark this transport as in the process of shutting down.
	t.shutdown()

	log.Println("sending close message to websocket")
	err := t.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	select {
	case <-t.incomingCmds:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("timed out waiting for channel to close: %s; %s", err, ctx.Err())
	}
}

func (t *Transport) sendClosingRepliesToAllInflight() {
	for _, id := range t.inflight.Keys() {
		ch, ok := t.inflight.Get(id)
		if !ok {
			// It's possible this command was responded to after Keys() was called. Ignore and continue.
			continue
		}

		tNow := time.Now()
		r := &message.Response{
			Error:      "transport shutting down",
			Success:    false,
			ArriveTime: &tNow,
		}

		ch <- r
	}
}

func (t *Transport) IncomingCmds() <-chan *message.Command {
	return t.incomingCmds
}

func (t *Transport) handleClosedWebsocket() {
	// Accept no more outgoing commands.
	t.shutdown()

	// Formally close the websocket
	log.Println("websocket close:", t.conn.Close())

	// Notify anyone waiting on a response that no response will be arriving.
	t.sendClosingRepliesToAllInflight()

	// Notify callers that no further incoming commands are expected.
	close(t.incomingCmds)
}

func (t *Transport) listen() {
	defer t.handleClosedWebsocket()

	for {
		err := t.readJSON()
		if err != nil {
			closeErr := &websocket.CloseError{}
			if errors.As(err, &closeErr) {
				if closeErr.Code == websocket.CloseNormalClosure {
					log.Println("normal closure message received")
					err = nil
				}
				return
			}

			log.Println("Error reading message from conn:", err)
			return
		}
	}
}

func (t *Transport) readJSON() error {
	m := &message.Message{}
	err := t.conn.ReadJSON(m)
	if err != nil {
		return err
	}

	switch {
	case m.Command != nil:
		t.handleCommand(m.Command)
	case m.Response != nil:
		t.handleResponse(m.Response)
	default:
		return errors.New("invalid message")
	}

	return nil
}

func (t *Transport) handleCommand(c *message.Command) {
	tNow := time.Now()
	c.ArriveTime = &tNow

	if t.Stopping() {
		err := t.SendResponse(nil, errors.New("not accepting commands due to closing websocket"), c)
		if err != nil {
			log.Println("Sending socket-closed response:", err)
		}
		return
	}

	t.incomingCmds <- c
}

func (t *Transport) handleResponse(r *message.Response) {
	tNow := time.Now()
	r.ArriveTime = &tNow

	ch, ok := t.inflight.Get(r.Command.Opaque)
	if !ok {
		log.Println("No one waiting for response")
		return
	}

	ch <- r

	close(ch)
}

func (t *Transport) SendCommand(c *message.Command) (<-chan *message.Response, error) {
	// Keep the lock held until the message is sent; can't gracefully stop
	// until this process completes.
	t.stopMu.RLock()
	defer t.stopMu.RUnlock()

	if t.stopping {
		return nil, errors.New("transport is stopping and not accepting new outbound commands")
	}

	tNow := time.Now()
	c.SubmitTime = &tNow

	id, ch := t.inflight.Register()
	c.Opaque = id

	m := &message.Message{
		Command: c,
	}

	err := t.sendMessage(m)
	if err != nil {
		t.inflight.Unregister(id)
		return nil, err
	}

	return ch, nil
}

func (t *Transport) SendResponse(rp *message.ResponsePayload, gotErr error, cmd *message.Command) error {
	sendErr := ""
	if gotErr != nil {
		sendErr = gotErr.Error()
	}

	tNow := time.Now()
	m := &message.Message{
		Response: &message.Response{
			Success:    gotErr == nil,
			Error:      sendErr,
			Payload:    rp,
			Command:    cmd,
			SubmitTime: &tNow,
		},
	}

	return t.sendMessage(m)
}

func WaitOnResponse(respCh <-chan *message.Response, timeout time.Duration) (*message.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case resp := <-respCh:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (t *Transport) sendMessage(m *message.Message) error {
	// Prevent concurrent writes to the websocket.
	t.wMu.Lock()
	defer t.wMu.Unlock()

	return t.conn.WriteJSON(m)
}
