package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/redgoat650/barnacle-net/internal/message"
	"github.com/redgoat650/barnacle-net/internal/python"
	"github.com/redgoat650/barnacle-net/internal/transport"
)

type Barnacle struct {
	imagePYRunner *python.ImagePYRunner
	t             *transport.Transport
}

func NewBarnacle(server, path string) (*Barnacle, error) {
	t, err := transport.NewTransportConn(server, path)
	if err != nil {
		return nil, err
	}

	b := &Barnacle{
		imagePYRunner: python.NewImagePYRunner(getScriptPath()),
		t:             t,
	}

	return b, nil
}

func (b *Barnacle) handleIncomingCmds() error {
	// Handle interrupts.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	for {
		select {
		case cmd := <-b.t.IncomingCmds():
			if cmd == nil {
				return errors.New("transport layer has closed the websocket")
			}

			err := b.handleCommand(cmd)
			if err != nil {
				fmt.Println("Error handling incoming command")
			}

		case <-interrupt:
			b.handleInterrupt()
		}
	}
}

func (b *Barnacle) handleInterrupt() {
	log.Println("Caught interrupt signal - gracefully disconnecting websocket")
	err := b.t.GracefullyClose()
	if err != nil {
		log.Println("Websocket close error:", err)
		return
	}
}

func (b *Barnacle) handleCommand(cmd *message.Command) error {
	var (
		rp  *message.ResponsePayload
		err error
	)

	switch cmd.Op {
	case message.Identify:
		rp, err = b.handleIdentify()
	default:
		return fmt.Errorf("unrecognized command: %s", cmd.Op)
	}

	if err != nil {
		log.Printf("Hit error handling command %s. Attempting to send error as response: %s", cmd.Op, err)
	}

	return b.t.SendResponse(rp, err, cmd)
}

func (b *Barnacle) handleIdentify() (*message.ResponsePayload, error) {
	rp, err := makeIDResponsePayload()
	if err != nil {
		return nil, err
	}

	return &message.ResponsePayload{
		IdentifyResponse: rp,
	}, nil
}

func makeIDResponsePayload() (*message.IdentifyResponsePayload, error) {
	user, err := user.Current()
	if err != nil {
		return nil, err
	}

	host, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return &message.IdentifyResponsePayload{
		Identity: message.Identity{
			Role:     message.NodeRole,
			Username: user.Name,
			Hostname: host,
			NumCPU:   runtime.NumCPU(),
			PID:      os.Getpid(),
		},
	}, nil
}

const (
	imgFileCachePath = "images"
	imgPyScriptPath  = "scripts/image.py"
)

// func (b *Barnacle) handleMessage(ctx context.Context, mt int, msg []byte) error {
// 	switch mt {
// 	case websocket.BinaryMessage:
// 		return b.handleBinaryMessage(ctx, msg)
// 	}

// 	return errors.New("unsupported message type")
// }

// func (b *Barnacle) handleBinaryMessage(ctx context.Context, msg []byte) error {
// 	m := &message.Message{}
// 	err := json.Unmarshal(msg, m)
// 	if err != nil {
// 		return err
// 	}

// 	switch {
// 	case m.Command != nil:
// 		return b.handleCommand(ctx, m.Command)
// 	case m.Response != nil:
// 		return b.handleResponse(ctx, m.Response)
// 	}

// 	return errors.New("invalid message contains neither command nor response")
// }

// func (b *Barnacle) handleCommand(ctx context.Context, c *message.Command) error {
// 	var (
// 		err         error
// 		respPayload *message.ResponsePayload
// 	)
// 	switch c.Op {
// 	case message.SetImageCmd:
// 		err = b.handleSetImagePayload(ctx, c.Payload)
// 	default:
// 		return errors.New("unsupported command")
// 	}

// 	b.sendResponse(c, respPayload, err)

// 	return nil
// }

