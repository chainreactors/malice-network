package configs

import (
	"github.com/chainreactors/logs"
	"github.com/gookit/config/v2"
)

var ListenerConfigFileName = "listener.yaml"

func GetListenerConfig() *ListenerConfig {
	l := &ListenerConfig{}
	err := config.MapStruct("listeners", l)
	if err != nil {
		logs.Log.Errorf("Failed to map listener config %s", err)
		return nil
	}
	return l
}

type ListenerConfig struct {
	Host          string                `host:"name"`
	Name          string                `config:"name"`
	ServerAddr    string                `config:"server_addr"`
	TcpPipelines  []*TcpPipelineConfig  `config:"tcp"`
	HttpPipelines []*HttpPipelineConfig `config:"http"`
}

type TcpPipelineConfig struct {
	Enable bool   `config:"enable"`
	Name   string `config:"name"`
	Host   string `config:"host"`
	Port   uint16 `config:"port"`
}

type HttpPipelineConfig struct {
	Enable bool   `config:"enable"`
	Name   string `config:"name"`
	Host   string `config:"host"`
	Port   uint16 `config:"port"`
}
