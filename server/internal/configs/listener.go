package configs

import (
	"crypto/x509/pkix"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/helper"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
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
	Name          string                `config:"name"`
	Enable        bool                  `config:"enable"`
	Auth          string                `config:"auth"`
	TcpPipelines  []*TcpPipelineConfig  `config:"tcp"`
	HttpPipelines []*HttpPipelineConfig `config:"http"`
	Websites      []*WebsiteConfig      `config:"websites"`
}

type TcpPipelineConfig struct {
	Enable           bool              `config:"enable"`
	Name             string            `config:"name"`
	Host             string            `config:"host"`
	Port             uint16            `config:"port"`
	TlsConfig        *TlsConfig        `config:"tls"`
	EncryptionConfig *EncryptionConfig `config:"encryption"`
}

type HttpPipelineConfig struct {
	Enable    bool       `config:"enable"`
	Name      string     `config:"name"`
	Host      string     `config:"host"`
	Port      uint16     `config:"port"`
	TlsConfig *TlsConfig `config:"tls"`
}

type WebsiteConfig struct {
	Enable      bool       `config:"enable"`
	RootPath    string     `config:"rootPath"`
	WebsiteName string     `config:"websiteName"`
	Port        uint16     `config:"port"`
	TlsConfig   *TlsConfig `config:"tls"`
}

type TlsConfig struct {
	Enable   bool   `config:"enable"`
	Name     string `config:"name"`
	CN       string `config:"CN"`
	O        string `config:"O"`
	C        string `config:"C"`
	L        string `config:"L"`
	OU       string `config:"OU"`
	ST       string `config:"ST"`
	Validity string `config:"validity"`
	CertFile string `config:"cert"`
	KeyFile  string `config:"key"`
}

func (t *TlsConfig) ToPkix() *pkix.Name {
	return &pkix.Name{
		Organization:       []string{t.O},
		Country:            []string{t.C},
		Locality:           []string{t.L},
		OrganizationalUnit: []string{t.OU},
		Province:           []string{t.ST},
	}
}

func (t *TlsConfig) ToProtobuf() *lispb.TLS {
	return &lispb.TLS{
		Cert:   t.CertFile,
		Key:    t.KeyFile,
		Enable: t.Enable,
	}
}

func GenerateTlsConfig(name string) TlsConfig {
	subject := certs.RandomSubject(name)
	return TlsConfig{
		Name: name,
		CN:   subject.CommonName,
		O:    helper.JoinStringSlice(subject.Organization),
		C:    helper.JoinStringSlice(subject.Country),
		L:    helper.JoinStringSlice(subject.Locality),
		OU:   helper.JoinStringSlice(subject.OrganizationalUnit),
		ST:   helper.JoinStringSlice(subject.Province),
	}
}

type EncryptionConfig struct {
	Enable bool   `config:"enable"`
	Type   string `config:"type"`
	Key    string `config:"key"`
}
