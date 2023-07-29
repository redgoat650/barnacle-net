package transport

import (
	"errors"
	"log"
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
	close(t.incomingCmds)
}

func (t *Transport) IncomingCmds() <-chan *message.Command {
	return t.incomingCmds
}

func (t *Transport) listen() {
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

	ch <- *r
}

func (t *Transport) SendCommand(c *message.Command) (<-chan message.Response, error) {
	id, ch := t.inflight.Register()

	c.Opaque = id

	// Prevent concurrent writes to the websocket.
	t.wMu.Lock()
	defer t.wMu.Unlock()

	err := t.conn.WriteJSON(c)
	if err != nil {
		return nil, err
	}

	return ch, nil
}
