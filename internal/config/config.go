package config

import (
	"strings"
	"time"

	"github.com/redgoat650/barnacle-net/internal/message"
	"github.com/spf13/viper"
)

const (
	ClientTimeoutKey = "clientTimeout"

	NodeNameConfigKey        = "node.name"
	NodeLabelsConfigKey      = "node.labels"
	NodeOrientationConfigKey = "node.orientation"

	NodesConfigKey = "nodes.config"

	DeployImageCfgPath          = "deploy.image"       // Deploy node - image to deploy
	DeployNodesCfgPath          = "deploy.nodes"       // Deploy node, set config - list of node configs for deploy/set config
	ConnectServerAddrCfgPath    = "connect.serveraddr" // Deploy node - Set to the server host address
	ConnectWebsocketPathCfgPath = "connect.wspath"     // Deploy node - Set the path to the websocket endpoint

	DeployServerPortConfigKey = "deploy.server.port" // Deploy server - Set to the port to serve the server over

	DefaultDeployImage = "redgoat650/barnacle-net:scratch"
)

func init() {
	viper.SetDefault(DeployServerPortConfigKey, "8080")
	viper.SetDefault(ConnectWebsocketPathCfgPath, "/ws")
	viper.SetDefault(ClientTimeoutKey, 60*time.Second)
	viper.SetDefault(NodeOrientationConfigKey, message.ButtonsL)
	viper.SetDefault(DeployImageCfgPath, DefaultDeployImage)
}

func TranslateOrientation(o string) (message.Orientation, bool) {
	switch strings.ToLower(o) {
	case "w", "west", "l", "left":
		return message.ButtonsL, true
	case "n", "north", "u", "up":
		return message.ButtonsU, true
	case "e", "east", "r", "right":
		return message.ButtonsR, true
	case "s", "south", "d", "down":
		return message.ButtonsD, true
	}

	return "", false
}
