package client

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"

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

	c, err := makeShowImageCmd(node, imgPaths...)
	if err != nil {
		return err
	}

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

	fmt.Println("success")

	return nil
}

func makeShowImageCmd(node string, imgPaths ...string) (*message.Command, error) {
	irefs, err := makeImageRefs(imgPaths...)
	if err != nil {
		return nil, err
	}

	return &message.Command{
		Op: message.ShowImagesCmd,
		Payload: &message.CommandPayload{
			ShowImagesPayload: &message.ShowImagesPayload{
				Images: irefs,
			},
		},
	}, nil
}

func makeImageRefs(imgPaths ...string) ([]message.ImageData, error) {
	var ret []message.ImageData
	for _, imgPath := range imgPaths {
		if imgData, ok := tryPathAsURL(imgPath); ok {
			ret = append(ret, *imgData)
			continue
		}

		b, err := os.ReadFile(imgPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read file %s: %v", imgPath, err)
		}

		h := sha256.Sum256(b)

		_, fp := path.Split(imgPath)

		ret = append(ret, message.ImageData{
			Name:   fp,
			Origin: imgPath,
			Hash:   h,
			Data:   b,
		})
	}

	return ret, nil
}

func tryPathAsURL(imgPath string) (*message.ImageData, bool) {
	resp, err := http.Get(imgPath)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()

	_, fp := path.Split(imgPath)

	buf := &bytes.Buffer{}

	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		log.Println("error copying http body to a buffer:", err)
		return nil, false
	}

	b := buf.Bytes()

	h := sha256.Sum256(b)

	return &message.ImageData{
		Name:   fp,
		Origin: imgPath,
		Hash:   h,
		Data:   b,
	}, true
}

func makeListNodesCmd(refresh bool) *message.Command {
	return &message.Command{
		Op: message.ListNodesCmd,
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
