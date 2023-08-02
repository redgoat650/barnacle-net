package client

import (
	"encoding/json"
	"fmt"

	"github.com/redgoat650/barnacle-net/internal/config"
	"github.com/redgoat650/barnacle-net/internal/message"
	"github.com/redgoat650/barnacle-net/internal/transport"
	"github.com/spf13/viper"
)

func ListNodes(refresh bool) error {
	t, err := connect()
	if err != nil {
		return err
	}

	defer func() {
		fmt.Println("closing websocket:", t.GracefullyClose())
	}()

	c := makeListNodesCmd(refresh)

	respCh, err := t.SendCommand(c)
	if err != nil {
		return err
	}

	resp, err := transport.WaitOnResponse(respCh, viper.GetDuration(config.ClientTimeoutKey))
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("error from request: %s", resp.Error)
	}

	if resp.Payload == nil || resp.Payload.ListNodesResponse == nil {
		return fmt.Errorf("malformatted response")
	}

	b, err := json.MarshalIndent(resp.Payload.ListNodesResponse, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(b))

	return nil
}

func ShowImage(node string, imgPaths ...string) error {
	t, err := connect()
	if err != nil {
		return err
	}

	defer func() {
		fmt.Println("closing websocket:", t.GracefullyClose())
	}()

	c := makeShowImageCmd()
}

func makeShowImageCmd(img string) *message.Command {
	return &message.Command{
		Op: message.SetImageCmd,
		Payload: &message.CommandPayload{
			SetImagePayload: &message.SetImagePayload{
				Name: img,
			},
		},
	}
}

func makeListNodesCmd(refresh bool) *message.Command {
	return &message.Command{
		Op: message.ListNodes,
		Payload: &message.CommandPayload{
			ListNodesPayload: &message.ListNodesPayload{
				RefreshIdentities: refresh,
			},
		},
	}
}

func connect() (*transport.Transport, error) {
	server := viper.GetString(config.ServerConfigKey)
	path := viper.GetString(config.WSPathConfigKey)

	fmt.Println("Connecting to:", server, "at", path)

	t, err := transport.NewTransportConn(server, path)
	if err != nil {
		return nil, fmt.Errorf("instantiating transport: %s", err)
	}

	return t, nil
}
