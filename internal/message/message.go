package message

import "time"

type Message struct {
	Command  *Command  `json:"command,omitempty"`
	Response *Response `json:"response,omitempty"`
}

type Command struct {
	Op         Op              `json:"op"`
	Opaque     uint64          `json:"opaque"`
	Payload    *CommandPayload `json:"payload,omitempty"`
	SubmitTime *time.Time      `json:"submitTime,omitempty"`
	ArriveTime *time.Time      `json:"arriveTime,omitempty"`
}

type Op string

const (
	SetImageCmd Op = "setImage"
	GetImageCmd Op = "getImage"
	Identify    Op = "identify"
	ListNodes   Op = "listNodes"
)

type CommandPayload struct {
	SetImagePayload  *SetImagePayload  `json:"setImagePayload,omitempty"`
	GetImagePayload  *GetImagePayload  `json:"getImagePayload,omitempty"`
	ListNodesPayload *ListNodesPayload `json:"listNodesPayload,omitempty"`
}

type SetImagePayload struct {
	Name       string   `json:"name"`
	Saturation *float64 `json:"saturation,omitempty"`
}

type GetImagePayload struct {
	Name string `json:"name"`
}

type ListNodesPayload struct {
	RefreshIdentities bool `json:"refreshIdentities,omitempty"`
}

type Response struct {
	Command    *Command         `json:"command,omitempty"`
	Success    bool             `json:"success"`
	Payload    *ResponsePayload `json:"payload,omitempty"`
	Error      string           `json:"error,omitempty"`
	SubmitTime *time.Time       `json:"submitTime,omitempty"`
	ArriveTime *time.Time       `json:"arriveTime,omitempty"`
}

type ResponsePayload struct {
	GetImageResponse  *GetImageResponsePayload  `json:"getImageResponse,omitempty"`
	IdentifyResponse  *IdentifyResponsePayload  `json:"identifyResponse,omitempty"`
	ListNodesResponse *ListNodesResponsePayload `json:"listNodesResponse,omitempty"`
}

type GetImageResponsePayload struct {
	Name      string `json:"name"`
	ImageData []byte `json:"imgData,omitempty"`
}

type IdentifyResponsePayload struct {
	Identity Identity `json:"identity,omitempty"`
}

type ListNodesResponsePayload struct {
	Nodes map[string]NodeStatus `json:"nodes,omitempty"`
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
	Role     Role   `json:"role"`
	Username string `json:"username"`
	Hostname string `json:"hostname"`
	NumCPU   int    `json:"numCPU"`
	PID      int    `json:"pid"`
}
