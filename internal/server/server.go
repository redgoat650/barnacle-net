package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redgoat650/barnacle-net/internal/config"
	"github.com/redgoat650/barnacle-net/internal/hash"
	"github.com/redgoat650/barnacle-net/internal/message"
	"github.com/redgoat650/barnacle-net/internal/transport"
	"github.com/spf13/viper"
)

const (
	defaultTimeout = 10 * time.Second
	imgCacheDir    = "barnacle-images"
)

type Server struct {
	conns  map[string]*connInfo
	connMu *sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	imgDir string
}

type connInfo struct {
	remoteAddr string
	t          *transport.Transport
	nodeStatus *message.NodeStatus
	mu         *sync.Mutex
}

func RunServer(v *viper.Viper) error {
	port := v.GetString(config.DeployServerPortConfigKey) // ":8080"
	addr := ":" + port

	s := NewServer()
	setupRoutes(s)

	log.Println("Serving at", addr)

	return http.ListenAndServe(addr, nil)
}

func NewServer() *Server {
	ctx, cancel := context.WithCancel(context.Background())

	imageDir := filepath.Join(os.TempDir(), imgCacheDir)

	err := os.MkdirAll(imageDir, 0644)
	if err != nil {
		panic(err)
	}

	return &Server{
		conns:  make(map[string]*connInfo),
		connMu: new(sync.RWMutex),
		ctx:    ctx,
		cancel: cancel,
		imgDir: imageDir,
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
		log.Println("client Connected", remoteAddr)

		t := transport.NewTransportForConn(ws)

		s.connMu.Lock()
		c := &connInfo{
			remoteAddr: remoteAddr,
			t:          t,
			mu:         new(sync.Mutex),
		}
		s.conns[remoteAddr] = c
		s.connMu.Unlock()

		defer func() {
			log.Println("shutting down client connection:", remoteAddr)
			s.connMu.Lock()
			delete(s.conns, remoteAddr)
			s.connMu.Unlock()
		}()

		s.handleIncomingCommands(c)
	}
}

