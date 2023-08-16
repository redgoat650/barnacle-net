package config

import (
	"time"

	"github.com/spf13/viper"
)

const (
	WSPathConfigKey  = "wspath"
	PortConfigKey    = "port"
	ServerConfigKey  = "server"
	ClientTimeoutKey = "clientTimeout"
)

func init() {
	viper.SetDefault(PortConfigKey, "8080")
	viper.SetDefault(ServerConfigKey, "localhost:8080")
	viper.SetDefault(WSPathConfigKey, "/ws")
	viper.SetDefault(ClientTimeoutKey, 60*time.Second)
}
