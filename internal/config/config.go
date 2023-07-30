package config

import "github.com/spf13/viper"

const (
	WSPathConfigKey = "wspath"
	PortConfigKey   = "port"
	ServerConfigKey = "server"
)

func init() {
	viper.SetDefault(PortConfigKey, "8080")
	viper.SetDefault(ServerConfigKey, "localhost:8080")
	viper.SetDefault(WSPathConfigKey, "/ws")
}
