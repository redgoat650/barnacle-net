package barnacle

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redgoat650/barnacle-net/internal/config"
	"github.com/redgoat650/barnacle-net/internal/hash"
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
	cfgMu         *sync.Mutex
}

func RunBarnacle() error {
	var reconnectRetries int
	for {
		err := runBarnacle()
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

func runBarnacle() error {
	b, err := NewBarnacle()
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

func NewBarnacle() (*Barnacle, error) {
	server := viper.GetString(config.ConnectServerAddrCfgPath)
	path := viper.GetString(config.ConnectWebsocketPathCfgPath)

	log.Println("connecting to:", server, "at", path)

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
		cfgMu:         new(sync.Mutex),
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
	case message.ListFilesCmd:
		rp, err = b.handleListFiles()
	case message.ConfigSetCmd:
		err = b.handleConfigSet(cmd.Payload)
	default:
		err = fmt.Errorf("unrecognized command: %s", cmd.Op)
	}

	if err != nil {
		log.Printf("Hit error handling command %s. Attempting to send error as response: %s", cmd.Op, err)
	}

	return b.t.SendResponse(rp, err, cmd)
}

func (b *Barnacle) handleConfigSet(p *message.CommandPayload) error {
	if p == nil || p.ConfigSetPayload == nil {
		return errors.New("invalid command payload")
	}

	// Ensure config changes atomically.
	b.cfgMu.Lock()
	defer b.cfgMu.Unlock()

	changed := false
	cfgPl := p.ConfigSetPayload.Configs

	if len(cfgPl) != 1 {
		return fmt.Errorf("malformed set config payload without config len 1: len is %d", len(cfgPl))
	}

	var name string
	var cfg message.NodeConfig
	for name, cfg = range cfgPl {
	}

	wantName := viper.GetString(config.NodeNameConfigKey)
	if name != wantName {
		return fmt.Errorf("malformed set config, name does not match got %s != want %s", name, wantName)
	}

	if cfg.Orientation != nil {
		viper.Set(config.NodeOrientationConfigKey, *cfg.Orientation)
		changed = true
	}

	if cfg.Labels != nil {
		viper.Set(config.NodeLabelsConfigKey, cfg.Labels)
		changed = true
	}

	if changed {
		// Register asynchronously (since it might take a bit to perform the eeprom checks).
		// Server can assume an eventual update.
		go func() {
			if err := b.Register(); err != nil {
				log.Println("node unable to re-register after config change:", err)
			}
		}()
	}

	return nil
}

func (b *Barnacle) handleListFiles() (*message.ResponsePayload, error) {
	var ret []message.FileInfo

	err := filepath.Walk(b.imageDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk error: %s", err)
		}

		name := info.Name()
		fullPath := filepath.Join(b.imageDir, name)

		log.Println("Walk reached", name, fullPath, "dir?", info.IsDir())

		if info.IsDir() {
			return nil
		}

		_, h, err := hash.ReadHashFile(fullPath)
		if err != nil {
			return fmt.Errorf("unable to read file %s: %s", fullPath, err)
		}

		ret = append(ret, message.FileInfo{
			Name:    name,
			Size:    info.Size(),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			Hash:    h,
		})

		return nil
	})

	return &message.ResponsePayload{
		ListFilesResponse: &message.ListFilesResponsePayload{
			FileMap: map[string][]message.FileInfo{
				"self": ret,
			},
		},
	}, err
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

	orient := viper.GetString(config.NodeOrientationConfigKey)
	rot := orientationToRotation(message.Orientation(orient))

	err = b.imagePYRunner.RunImagePY(filePath, rot, imgData.Saturation)
	if err != nil {
		return nil, fmt.Errorf("running image setting script: %s", err)
	}

	return nil, nil
}

func orientationToRotation(o message.Orientation) int {
	rotationDeg := 0
	switch o {
	case message.ButtonsD:
		rotationDeg = 270
	case message.ButtonsR:
		rotationDeg = 180
	case message.ButtonsU:
		rotationDeg = 90
	}
	return rotationDeg
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

	name := viper.GetString(config.NodeNameConfigKey)
	orient := viper.GetString(config.NodeOrientationConfigKey)
	aliases := viper.GetStringSlice(config.NodeLabelsConfigKey)

	return &message.Identity{
		Name:           name,
		Orientation:    message.Orientation(orient),
		Aliases:        aliases,
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

	kv := toKV(out, ":")

	w, h := displayToWH(kv)

	return &message.DisplayInfo{
		DisplayResponding: err == nil,
		Width:             w,
		Height:            h,
		RefreshEstimate:   60 * time.Second,
		Raw:               out,
	}, err
}

func toKV(b []byte, sep string) map[string]string {
	ret := make(map[string]string)
	for _, l := range strings.Split(string(b), "\n") {
		spl := strings.Split(l, ":")
		if len(spl) != 2 {
			continue
		}

		ret[spl[0]] = spl[1]
	}

	return ret
}

func displayToWH(kv map[string]string) (int, int) {
	displayStr, ok := kv["Display"]
	if !ok {
		log.Println("could not find display field")
		return 0, 0
	}

	d := strings.Split(displayStr, "x")

	if len(d) != 2 {
		log.Printf("display field %q did not split as expected", displayStr)
		return 0, 0
	}

	w, err := strconv.Atoi(strings.TrimSpace(d[0]))
	if err != nil {
		log.Printf("error parsing width %q: %s", d[0], err)
		return 0, 0
	}

	h, err := strconv.Atoi(strings.TrimSpace(d[1]))
	if err != nil {
		log.Printf("error parsing height %q: %s", d[1], err)
		return 0, 0
	}

	return w, h
}

const (
	imgFileCachePath = "images"
	scriptsDir       = "/scripts"
)

func getScriptDir() string {
	return scriptsDir
}