// func (b *Barnacle) sendResponse(c *message.Command, respPayload *message.ResponsePayload, gotErr error) {
// 	resp := &message.Response{
// 		Command: c,
// 		Success: gotErr == nil,
// 		Error:   gotErr.Error(),
// 		Payload: respPayload,
// 	}

// 	b.send <- &message.Message{
// 		Response: resp,
// 	}
// }

// func (b *Barnacle) handleResponse(ctx context.Context, r *message.Response) error {
// 	if r.Command == nil || r.Command.Opaque == 0 {
// 		return errors.New("invalid response does not contain command identifier")
// 	}

// 	opaque := r.Command.Opaque

// 	respChan, ok := b.inflight[opaque]
// 	if !ok {
// 		return errors.New("no one waiting for response")
// 	}

// 	delete(b.inflight, opaque)

// 	respChan <- r

// 	return nil
// }

// func (b *Barnacle) handleSetImagePayload(ctx context.Context, payload *message.CommandPayload) error {
// 	if payload == nil || payload.SetImagePayload == nil {
// 		return errors.New("malformed setImage payload")
// 	}

// 	p := payload.SetImagePayload

// 	// Check if we already have the image downloaded, keyed by filename.
// 	fullPath := getFilePath(p.Name)

// 	_, err := os.Stat(fullPath)
// 	if err != nil {
// 		if errors.Is(err, os.ErrNotExist) {
// 			// Request the server to download.
// 			err := b.downloadImage(ctx, p.Name)
// 			if err != nil {
// 				return err
// 			}
// 		} else {
// 			return err
// 		}
// 	}

// 	// Image should be present locally now. Invoke the image set routine.
// 	b.imagePYRunner.RunImagePY(fullPath, p.Saturation)

// 	return nil
// }

// const cmdTimeout = 60 * time.Second

// func (b *Barnacle) downloadImage(ctx context.Context, name string) error {
// 	cmd := &message.Command{
// 		Op: message.GetImageCmd,
// 		Payload: &message.CommandPayload{
// 			GetImagePayload: &message.GetImagePayload{
// 				Name: name,
// 			},
// 		},
// 	}

// 	respCh, err := b.sendCmd(cmd)
// 	if err != nil {
// 		return err
// 	}

// 	ctx, cancel := context.WithTimeout(ctx, cmdTimeout)
// 	defer cancel()

// 	select {
// 	case resp := <-respCh:
// 		return b.handleDownloadImageResponse(name, resp)
// 	case <-ctx.Done():
// 		return ctx.Err()
// 	}
// }

// func (b *Barnacle) sendCmd(cmd *message.Command) (chan *message.Response, error) {
// 	b.mapMu.Lock()
// 	defer b.mapMu.Unlock()

// 	opaque := rand.Uint64()
// 	for _, ok := b.inflight[opaque]; ok; {
// 		opaque = rand.Uint64()
// 	}

// 	cmd.Opaque = opaque
// 	msg := &message.Message{
// 		Command: cmd,
// 	}

// 	respChan := make(chan *message.Response)

// 	b.inflight[opaque] = respChan

// 	b.send <- msg

// 	return respChan, nil
// }

// func (b *Barnacle) handleDownloadImageResponse(name string, resp *message.Response) error {
// 	if resp == nil || resp.Payload == nil || resp.Payload.GetImageResponse == nil {
// 		return errors.New("malformed download image response payload")
// 	}

// 	payload := resp.Payload.GetImageResponse

// 	if len(payload.ImageData) == 0 {
// 		return errors.New("empty image data in response")
// 	}

// 	if payload.Name == "" {
// 		return errors.New("empty image name")
// 	}

// 	if payload.Name != name {
// 		fmt.Println("unexpected image name mismatch")
// 	}

// 	filename := getFilePath(name)
// 	return os.WriteFile(filename, payload.ImageData, 0755)
// }

// func getFilePath(name string) string {
// 	tempDir := filepath.Join(os.TempDir(), imgFileCachePath)
// 	return filepath.Join(tempDir, name)
// }