func (s *Server) handleIncomingCommands(c *connInfo) {
	for {
		select {
		case cmd := <-c.t.IncomingCmds():
			err := s.handleIncomingCommand(cmd, c)
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

func (s *Server) handleIncomingCommand(cmd *message.Command, c *connInfo) error {
	if cmd == nil {
		return errors.New("transport shutting down websocket conn")
	}

	var (
		rp  *message.ResponsePayload
		err error
	)

	switch cmd.Op {
	case message.ListNodesCmd:
		rp, err = s.handleListNodes(cmd)
	case message.RegisterCmd:
		rp, err = s.handleRegister(cmd, c)
	case message.ShowImagesCmd:
		err = s.handleShowImages(cmd, c)
	case message.GetImageCmd:
		rp, err = s.handleGetImage(cmd)
	case message.ListFilesCmd:
		rp, err = s.handleListFiles(cmd)
	case message.ConfigSetCmd:
		err = s.handleConfigSet(cmd)
	default:
		err = fmt.Errorf("unrecognized command: %s", cmd.Op)
	}

	log.Println("handling command", cmd.Op, err)
	return c.t.SendResponse(rp, err, cmd)
}

func (s *Server) handleConfigSet(cmd *message.Command) error {
	p := cmd.Payload

	if p == nil || p.ConfigSetPayload == nil {
		return errors.New("invalid config set payload")
	}

	configSetPayload := p.ConfigSetPayload

	for name, cfg := range configSetPayload.Configs {
		conn, found := s.getConnInfoByName(name)
		if !found {
			return fmt.Errorf("could not find connected node with name %s", name)
		}

		c := &message.Command{
			Op: message.ConfigSetCmd,
			Payload: &message.CommandPayload{
				ConfigSetPayload: &message.ConfigSetPayload{
					Configs: map[string]message.NodeConfig{
						name: cfg,
					},
				},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resp, err := conn.t.SendCommandWaitResponse(ctx, c)
		if err != nil {
			return fmt.Errorf("sending config set command to node: %s", err)
		}

		if !resp.Success {
			return fmt.Errorf("unable to set config on node %s: %s", name, resp.Error)
		}
	}

	return nil
}

func (s *Server) getConnInfoByName(name string) (*connInfo, bool) {
	s.connMu.Lock()
	defer s.connMu.Unlock()

	for gotID, chkConn := range s.conns {
		if chkConn != nil && chkConn.nodeStatus != nil && chkConn.nodeStatus.Identity.Name == name {
			log.Println("matched node by name", name, gotID)
			return chkConn, true
		}
	}

	// Could not find a node matching this config.
	return nil, false
}

func (s *Server) handleListFiles(cmd *message.Command) (*message.ResponsePayload, error) {
	var ret []message.FileInfo

	err := filepath.Walk(s.imgDir, func(path string, info fs.FileInfo, err error) error {
		fmt.Println("WALKED TO", path)
		if err != nil {
			return fmt.Errorf("walk error: %s", err)
		}

		if info.IsDir() {
			return nil
		}

		name := info.Name()

		_, h, err := hash.ReadHashFile(path)
		if err != nil {
			return fmt.Errorf("unable to read file on server %s: %s", name, err)
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

	retMap := map[string][]message.FileInfo{
		"server": ret,
	}

	s.connMu.RLock()
	defer s.connMu.RUnlock()

	// TODO: can be done in parallel
	for nodeName, conn := range s.conns {
		conn.mu.Lock()
		defer conn.mu.Unlock()

		if conn.nodeStatus == nil || conn.nodeStatus.Identity.Role != message.NodeRole {
			log.Printf("node %s is not connected to the filesystem", nodeName)
			continue
		}

		c := &message.Command{
			Op: message.ListFilesCmd,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resp, err := conn.t.SendCommandWaitResponse(ctx, c)
		if err != nil {
			return nil, fmt.Errorf("error getting response from %s: %s", nodeName, err)
		}

		if !resp.Success {
			return nil, fmt.Errorf("unable to list files on node %s: %s", nodeName, resp.Error)
		}

		if resp == nil || resp.Payload == nil || resp.Payload.ListFilesResponse == nil {
			return nil, fmt.Errorf("invalid list files payload returned from %s", nodeName)
		}

		fm := resp.Payload.ListFilesResponse.FileMap
		if len(fm) != 1 {
			return nil, fmt.Errorf("expected 1 map entry in response from node but got %d", len(fm))
		}

		var nodeFiles []message.FileInfo
		for _, nodeFiles = range fm {
		}

		retMap[nodeName] = nodeFiles
	}

	return &message.ResponsePayload{
		ListFilesResponse: &message.ListFilesResponsePayload{
			FileMap: retMap,
		},
	}, err
}

func (s *Server) handleGetImage(cmd *message.Command) (*message.ResponsePayload, error) {
	p := cmd.Payload

	if p == nil || p.GetImagePayload == nil {
		return nil, errors.New("invalid get image payload")
	}

	fileName := p.GetImagePayload.Name

	filePath := s.imgFilePath(fileName)

	b, h, err := hash.ReadHashFile(filePath)
	if err != nil {
		return nil, err
	}

	return &message.ResponsePayload{
		GetImageResponse: &message.GetImageResponsePayload{
			Name:      fileName,
			ImageData: b,
			Hash:      h,
		},
	}, nil
}

func (s *Server) handleShowImages(cmd *message.Command, c *connInfo) error {
	p := cmd.Payload

	if p == nil || p.ShowImagesPayload == nil {
		return errors.New("invalid show images payload")
	}

	showImgPayload := p.ShowImagesPayload

	if len(showImgPayload.Images) == 0 {
		return errors.New("no images received")
	}

	for _, imgData := range showImgPayload.Images {
		err := s.saveImage(imgData)
		if err != nil {
			return fmt.Errorf("error saving image %s: %s", imgData.Name, err)
		}
	}

	s.connMu.RLock()
	defer s.connMu.RUnlock()

	var filteredConns []*connInfo

	nodeSelectors := showImgPayload.NodeSelectors
	for _, conn := range s.conns {
		// Default assume match ANY
		includeConn := true
		for _, sel := range nodeSelectors {
			match := connMatchesSelector(conn, sel)

			switch sel.Logic {
			case message.LogicAnd:
				includeConn = includeConn && match
			case message.LogicOr:
				includeConn = includeConn || match
			default:
				includeConn = includeConn && match
			}
		}

		if includeConn {
			filteredConns = append(filteredConns, conn)
		}
	}

	if len(filteredConns) == 0 {
		return errors.New("no nodes are eligible to display")
	}

	lnodes, pnodes := filterOrientations(filteredConns)

	var errs []error

	lastImgIdx := len(showImgPayload.Images) - 1

	for i := lastImgIdx; i >= 0; i-- {
		imgData := showImgPayload.Images[i]
		b := bytes.NewBuffer(imgData.Data)

		img, _, err := image.Decode(b)
		if err != nil {
			return fmt.Errorf("decoding image data: %s", err)
		}

		rect := img.Bounds()

		log.Printf("image %s is %dx%d", imgData.Name, rect.Dx(), rect.Dy())

		prefer, backup := &pnodes, &lnodes
		if rect.Dx() > rect.Dy() {
			prefer, backup = backup, prefer
		}

		var displayOnNode *connInfo
		if len(*prefer) > 0 {
			// Display available in preferred orientation
			displayOnNode = (*prefer)[0]
			*prefer = (*prefer)[1:]

			log.Printf("displaying image %s in preferred orientation on %s", imgData.Name, displayOnNode.remoteAddr)

		} else {
			if showImgPayload.MustFitOrientation {
				errs = append(errs, fmt.Errorf("orientation mismatch: no preferred orientation nodes found to display %s", imgData.Name))
				continue
			}

			if len(*backup) == 0 {
				// No more nodes to display images. We're done.
				break
			}

			displayOnNode = (*backup)[0]
			*backup = (*backup)[1:]

			log.Printf("displaying image %s in unpreferred orientation on %s", imgData.Name, displayOnNode.remoteAddr)
		}

		err = s.displayOverConn(imgData, displayOnNode, showImgPayload.FitPolicy)
		if err != nil {
			errs = append(errs, err)
		}
	}

	var retErr error
	for _, err := range errs {
		if retErr == nil {
			retErr = err
			continue
		}

		retErr = fmt.Errorf("AND %s", err)
	}

	return retErr
}

func connMatchesSelector(conn *connInfo, sel message.NodeSelector) bool {
	switch sel.Key {
	case message.MatchAnySelKey:
		return true
	case message.MatchNoneSelKey:
		return false
	case message.NameSelKey, message.NameEqualsSelKey:
		return sel.Value == conn.nodeStatus.Identity.Name
	case message.NameContainsSelKey:
		return strings.Contains(conn.nodeStatus.Identity.Name, sel.Value)
	case message.HasLabelSelKey:
		return hasLabel(conn, sel.Value)
	}

	return false
}

func hasLabel(conn *connInfo, label string) bool {
	for _, connLabel := range conn.nodeStatus.Identity.Labels {
		if connLabel == label {
			return true
		}
	}
	return false
}

func filterOrientations(conns []*connInfo) (l, p []*connInfo) {
	for _, c := range conns {
		if c.nodeStatus == nil || c.nodeStatus.Identity.Display == nil || !c.nodeStatus.Identity.Display.DisplayResponding {
			log.Printf("ignoring node %s, not ready", c.remoteAddr)
			continue
		}

		if isPortrait(c.nodeStatus.Identity) {
			p = append(p, c)
			continue
		}

		l = append(l, c)
	}

	return
}

func isPortrait(identity message.Identity) bool {
	switch identity.Orientation {
	case message.ButtonsD, message.ButtonsU:
		return true
	case message.ButtonsL, message.ButtonsR:
		return false
	}

	return false
}

func (s *Server) displayOverConn(imgData message.ImageData, conn *connInfo, fitPolicy message.FitPolicy) error {
	// connInfo should be already locked
	t := conn.t

	sat := float64(0.5)

	c := &message.Command{
		Op: message.SetImageCmd,
		Payload: &message.CommandPayload{
			SetImagePayload: &message.SetImagePayload{
				Name:       imgData.Name,
				Hash:       imgData.Hash,
				Saturation: &sat,
				FitPolicy:  fitPolicy,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := t.SendCommandWaitResponse(ctx, c)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("failed to display image: %s", resp.Error)
	}

	return nil
}

func (s *Server) saveImage(imgData message.ImageData) error {
	fullPath := s.imgFilePath(imgData.Name)

	return os.WriteFile(fullPath, imgData.Data, 0644)
}

func (s *Server) imgFilePath(name string) string {
	return filepath.Join(s.imgDir, name)
}

func (s *Server) handleRegister(cmd *message.Command, c *connInfo) (*message.ResponsePayload, error) {
	p := cmd.Payload

	if p == nil || p.RegisterPayload == nil {
		return nil, errors.New("invalid register payload")
	}

	arrTime := time.Now()
	if cmd.ArriveTime != nil {
		arrTime = *cmd.ArriveTime
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.nodeStatus = &message.NodeStatus{
		UpdateTime: arrTime,
		Identity:   cmd.Payload.RegisterPayload.Identity,
	}

	return nil, nil
}

func (s *Server) handleListNodes(cmd *message.Command) (*message.ResponsePayload, error) {
	p := cmd.Payload

	refreshIDs := false
	if p != nil && p.ListNodesPayload != nil {
		refreshIDs = p.ListNodesPayload.RefreshIdentities
	}

	s.connMu.RLock()
	defer s.connMu.RUnlock()

	if refreshIDs {
		for remoteAddr, connInfo := range s.conns {
			if connInfo == nil || connInfo.nodeStatus == nil || connInfo.nodeStatus.Identity.Role != message.NodeRole {
				log.Println("skipping refresh for node", remoteAddr)
				continue
			}

			log.Println("sending id refresh for node", remoteAddr)

			err := s.updateConnIdentity(connInfo)
			if err != nil {
				log.Println("identify failed for", remoteAddr, "error:", err)
				continue
			}
		}
	}

	nodeStatusMap := make(map[string]message.NodeStatus)
	for remoteAddr, connInfo := range s.conns {
		connInfo.mu.Lock()

		if ns := connInfo.nodeStatus; ns != nil {
			nodeStatusMap[remoteAddr] = *ns
		}

		connInfo.mu.Unlock()
	}

	return &message.ResponsePayload{
		ListNodesResponse: &message.ListNodesResponsePayload{
			Nodes: nodeStatusMap,
		},
	}, nil
}

func (s *Server) updateConnIdentity(connInfo *connInfo) error {
	ns, err := s.identifyOverConn(connInfo)
	if err != nil {
		return err
	}

	connInfo.mu.Lock()
	defer connInfo.mu.Unlock()

	connInfo.nodeStatus = ns

	return nil
}

func (s *Server) identifyOverConn(connInfo *connInfo) (*message.NodeStatus, error) {
	c := &message.Command{
		Op: message.IdentifyCmd,
	}

	resp, err := connInfo.t.SendCommand(c)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	select {
	case resp := <-resp:
		return s.handleIdentifyResponse(resp, connInfo)

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *Server) handleIdentifyResponse(resp *message.Response, connInfo *connInfo) (*message.NodeStatus, error) {
	if resp == nil {
		return nil, fmt.Errorf("inflight command response lost")
	}

	if resp.Command.Op != message.IdentifyCmd {
		return nil, fmt.Errorf("response is not for identify command: %s", resp.Command.Op)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("error in identity response: %s", resp.Error)
	}

	if resp.Payload == nil || resp.Payload.IdentifyResponse == nil {
		return nil, fmt.Errorf("identify response payload is empty")
	}

	if resp.ArriveTime == nil {
		return nil, fmt.Errorf("malformed response - arrival time unset")
	}

	return &message.NodeStatus{
		UpdateTime: *resp.ArriveTime,
		Identity:   resp.Payload.IdentifyResponse.Identity,
	}, nil
}

func setupRoutes(s *Server) {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/ws", makeWSHandler(s))
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
