package barnacle

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	"github.com/redgoat650/barnacle-net/internal/config"
	"github.com/redgoat650/barnacle-net/internal/message"
	"github.com/redgoat650/barnacle-net/internal/python"
	"github.com/redgoat650/barnacle-net/internal/transport"
	"github.com/spf13/viper"
)

const (
	registerTimeout  = 10 * time.Second
	reconnectBackoff = 10 * time.Second
	imgCacheDir      = "barnacle-images"
)

type Barnacle struct {
	imageDir      string
	imagePYRunner *python.PyRunner
	t             *transport.Transport
}

func RunBarnacle(v *viper.Viper) error {
	var reconnectRetries int
	for {
		err := runBarnacle(v)
		if err != nil {
			if errors.Is(err, ErrInterrupt) {
				log.Println("node shutting down:", err)
				return err
			}
			log.Println("error running barnacle:", err)
		}

		reconnectRetries++
		log.Printf("attempting reconnect %d in %v", reconnectRetries, reconnectBackoff)
		time.Sleep(reconnectBackoff)
	}
}

func runBarnacle(v *viper.Viper) error {
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
	return b.handleIncomingCmds()
}

func NewBarnacle(server, path string) (*Barnacle, error) {
	t, err := transport.NewTransportConn(server, path)
	if err != nil {
		return nil, err
	}

	imageDir := filepath.Join(os.TempDir(), imgCacheDir)

	err = os.MkdirAll(imageDir, 0644)
	if err != nil {
		panic(err)
	}

	b := &Barnacle{
		imageDir:      imageDir,
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
		Op: message.RegisterCmd,
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

var (
	ErrInterrupt       = errors.New("received interrupt")
	ErrTransportClosed = errors.New("transport layer has closed the websocket")
)

func (b *Barnacle) handleIncomingCmds() error {
	// Handle interrupts.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	for {
		select {
		case cmd := <-b.t.IncomingCmds():
			if cmd == nil {
				return ErrTransportClosed
			}

			err := b.handleIncomingCommand(cmd)
			if err != nil {
				fmt.Println("Error handling incoming command", err)
			}

		case <-interrupt:
			b.handleInterrupt()
			return ErrInterrupt
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
	case message.IdentifyCmd:
		rp, err = b.handleIdentify()
	case message.SetImageCmd:
		rp, err = b.handleSetImage(cmd.Payload)

	default:
		err = fmt.Errorf("unrecognized command: %s", cmd.Op)
	}

	if err != nil {
		log.Printf("Hit error handling command %s. Attempting to send error as response: %s", cmd.Op, err)
	}

	return b.t.SendResponse(rp, err, cmd)
}

func (b *Barnacle) handleSetImage(p *message.CommandPayload) (*message.ResponsePayload, error) {
	if p == nil || p.SetImagePayload == nil {
		return nil, errors.New("invalid command payload")
	}

	imgData := p.SetImagePayload
	fileName := imgData.Name
	filePath := b.getFilePath(fileName)

	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Download the file
			fmt.Printf("download file %s to %s", fileName, filePath)
			if err := b.downloadFile(fileName); err != nil {
				return nil, err
			}
		}
	}

	err = b.imagePYRunner.RunImagePY(filePath, imgData.Saturation)
	if err != nil {
		return nil, fmt.Errorf("running image setting script: %s", err)
	}

	return nil, nil
}

func (b *Barnacle) downloadFile(fileName string) error {
	c := &message.Command{
		Op: message.GetImageCmd,
		Payload: &message.CommandPayload{
			GetImagePayload: &message.GetImagePayload{
				Name: fileName,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := b.t.SendCommandWaitResponse(ctx, c)
	if err != nil {
		return fmt.Errorf("barnacle error downloading file from server: %s", err)
	}

	if resp == nil || resp.Payload == nil || resp.Payload.GetImageResponse == nil || len(resp.Payload.GetImageResponse.ImageData) == 0 {
		return errors.New("unexpected payload returned for download image request")
	}

	filePath := b.getFilePath(fileName)

	err = os.WriteFile(filePath, resp.Payload.GetImageResponse.ImageData, 0644)
	if err != nil {
		return fmt.Errorf("unable to write image data to file %s: %s", filePath, err)
	}

	return nil
}

func (b *Barnacle) getFilePath(fileName string) string {
	return filepath.Join(b.imageDir, fileName)
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

	return &message.DisplayInfo{
		Raw: out,
	}, err
}

const (
	imgFileCachePath = "images"
	scriptsDir       = "/scripts"
)

func getScriptDir() string {
	return scriptsDir
}
