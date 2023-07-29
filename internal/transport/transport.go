package transport

import (
	"errors"
	"log"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/redgoat650/barnacle-net/internal/inflight"
	"github.com/redgoat650/barnacle-net/internal/message"
)

type Transport struct {
	incomingCmds chan *message.Command
	inflight     *inflight.Inflight
	conn         *websocket.Conn
	wMu          *sync.Mutex
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
	}

	go t.listen()

	return t
}

func (t *Transport) Shutdown() {
	t.conn.Close()
	// close(t.incomingCmds)
}

func (t *Transport) GracefullyClose() error {
	return t.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

func (t *Transport) IncomingCmds() <-chan *message.Command {
	return t.incomingCmds
}

func (t *Transport) listen() {
	defer close(t.incomingCmds)

	for {
		err := t.readJSON()
		if err != nil {
			log.Println("error reading message from conn:", err)
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
	t.incomingCmds <- c
}

func (t *Transport) handleResponse(r *message.Response) {
	ch, ok := t.inflight.Get(r.Command.Opaque)
	if !ok {
		log.Println("No one waiting for response")
		return
	}

	ch <- r

	close(ch)
}

func (t *Transport) SendCommand(c *message.Command) (<-chan *message.Response, error) {
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
	m := &message.Message{
		Response: &message.Response{
			Payload: rp,
			Command: cmd,
		},
	}

	return t.sendMessage(m)
}

func (t *Transport) sendMessage(m *message.Message) error {
	// Prevent concurrent writes to the websocket.
	t.wMu.Lock()
	defer t.wMu.Unlock()

	return t.conn.WriteJSON(m)
}
