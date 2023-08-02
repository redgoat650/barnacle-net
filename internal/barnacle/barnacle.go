package barnacle

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"os/user"
	"runtime"
	"time"

	"github.com/redgoat650/barnacle-net/internal/config"
	"github.com/redgoat650/barnacle-net/internal/message"
	"github.com/redgoat650/barnacle-net/internal/python"
	"github.com/redgoat650/barnacle-net/internal/transport"
	"github.com/spf13/viper"
)

const (
	registerTimeout = 10 * time.Second
)

type Barnacle struct {
	imagePYRunner *python.PyRunner
	t             *transport.Transport
}

func RunBarnacle(v *viper.Viper) error {
	server := v.GetString(config.ServerConfigKey)
	path := v.GetString(config.WSPathConfigKey)
	log.Println("connecting to:", server, "at", path)

	b, err := NewBarnacle(server, path)
	if err != nil {
		return fmt.Errorf("instantiating barnacle: %s", err)
	}

	// Register host with the server.
	if err := b.Register(); err != nil {
		log.Println("closing connection:", b.t.GracefullyClose())
		return fmt.Errorf("failed to register with server: %s", err)
	}

	// Block while handling incoming commands.
	err = b.handleIncomingCmds()

	log.Println("Node shutting down:", err)

	return nil
}

func NewBarnacle(server, path string) (*Barnacle, error) {
	t, err := transport.NewTransportConn(server, path)
	if err != nil {
		return nil, err
	}

	b := &Barnacle{
		imagePYRunner: python.NewImagePYRunner(getScriptDir()),
		t:             t,
	}

	return b, nil
}

func (b *Barnacle) Register() error {
	id, err := b.getIdentity()
	if err != nil {
		return err
	}

	c := &message.Command{
		Op: message.Register,
		Payload: &message.CommandPayload{
			RegisterPayload: &message.RegisterPayload{
				Identity: *id,
			},
		},
	}

	respCh, err := b.t.SendCommand(c)
	if err != nil {
		return err
	}

	resp, err := transport.WaitOnResponse(respCh, registerTimeout)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("register error returned from server: %s", resp.Error)
	}

	return nil
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

			err := b.handleIncomingCommand(cmd)
			if err != nil {
				fmt.Println("Error handling incoming command", err)
			}

		case <-interrupt:
			b.handleInterrupt()
			return nil
		}
	}
}

func (b *Barnacle) handleInterrupt() {
	log.Println("Caught interrupt signal - gracefully disconnecting websocket")
	log.Println("Websocket close error:", b.t.GracefullyClose()) // Blocks until incoming cmds channel closes
}

func (b *Barnacle) handleIncomingCommand(cmd *message.Command) error {
	var (
		rp  *message.ResponsePayload
		err error
	)

	switch cmd.Op {
	case message.Identify:
		rp, err = b.handleIdentify()
	default:
		err = fmt.Errorf("unrecognized command: %s", cmd.Op)
	}

	if err != nil {
		log.Printf("Hit error handling command %s. Attempting to send error as response: %s", cmd.Op, err)
	}

	return b.t.SendResponse(rp, err, cmd)
}

func (b *Barnacle) handleIdentify() (*message.ResponsePayload, error) {
	rp, err := b.makeIDResponsePayload()
	if err != nil {
		return nil, err
	}

	return &message.ResponsePayload{
		IdentifyResponse: rp,
	}, nil
}

func (b *Barnacle) makeIDResponsePayload() (*message.IdentifyResponsePayload, error) {
	id, err := b.getIdentity()
	if err != nil {
		return nil, err
	}

	return &message.IdentifyResponsePayload{
		Identity: *id,
	}, nil
}

func (b *Barnacle) getIdentity() (*message.Identity, error) {
	user, err := user.Current()
	if err != nil {
		return nil, err
	}

	host, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	display, err := b.detectDisplay()
	var errMsg string
	if err != nil {
		log.Println("error detecting display:", err)
		errMsg = err.Error()
		// Continue to identify anyway, display will be nil.
	}

	return &message.Identity{
		Role:           message.NodeRole,
		Username:       user.Name,
		Hostname:       host,
		NumCPU:         runtime.NumCPU(),
		PID:            os.Getpid(),
		Display:        display,
		DisplayIDError: errMsg,
	}, nil
}

func (b *Barnacle) detectDisplay() (*message.DisplayInfo, error) {
	out, err := b.imagePYRunner.RunIdentifyPY()
	if err != nil {
		return nil, err
	}

	return &message.DisplayInfo{
		Raw: out,
	}, nil
}

const (
	imgFileCachePath = "images"
	scriptsDir       = "/scripts"
)

func getScriptDir() string {
	return scriptsDir
}