func getScriptPath() string {
	return filepath.Join(os.TempDir(), imgPyScriptPath)
}

// func (b *Barnacle) sendMessage(c *websocket.Conn, m *message.Message) error {
// 	buf, err := json.Marshal(m)
// 	if err != nil {
// 		b.inflightErr(m, err)
// 		return err
// 	}

// 	err = c.WriteMessage(websocket.BinaryMessage, buf)
// 	if err != nil {
// 		b.inflightErr(m, err)
// 		return err
// 	}

// 	return nil
// }

// func (b *Barnacle) inflightErr(m *message.Message, err error) {
// 	b.mapMu.Lock()
// 	defer b.mapMu.Unlock()

// 	respCh, ok := b.inflight[m.Command.Opaque]
// 	if !ok {
// 		log.Printf("command errored but no one was waiting for a response")
// 	}

// 	resp := &message.Response{
// 		Command: m.Command,
// 		Success: false,
// 		Error:   err.Error(),
// 	}

// 	respCh <- resp
// }

func getServerConnInfo() (string, string, error) {
	arguments := os.Args
	if len(arguments) != 3 {
		return "", "", errors.New("must be initialized with two arguments: server and path")
	}

	server := arguments[1]
	path := arguments[2]

	return server, path, nil
}

func main() {
	// Read connection info from args.
	server, path, err := getServerConnInfo()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	fmt.Println("Connecting to:", server, "at", path)

	b, err := NewBarnacle(server, path)
	if err != nil {
		log.Println("instantiating barnacle:", err)
		os.Exit(1)
	}

	// Block while handling incoming commands.
	err = b.handleIncomingCmds()

	log.Println("Node shutting down:", err)
}

// 	// Read connection info from args.
// 	arguments := os.Args
// 	if len(arguments) != 3 {
// 		fmt.Println("Must be initialized with two arguments, server and path")
// 		return
// 	}
// 	server := arguments[1]
// 	path := arguments[2]
// 	fmt.Println("Connecting to:", server, "at", path)

// 	// Handle interrupts.
// 	interrupt := make(chan os.Signal, 1)
// 	signal.Notify(interrupt, os.Interrupt)

// 	// Instantiate the barnacle data structure.
// 	b := NewBarnacle()

// 	// Dial the websocket at host/path.
// 	URL := url.URL{Scheme: "ws", Host: server, Path: path}
// 	c, _, err := websocket.DefaultDialer.Dial(URL.String(), nil)
// 	if err != nil {
// 		log.Println("Error:", err)
// 		return
// 	}
// 	defer c.Close()

// 	// Read and handle all messages from the websocket connection.
// 	ctx := context.Background()
// 	done := make(chan struct{})
// 	go func() {
// 		defer close(done)
// 		for {
// 			mt, msg, err := c.ReadMessage()
// 			if err != nil {
// 				log.Println("ReadMessage() error:", err)
// 				return
// 			}

// 			log.Printf("Received: %s", msg)
// 			if err := b.handleMessage(ctx, mt, msg); err != nil {
// 				log.Println("Handling message error:", err)
// 			}
// 		}
// 	}()

// 	// Main process loop:
// 	for {
// 		select {
// 		case <-done:
// 			return
// 		case m := <-b.send:
// 			err := b.sendMessage(c, m)
// 			if err != nil {
// 				log.Println("Write error:", err)
// 				continue
// 			}

// 		case <-interrupt:
// 			log.Println("Caught interrupt signal - quitting!")
// 			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

// 			if err != nil {
// 				log.Println("Write close error:", err)
// 				return
// 			}
// 			select {
// 			case <-done:
// 				// Websocket read goroutine has shut down.
// 			case <-time.After(2 * time.Second):
// 				// Webosocket read goroutine did not shut down after 2 seconds.
// 			}
// 			return
// 		}
// 	}
// }
