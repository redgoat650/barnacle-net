package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"time"

	"github.com/redgoat650/barnacle-net/internal/config"
	"github.com/redgoat650/barnacle-net/internal/deploy"
	"github.com/redgoat650/barnacle-net/internal/hash"
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

	return displayJSON(resp.Payload.ListNodesResponse)
}

func AddWallet(walletID, desc, prof string) error {
	t, err := connect()
	if err != nil {
		return err
	}

	defer func() {
		fmt.Println("closing websocket:", t.GracefullyClose())
	}()

	cmd := &message.Command{
		Op: message.AddWalletCmd,
		Payload: &message.CommandPayload{
			AddWalletPayload: &message.AddWalletPayload{
				WalletID:    walletID,
				Description: desc,
				UseProfile:  prof,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := t.SendCommandWaitResponse(ctx, cmd)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("error adding wallet %s: %s", walletID, resp.Error)
	}

	return nil
}

func ListWallets() ([]message.WalletInfo, error) {
	t, err := connect()
	if err != nil {
		return nil, err
	}

	cmd := &message.Command{
		Op: message.GetWalletsCmd,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := t.SendCommandWaitResponse(ctx, cmd)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("error listing wallets: %s", resp.Error)
	}

	return resp.Payload.GetWalletsResponse.Items, nil
}

func RemoveWallet(walletID string) error {
	t, err := connect()
	if err != nil {
		return err
	}

	defer func() {
		fmt.Println("closing websocket:", t.GracefullyClose())
	}()

	cmd := &message.Command{
		Op: message.RemoveWalletCmd,
		Payload: &message.CommandPayload{
			RemoveWalletPayload: &message.RemoveWalletPayload{
				WalletID: walletID,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := t.SendCommandWaitResponse(ctx, cmd)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("error removing wallet %s: %s", walletID, resp.Error)
	}

	return nil
}

func CreateBCProfile(name, chain, apiKey string) error {
	t, err := connect()
	if err != nil {
		return err
	}

	defer func() {
		fmt.Println("closing websocket:", t.GracefullyClose())
	}()

	cmd := &message.Command{
		Op: message.AddBlockchainAPIProfileCmd,
		Payload: &message.CommandPayload{
			AddBlockchainAPIProfilePayload: &message.AddBlockchainAPIProfilePayload{
				Name:   name,
				Chain:  chain,
				APIKey: []byte(apiKey),
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := t.SendCommandWaitResponse(ctx, cmd)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("error sending add blockchain API profile: %s", resp.Error)
	}

	return nil
}

func ListProfiles() ([]message.BCProfile, error) {
	t, err := connect()
	if err != nil {
		return nil, err
	}

	cmd := &message.Command{
		Op: message.GetBlockchainAPIProfilesCmd,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := t.SendCommandWaitResponse(ctx, cmd)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("error listing profiles: %s", resp.Error)
	}

	return resp.Payload.GetBlockchainAPIProfilesResponse.Items, nil
}

func RemoveBCProfile(name string) error {
	t, err := connect()
	if err != nil {
		return err
	}

	defer func() {
		fmt.Println("closing websocket:", t.GracefullyClose())
	}()

	cmd := &message.Command{
		Op: message.RemoveBlockchainAPIProfileCmd,
		Payload: &message.CommandPayload{
			RemoveBlockchainAPIProfilePayload: &message.RemoveBlockchainAPIProfilePayload{
				Name: name,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := t.SendCommandWaitResponse(ctx, cmd)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("error sending add blockchain API profile: %s", resp.Error)
	}

	return nil
}

func displayJSON(p any) error {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(b))
	return nil
}

func ConfigSet(cfgs ...deploy.NodeDeploySettings) error {
	t, err := connect()
	if err != nil {
		return err
	}

	cfgMap := make(map[string]message.NodeConfig)

	for _, cfg := range cfgs {
		cfgMap[cfg.Name] = cfg.Config
	}

	c := &message.Command{
		Op: message.ConfigSetCmd,
		Payload: &message.CommandPayload{
			ConfigSetPayload: &message.ConfigSetPayload{
				Configs: cfgMap,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Second)
	defer cancel()

	resp, err := t.SendCommandWaitResponse(ctx, c)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("error from request: %s", resp.Error)
	}

	return nil
}

func ShowImage(node string, fit string, imgPaths ...string) error {
	t, err := connect()
	if err != nil {
		return err
	}

	defer func() {
		fmt.Println("closing websocket:", t.GracefullyClose())
	}()

	c, err := makeShowImageCmd(node, fit, imgPaths...)
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

func ListFiles() error {
	t, err := connect()
	if err != nil {
		return err
	}

	defer func() {
		fmt.Println("closing websocket:", t.GracefullyClose())
	}()

	c := makeListFilesCmd()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := t.SendCommandWaitResponse(ctx, c)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("error from request: %s", resp.Error)
	}

	return displayJSON(resp.Payload.ListFilesResponse)
}

func makeListFilesCmd() *message.Command {
	return &message.Command{
		Op: message.ListFilesCmd,
	}
}

func makeShowImageCmd(node string, fit string, imgPaths ...string) (*message.Command, error) {
	irefs, err := makeImageRefs(imgPaths...)
	if err != nil {
		return nil, err
	}

	fitMsg, err := fitStrToPolicy(fit)
	if err != nil {
		return nil, err
	}

	return &message.Command{
		Op: message.ShowImagesCmd,
		Payload: &message.CommandPayload{
			ShowImagesPayload: &message.ShowImagesPayload{
				Images:    irefs,
				FitPolicy: fitMsg,
			},
		},
	}, nil
}

func fitStrToPolicy(s string) (message.FitPolicy, error) {
	switch s {
	case "crop", message.CropToFit:
		return message.CropToFit, nil
	case "pad", message.PadToFit:
		return message.PadToFit, nil
	default:
		return "", fmt.Errorf("unrecognized fit type: %s", s)
	}
}

func makeImageRefs(imgPaths ...string) ([]message.ImageData, error) {
	var ret []message.ImageData
	for _, imgPath := range imgPaths {
		if imgData, ok := tryPathAsURL(imgPath); ok {
			ret = append(ret, *imgData)
			continue
		}

		b, h, err := hash.ReadHashFile(imgPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read/hash file %s: %v", imgPath, err)
		}

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

	h, err := hash.HashBytes(b)
	if err != nil {
		log.Println("hashing data:", err)
		return nil, false
	}

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
	server := viper.GetString(config.ConnectServerAddrCfgPath)
	path := viper.GetString(config.ConnectWebsocketPathCfgPath)

	fmt.Println("Connecting to:", server, "at", path)

	t, err := transport.NewTransportConn(server, path)
	if err != nil {
		return nil, fmt.Errorf("instantiating transport: %s", err)
	}

	return t, nil
}
