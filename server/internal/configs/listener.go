package configs

import (
	"crypto/x509/pkix"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/consts"
	cryptostream "github.com/chainreactors/malice-network/helper/cryptography/stream"
	"github.com/chainreactors/malice-network/helper/proto/listener/lispb"
	"os"
)

var ListenerConfigFileName = "listener.yaml"

type ListenerConfig struct {
	Enable       bool                 `config:"enable" default:"true"`
	Name         string               `config:"name" default:"listener"`
	Auth         string               `config:"auth" default:"listener.auth"`
	TcpPipelines []*TcpPipelineConfig `config:"tcp" `
	//HttpPipelines []*HttpPipelineConfig `config:"http" default:""`
	Websites []*WebsiteConfig `config:"websites"`
}

type TcpPipelineConfig struct {
	Enable           bool              `config:"enable" default:"true"`
	Name             string            `config:"name" default:"tcp"`
	Host             string            `config:"host" default:"0.0.0.0"`
	Port             uint16            `config:"port" default:"5001"`
	TlsConfig        *TlsConfig        `config:"tls"`
	EncryptionConfig *EncryptionConfig `config:"encryption"`
}

func (tcpPipeline *TcpPipelineConfig) ToProtobuf(lisId string) *lispb.Pipeline {
	tls, err := tcpPipeline.TlsConfig.ReadCert()
	if err != nil {
		panic(err.Error())
	}
	return &lispb.Pipeline{
		Body: &lispb.Pipeline_Tcp{
			Tcp: &lispb.TCPPipeline{
				Name:       tcpPipeline.Name,
				Host:       tcpPipeline.Host,
				Port:       uint32(tcpPipeline.Port),
				ListenerId: lisId,
			},
		},
		Tls:        tls.ToProtobuf(),
		Encryption: tcpPipeline.EncryptionConfig.ToProtobuf(),
	}
}

type HttpPipelineConfig struct {
	Enable    bool       `config:"enable" default:"false"`
	Name      string     `config:"name" default:"http"`
	Host      string     `config:"host" default:"0.0.0.0"`
	Port      uint16     `config:"port" default:"8443"`
	TlsConfig *TlsConfig `config:"tls"`
}

type WebsiteConfig struct {
	Enable      bool       `config:"enable" default:"false"`
	RootPath    string     `config:"root" default:"."`
	WebsiteName string     `config:"name" default:"web"`
	Port        uint16     `config:"port" default:"443"`
	ContentPath string     `config:"content_path" default:""`
	TlsConfig   *TlsConfig `config:"tls" `
}

type CertConfig struct {
	Cert   string `yaml:"cert"`
	CA     string `yaml:"ca"`
	Key    string `yaml:"key"`
	Enable bool   `yaml:"enable"`
}

func (t *CertConfig) ToProtobuf() *lispb.TLS {
	return &lispb.TLS{
		Cert:   t.Cert,
		Key:    t.Key,
		Enable: t.Enable,
	}
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
	CertFile string `config:"cert_file"`
	KeyFile  string `config:"key_file"`
	CAFile   string `config:"ca_file"`
}

func (t *TlsConfig) ReadCert() (*CertConfig, error) {
	if t == nil {
		return &CertConfig{Enable: false}, nil
	}
	var err error
	if t.CertFile == "" || t.KeyFile == "" || t.CAFile == "" {
		return &CertConfig{
			Cert:   "",
			Key:    "",
			CA:     "",
			Enable: t.Enable,
		}, nil
	}
	cert, err := os.ReadFile(t.CertFile)
	if err != nil {
		return nil, err
	}
	key, err := os.ReadFile(t.KeyFile)
	if err != nil {
		return nil, err
	}
	ca, err := os.ReadFile(t.CAFile)
	if err != nil {
		return nil, err
	}
	return &CertConfig{
		Cert:   string(cert),
		Key:    string(key),
		CA:     string(ca),
		Enable: t.Enable,
	}, nil
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

func GenerateTlsConfig(name string) TlsConfig {
	subject := certs.RandomSubject(name)
	return TlsConfig{
		Name: name,
		CN:   subject.CommonName,
		O:    JoinStringSlice(subject.Organization),
		C:    JoinStringSlice(subject.Country),
		L:    JoinStringSlice(subject.Locality),
		OU:   JoinStringSlice(subject.OrganizationalUnit),
		ST:   JoinStringSlice(subject.Province),
	}
}

type EncryptionConfig struct {
	Enable bool   `config:"enable"`
	Type   string `config:"type"`
	Key    string `config:"key"`
}

func (e *EncryptionConfig) NewCrypto() (cryptostream.Cryptor, error) {
	if !e.Enable {
		return cryptostream.NewCryptor(consts.CryptorRAW, nil, nil)
	}
	return cryptostream.NewCryptor(e.Type, []byte(e.Key), cryptostream.PKCS7Pad([]byte(e.Key), 16))
}

func (e *EncryptionConfig) ToProtobuf() *lispb.Encryption {
	if e == nil {
		return &lispb.Encryption{
			Enable: false,
		}
	}
	return &lispb.Encryption{
		Type:   e.Type,
		Key:    e.Key,
		Enable: e.Enable,
	}
}

// JoinStringSlice Helper function to join string slices
func JoinStringSlice(slice []string) string {
	if len(slice) > 0 {
		return slice[0] // Just return the first element for simplicity
	}
	return ""
}
