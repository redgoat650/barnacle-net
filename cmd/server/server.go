package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redgoat650/barnacle-net/internal/message"
	"github.com/redgoat650/barnacle-net/internal/transport"
)

const (
	defaultTimeout = 10 * time.Second
)

type Server struct {
	conns  map[string]*connInfo
	connMu *sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

type connInfo struct {
	t        *transport.Transport
	identity *message.IdentifyResponsePayload
	mu       *sync.Mutex
}

func NewServer() *Server {
	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		conns:  make(map[string]*connInfo),
		connMu: new(sync.RWMutex),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *Server) Shutdown() {
	s.cancel()
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "...placeholder")
}

func makeWSHandler(s *Server) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }

		// Upgrade to a WebSocket connection.
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		// Connection received; log connection event.
		remoteAddr := ws.RemoteAddr().String()
		log.Println("Client Connected", remoteAddr)

		t := transport.NewTransportForConn(ws)

		s.connMu.Lock()
		s.conns[remoteAddr] = &connInfo{
			t:  t,
			mu: new(sync.Mutex),
		}
		s.connMu.Unlock()

		defer func() {
			log.Println("Shutting down client connection:", remoteAddr)
			s.connMu.Lock()
			delete(s.conns, remoteAddr)
			s.connMu.Unlock()

			t.Shutdown()
		}()

		err = s.identify(remoteAddr)
		if err != nil {
			log.Println("error identifying connected client:", err)
			return
		}

		s.handleIncomingCommands(remoteAddr)
	}
}

func (s *Server) handleIncomingCommands(remoteAddr string) {
	for {
		c, ok := s.lookupConn(remoteAddr)
		if !ok {
			log.Println("no conn info stored in server for", remoteAddr)
		}

		select {
		case cmd := <-c.t.IncomingCmds():
			err := s.handleCommand(cmd, c.t)
			if err != nil {
				log.Println("error handling command:", err)
				return
			}

		case <-s.ctx.Done():
			log.Println("context canceled:", s.ctx.Err())
			return
		}
	}
}

func (s *Server) handleCommand(cmd *message.Command, t *transport.Transport) error {
	if cmd == nil {
		return errors.New("transport shutting down websocket conn")
	}

	var (
		rp  *message.ResponsePayload
		err error
	)

	switch cmd.Op {
	case message.ListNodes:
		rp, err = s.handleListNodes(cmd, t)
	default:
		return fmt.Errorf("unrecognized command: %s", cmd.Op)
	}

	log.Println("handling command", cmd.Op)
	return t.SendResponse(rp, err, cmd)
}

func (s *Server) handleListNodes(cmd *message.Command, t *transport.Transport) (*message.ResponsePayload, error) {
	p := cmd.Payload

	refreshIDs := false
	if p != nil && p.ListNodesPayload != nil {
		refreshIDs = p.ListNodesPayload.RefreshIdentities
	}

	s.connMu.RLock()
	defer s.connMu.RUnlock()

	if refreshIDs {
		for remoteAddr, connInfo := range s.conns {
			err := s.identifyOverConn(connInfo)
			if err != nil {
				log.Println("Identify failed for", remoteAddr, "error:", err)
				continue
			}
		}
	}

	nodeIDs := make(map[string]*message.IdentifyResponsePayload)
	for remoteAddr, connInfo := range s.conns {
		connInfo.mu.Lock()
		if id := connInfo.identity; id != nil {
			nodeIDs[remoteAddr] = id
		}

		connInfo.mu.Unlock()
	}

	return &message.ResponsePayload{
		ListNodesResponse: &message.ListNodesResponsePayload{
			Nodes: nodeIDs,
		},
	}, nil
}

func (s *Server) identify(remoteAddr string) error {
	connInfo, ok := s.lookupConn(remoteAddr)
	if !ok {
		return fmt.Errorf("could not identify unknown connection: %s", remoteAddr)
	}

	return s.identifyOverConn(connInfo)
}

func (s *Server) identifyOverConn(connInfo *connInfo) error {
	c := &message.Command{
		Op: message.Identify,
	}

	resp, err := connInfo.t.SendCommand(c)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	select {
	case resp := <-resp:
		return s.handleIdentifyResponse(resp, connInfo)

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Server) handleIdentifyResponse(resp *message.Response, connInfo *connInfo) error {
	if resp == nil {
		return fmt.Errorf("inflight command response lost")
	}

	if resp.Command.Op != message.Identify {
		return fmt.Errorf("response is not for identify command: %s", resp.Command.Op)
	}

	if resp.Error != "" {
		return fmt.Errorf("error in identity response: %s", resp.Error)
	}

	if resp.Payload.IdentifyResponse == nil {
		return fmt.Errorf("identify response payload is empty")
	}

	connInfo.mu.Lock()
	defer connInfo.mu.Unlock()

	connInfo.identity = resp.Payload.IdentifyResponse

	return nil
}

func (s *Server) lookupConn(addr string) (*connInfo, bool) {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	connInfo, ok := s.conns[addr]
	return connInfo, ok
}

// func handleIncomingCmds(ws *websocket.Conn) {
// 	// Server control logic goes here.
// }

// func listen(conn *websocket.Conn) {
// 	for {
// // Read message from the connected client.
// messageType, messageContent, err := conn.ReadMessage()
// timeReceive := time.Now()
// if err != nil {
// 	log.Println(err)
// 	return
// }

// // Log the message.
// log.Println(string(messageContent))

// // reponse message
// messageResponse := fmt.Sprintf("Your message is: %s. Time received : %v", messageContent, timeReceive)

// if err := conn.WriteMessage(messageType, []byte(messageResponse)); err != nil {
// 	log.Println(err)
// 	return
// }

// 	}
// }

func setupRoutes(s *Server) {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/ws", makeWSHandler(s))
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	s := NewServer()
	setupRoutes(s)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
