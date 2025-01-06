package configs

import (
	"crypto/x509/pkix"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/stream"
	"os"
	"slices"
)

var ListenerConfigFileName = "listener.yaml"

type ListenerConfig struct {
	Enable             bool                  `config:"enable" default:"true"`
	Name               string                `config:"name" default:"listener"`
	Auth               string                `config:"auth" default:"listener.auth"`
	TcpPipelines       []*TcpPipelineConfig  `config:"tcp" `
	BindPipelineConfig []*BindPipelineConfig `config:"bind"`
	//HttpPipelines []*HttpPipelineConfig `config:"http" default:""`
	Websites []*WebsiteConfig `config:"website"`
	REMs     []*REMConfig     `config:"rem"`
}

type TcpPipelineConfig struct {
	Enable           bool              `config:"enable" default:"true"`
	Name             string            `config:"name" default:"tcp"`
	Host             string            `config:"host" default:"0.0.0.0"`
	Port             uint16            `config:"port" default:"5001"`
	Parser           string            `config:"parser" default:"malefic"`
	TlsConfig        *TlsConfig        `config:"tls"`
	EncryptionConfig *EncryptionConfig `config:"encryption"`
}

func (tcp *TcpPipelineConfig) ToProtobuf(lisId string) (*clientpb.Pipeline, error) {
	tls, err := tcp.TlsConfig.ReadCert()
	if err != nil {
		return nil, err
	}

	return &clientpb.Pipeline{
		Name:       tcp.Name,
		ListenerId: lisId,
		Enable:     tcp.Enable,
		Parser:     tcp.Parser,
		Body: &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Host: tcp.Host,
				Port: uint32(tcp.Port),
			},
		},
		Tls:        tls.ToProtobuf(),
		Encryption: tcp.EncryptionConfig.ToProtobuf(),
	}, nil
}

type BindPipelineConfig struct {
	Enable           bool              `config:"enable" default:"true"`
	Name             string            `config:"name" default:"bind"`
	TlsConfig        *TlsConfig        `config:"tls"`
	EncryptionConfig *EncryptionConfig `config:"encryption"`
}

func (pipeline *BindPipelineConfig) ToProtobuf(lisId string) (*clientpb.Pipeline, error) {
	tls, err := pipeline.TlsConfig.ReadCert()
	if err != nil {
		return nil, err
	}
	return &clientpb.Pipeline{
		Name:       pipeline.Name,
		Enable:     pipeline.Enable,
		ListenerId: lisId,
		Parser:     consts.ImplantMalefic,
		Body: &clientpb.Pipeline_Bind{
			Bind: &clientpb.BindPipeline{},
		},
		Tls:        tls.ToProtobuf(),
		Encryption: pipeline.EncryptionConfig.ToProtobuf(),
	}, nil
}

type HttpPipelineConfig struct {
	Enable    bool       `config:"enable" default:"false"`
	Name      string     `config:"name" default:"default-http"`
	Host      string     `config:"host" default:"0.0.0.0"`
	Port      uint16     `config:"port" default:"8443"`
	TlsConfig *TlsConfig `config:"tls"`
}

type REMConfig struct {
	Enable  bool   `config:"enable" default:"false"`
	Name    string `config:"name" default:"default-rem"`
	Console string `config:"console" default:"tcp://0.0.0.0"`
}

func (r *REMConfig) ToProtobuf(lisId string) (*clientpb.Pipeline, error) {
	return &clientpb.Pipeline{
		Name:       r.Name,
		Enable:     r.Enable,
		ListenerId: lisId,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Console: r.Console,
			},
		},
	}, nil
}

type WebsiteConfig struct {
	Enable      bool          `config:"enable" default:"false"`
	RootPath    string        `config:"root" default:"."`
	WebsiteName string        `config:"name" default:"web"`
	Port        uint16        `config:"port" default:"443"`
	WebContents []*WebContent `config:"content" default:""`
	TlsConfig   *TlsConfig    `config:"tls" `
}

type WebContent struct {
	File             string            `config:"file"`
	Path             string            `config:"path"`
	Type             string            `config:"type" default:"raw"`
	EncryptionConfig *EncryptionConfig `config:"encryption"`
}

func (content *WebContent) ToProtobuf() (*clientpb.WebContent, error) {
	var data []byte
	var err error
	if content.Type == "raw" {
		data, err = os.ReadFile(content.File)
		if err != nil {
			return nil, err
		}
	}

	return &clientpb.WebContent{
		File:       content.File,
		Path:       content.Path,
		Size:       uint64(len(data)),
		Type:       content.Type,
		Content:    data,
		Encryption: content.EncryptionConfig.ToProtobuf(),
	}, nil
}

type CertConfig struct {
	Cert   string `yaml:"cert"`
	CA     string `yaml:"ca"`
	Key    string `yaml:"key"`
	Enable bool   `yaml:"enable"`
}

func (t *CertConfig) ToProtobuf() *clientpb.TLS {
	return &clientpb.TLS{
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

func NewCrypto(e *clientpb.Encryption) (cryptostream.Cryptor, error) {
	if !e.Enable {
		return cryptostream.NewCryptor(consts.CryptorRAW, nil, nil)
	}
	iv := slices.Clone([]byte(e.Key))
	slices.Reverse(iv)
	return cryptostream.NewCryptor(e.Type, []byte(e.Key), cryptostream.PKCS7Pad(iv, 16))
}

func (e *EncryptionConfig) ToProtobuf() *clientpb.Encryption {
	if e == nil {
		return &clientpb.Encryption{
			Enable: false,
		}
	}
	return &clientpb.Encryption{
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
