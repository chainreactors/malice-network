package configs

import (
	"fmt"
	"github.com/gookit/config/v2"
)

func GetServerConfig() *ServerConfig {
	s := &ServerConfig{}
	err := config.MapStruct("server", s)
	if err != nil {
		return nil
	}
	return s
}

type ServerConfig struct {
	GRPCPort uint16 `config:"grpc_port"`
	GRPCHost string `config:"grpc_host"`
}

func (c *ServerConfig) String() string {
	return fmt.Sprintf("%s:%d", c.GRPCHost, c.GRPCPort)
}
