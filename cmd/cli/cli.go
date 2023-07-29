package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"os/user"
	"runtime"
	"sync"

	"github.com/redgoat650/barnacle-net/internal/message"
	"github.com/redgoat650/barnacle-net/internal/transport"
	"github.com/spf13/viper"
)

const (
	serverConfigKey = "server"
	pathConfigKey   = "path"
)

func main() {
	server := viper.GetString(serverConfigKey)
	path := viper.GetString(pathConfigKey)

	fmt.Println("Connecting to:", server, "at", path)

	t, err := transport.NewTransportConn(server, path)
	if err != nil {
		log.Println("instantiating transport:", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go handleIncomingCmds(ctx, t, wg)

	issueCmd(t)

	cancel()

	wg.Wait()
}

func issueCmd(t *transport.Transport) {
	log.Println("TODO")
}

func handleIncomingCmds(ctx context.Context, t *transport.Transport, wg *sync.WaitGroup) {
	defer wg.Done()

	// Handle interrupts.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	for {
		select {
		case cmd := <-t.IncomingCmds():
			err := handleCommand(cmd, t)
			if err != nil {
				fmt.Println("Error handling incoming command:", err)
			}

		case <-ctx.Done():
			handleCtxDone(ctx, t)
			return
		case <-interrupt:
			handleInterrupt(t)
			return
		}
	}
}

func handleCommand(cmd *message.Command, t *transport.Transport) error {
	if cmd == nil {
		return errors.New("transport layer has closed the websocket")
	}

	var (
		rp  *message.ResponsePayload
		err error
	)

	switch cmd.Op {
	case message.Identify:
		rp, err = handleIdentify()
	default:
		return fmt.Errorf("unknown command %s", cmd.Op)
	}

	return t.SendResponse(rp, err, cmd)
}

func handleIdentify() (*message.ResponsePayload, error) {
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
			Role:     message.ClientRole,
			Username: user.Name,
			Hostname: host,
			NumCPU:   runtime.NumCPU(),
			PID:      os.Getpid(),
		},
	}, nil
}

func handleCtxDone(ctx context.Context, t *transport.Transport) {
	log.Println("Context done:", ctx.Err())
	log.Println("Websocket close error:", t.GracefullyClose()) // Blocks until incoming cmds channel closes
}

func handleInterrupt(t *transport.Transport) {
	log.Println("Caught interrupt signal - gracefully disconnecting websocket")
	log.Println("Websocket close error:", t.GracefullyClose()) // Blocks until incoming cmds channel closes
}

func init() {
	viper.SetDefault(serverConfigKey, "localhost:8080")
	viper.SetDefault(pathConfigKey, "/ws")
}
