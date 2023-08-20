package message

import (
	"os"
	"time"
)

type Message struct {
	Command  *Command  `json:"command,omitempty"`
	Response *Response `json:"response,omitempty"`
}

type Command struct {
	Op      Op              `json:"op"`
	Payload *CommandPayload `json:"payload,omitempty"`

	// Set by transport layer.
	Opaque     uint64     `json:"opaque"`
	SubmitTime *time.Time `json:"submitTime,omitempty"`
	ArriveTime *time.Time `json:"arriveTime,omitempty"`
}

type Op string

const (
	ConfigSetCmd  Op = "ConfigSet"
	SetImageCmd   Op = "setImage"
	GetImageCmd   Op = "getImage"
	IdentifyCmd   Op = "identify"
	ListNodesCmd  Op = "listNodes"
	RegisterCmd   Op = "register"
	ShowImagesCmd Op = "showImages"
	ListFilesCmd  Op = "listFiles"
)

type CommandPayload struct {
	ConfigSetPayload  *ConfigSetPayload  `json:"configSetPayload,omitempty"`
	SetImagePayload   *SetImagePayload   `json:"setImagePayload,omitempty"`
	GetImagePayload   *GetImagePayload   `json:"getImagePayload,omitempty"`
	ListNodesPayload  *ListNodesPayload  `json:"listNodesPayload,omitempty"`
	RegisterPayload   *RegisterPayload   `json:"registerPayload,omitempty"`
	ShowImagesPayload *ShowImagesPayload `json:"showImagesPayload,omitempty"`
}

type ConfigSetPayload struct {
	Configs map[string]NodeConfig `json:"configs,omitempty"`
}

type NodeConfig struct {
	Labels      []string `json:"labels,omitempty"`
	Orientation *string  `json:"orientation,omitempty"`
}

type SetImagePayload struct {
	Name        string   `json:"name"`
	Hash        string   `json:"hash"`
	Saturation  *float64 `json:"saturation,omitempty"`
	RotationDeg int      `json:"rotationDeg,omitempty"`
}

type GetImagePayload struct {
	Name string `json:"name"`
}

type ListNodesPayload struct {
	RefreshIdentities bool `json:"refreshIdentities,omitempty"`
}

type RegisterPayload struct {
	Identity Identity `json:"identity,omitempty"`
}

type ShowImagesPayload struct {
	NodeName string      `json:"node,omitempty"`
	Images   []ImageData `json:"images,omitempty"`
}

type ImageData struct {
	Name   string `json:"name"`
	Origin string `json:"origin"`
	Hash   string `json:"hash"`
	Data   []byte `json:"data,omitempty"`
}

type Response struct {
	Command    *Command         `json:"command,omitempty"`
	Payload    *ResponsePayload `json:"payload,omitempty"`
	Success    bool             `json:"success"`
	Error      string           `json:"error,omitempty"`
	SubmitTime *time.Time       `json:"submitTime,omitempty"`
	ArriveTime *time.Time       `json:"arriveTime,omitempty"`
}

type ResponsePayload struct {
	GetImageResponse  *GetImageResponsePayload  `json:"getImageResponse,omitempty"`
	IdentifyResponse  *IdentifyResponsePayload  `json:"identifyResponse,omitempty"`
	ListNodesResponse *ListNodesResponsePayload `json:"listNodesResponse,omitempty"`
	ListFilesResponse *ListFilesResponsePayload `json:"listFilesResponse,omitempty"`
}

type GetImageResponsePayload struct {
	Name      string `json:"name"`
	ImageData []byte `json:"imgData,omitempty"`
	Hash      string `json:"hash"`
}

type IdentifyResponsePayload struct {
	Identity Identity `json:"identity,omitempty"`
}

type ListNodesResponsePayload struct {
	Nodes map[string]NodeStatus `json:"nodes,omitempty"`
}

type ListFilesResponsePayload struct {
	FileMap map[string][]FileInfo `json:"files,omitempty"`
}

type FileInfo struct {
	Name    string      `json:"name"`
	Size    int64       `json:"size"`
	Mode    os.FileMode `json:"mode"`
	ModTime time.Time   `json:"modTime"`
	Hash    string      `json:"hash"`
}

type Role string

const (
	NodeRole   = "node"
	ClientRole = "client"
)

type NodeStatus struct {
	UpdateTime time.Time `json:"updateTime,omitempty"`
	Identity   Identity  `json:"identity,omitempty"`
}

type Identity struct {
	Name           string       `json:"name"`
	Aliases        []string     `json:"aliases,omitempty"`
	Orientation    Orientation  `json:"orientation"`
	Role           Role         `json:"role"`
	Username       string       `json:"username"`
	Hostname       string       `json:"hostname"`
	NumCPU         int          `json:"numCPU"`
	PID            int          `json:"pid"`
	Display        *DisplayInfo `json:"display,omitempty"`
	DisplayIDError string       `json:"displayIDError,omitempty"`
}

type Orientation string

const (
	ButtonsL           Orientation = "buttonsLeft"
	ButtonsU           Orientation = "buttonsUp"
	ButtonsR           Orientation = "buttonsRight"
	ButtonsD           Orientation = "buttonsDown"
	DefaultOrientation             = ButtonsL
)

type DisplayInfo struct {
	DisplayResponding bool          `json:"displayResponding"`
	Colors            int           `json:"colorCount"`
	Width             int           `json:"xResolution"`
	Height            int           `json:"yResolution"`
	RefreshEstimate   time.Duration `json:"refreshEstimate"`
	Raw               []byte        `json:"raw,omitempty"`
}
