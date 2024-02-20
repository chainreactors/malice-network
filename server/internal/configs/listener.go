package configs

import (
	"crypto/x509/pkix"
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
	TlsConfig     *TlsConfig            `config:"tls"`
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

type TlsConfig struct {
	Enable   bool   `config:"enable"`
	CN       string `config:"CN"`
	O        string `config:"O"`
	C        string `config:"C"`
	L        string `config:"L"`
	OU       string `config:"OU"`
	ST       string `config:"ST"`
	Validity string `config:"validity"`
}

func (t *TlsConfig) ToPkix() *pkix.Name {
	return &pkix.Name{
		CommonName:         t.CN,
		Organization:       []string{t.O},
		Country:            []string{t.C},
		Locality:           []string{t.L},
		OrganizationalUnit: []string{t.OU},
		Province:           []string{t.ST},
	}
}
