package message

type Message struct {
	Command  *Command  `json:"command,omitempty"`
	Response *Response `json:"response,omitempty"`
}

type Command struct {
	Op      Op              `json:"op"`
	Opaque  uint64          `json:"opaque"`
	Payload *CommandPayload `json:"payload,omitempty"`
}

type CommandPayload struct {
	SetImagePayload *SetImagePayload `json:"setImagePayload,omitempty"`
	GetImagePayload *GetImagePayload `json:"getImagePayload,omitempty"`
}

type Op string

const (
	SetImageCmd Op = "setImage"
	GetImageCmd Op = "getImage"
	Identify    Op = "identify"
)

type SetImagePayload struct {
	Name       string   `json:"name"`
	Saturation *float64 `json:"saturation,omitempty"`
}

type GetImagePayload struct {
	Name string `json:"name"`
}

type Response struct {
	Command *Command         `json:"command,omitempty"`
	Success bool             `json:"success"`
	Payload *ResponsePayload `json:"payload,omitempty"`
	Error   string           `json:"error,omitempty"`
}

type ResponsePayload struct {
	GetImageResponse *GetImageResponsePayload `json:"getImageResponse,omitempty"`
	IdentifyResponse *IdentifyResponsePayload `json:"identifyResponse,omitempty"`
}

type GetImageResponsePayload struct {
	Name      string `json:"name"`
	ImageData []byte `json:"imgData,omitempty"`
}

type IdentifyResponsePayload struct {
	Role     Role   `json:"role"`
	Username string `json:"username"`
	Hostname string `json:"hostname"`
	NumCPU   int    `json:"numCPU"`
}

type Role string

const (
	NodeRole   = "node"
	ClientRole = "client"
)
