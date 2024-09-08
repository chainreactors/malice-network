package configs

import (
	"crypto/x509/pkix"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/helper"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
)

var ListenerConfigFileName = "listener.yaml"

type ListenerConfig struct {
	Enable       bool                 `config:"enable" default:"true"`
	Name         string               `config:"name" default:"listener"`
	Auth         string               `config:"auth" default:"listener.auth"`
	TcpPipelines []*TcpPipelineConfig `config:"tcp" default:""`
	//HttpPipelines []*HttpPipelineConfig `config:"http" default:""`
	Websites []*WebsiteConfig `config:"websites" default:""`
}

type TcpPipelineConfig struct {
	Enable           bool              `config:"enable" default:"false"`
	Name             string            `config:"name" default:"tcp"`
	Host             string            `config:"host" default:"0.0.0.0"`
	Port             uint16            `config:"port" default:"5001"`
	TlsConfig        *TlsConfig        `config:"tls" default:""`
	EncryptionConfig *EncryptionConfig `config:"encryption" default:""`
}

type HttpPipelineConfig struct {
	Enable    bool       `config:"enable" default:"false"`
	Name      string     `config:"name" default:"http"`
	Host      string     `config:"host" default:"0.0.0.0"`
	Port      uint16     `config:"port" default:"8443"`
	TlsConfig *TlsConfig `config:"tls" default:""`
}

type WebsiteConfig struct {
	Enable      bool       `config:"enable" default:"false"`
	RootPath    string     `config:"root" default:"."`
	WebsiteName string     `config:"name" default:"web"`
	Port        uint16     `config:"port" default:"443"`
	TlsConfig   *TlsConfig `config:"tls" default:""`
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
