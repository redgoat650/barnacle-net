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
	"github.com/spf13/viper"
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

func getScriptPath() string {
	return filepath.Join(os.TempDir(), imgPyScriptPath)
}

func getServerConnInfo() (string, string, error) {
	arguments := os.Args
	if len(arguments) != 3 {
		return "", "", errors.New("must be initialized with two arguments: server and path")
	}

	server := arguments[1]
	path := arguments[2]

	return server, path, nil
}

const (
	serverConfigKey = "server"
	pathConfigKey   = "path"
)

func main() {
	server := viper.GetString(serverConfigKey)
	path := viper.GetString(pathConfigKey)
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

func init() {
	viper.SetDefault(serverConfigKey, "localhost:8080")
	viper.SetDefault(pathConfigKey, "/ws")
}
