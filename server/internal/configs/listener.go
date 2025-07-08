package configs

import (
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"os"

	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	cryptostream "github.com/chainreactors/malice-network/server/internal/stream"
	"golang.org/x/exp/slices"
)

var ListenerConfigFileName = "listener.yaml"

type ListenerConfig struct {
	Enable bool   `config:"enable" default:"true"`
	Name   string `config:"name" default:"listener"`
	Auth   string `config:"auth" default:"listener.auth"`
	//Server             string                `config:"server" default:"127.0.0.1"`
	IP                 string                `config:"ip"`
	TcpPipelines       []*TcpPipelineConfig  `config:"tcp" `
	BindPipelineConfig []*BindPipelineConfig `config:"bind"`
	HttpPipelines      []*HttpPipelineConfig `config:"http"`
	Websites           []*WebsiteConfig      `config:"website"`
	REMs               []*REMConfig          `config:"rem"`
	AutoBuildConfig    *AutoBuildConfig      `config:"auto_build"`
}

type TcpPipelineConfig struct {
	Enable           bool                    `config:"enable" default:"true"`
	Name             string                  `config:"name" default:"tcp"`
	Host             string                  `config:"host" default:"0.0.0.0"`
	Port             uint16                  `config:"port" default:"5001"`
	Parser           string                  `config:"parser" default:"malefic"`
	TlsConfig        *TlsConfig              `config:"tls"`
	EncryptionConfig types.EncryptionsConfig `config:"encryption"`
}

type AutoBuildConfig struct {
	Enable     bool     `config:"enable" default:"false"`
	BuildPulse bool     `config:"build_pulse" default:"false"`
	Target     []string `config:"target" default:""`
	Pipeline   []string `config:"pipeline" default:""`
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
		Type:       consts.TCPPipeline,
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
	Enable           bool                    `config:"enable" default:"true"`
	Name             string                  `config:"name" default:"bind"`
	TlsConfig        *TlsConfig              `config:"tls"`
	EncryptionConfig types.EncryptionsConfig `config:"encryption"`
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
	Enable           bool                    `config:"enable" default:"true"`
	Name             string                  `config:"name" default:"http"`
	Host             string                  `config:"host" default:"0.0.0.0"`
	Port             uint16                  `config:"port" default:"8080"`
	Parser           string                  `config:"parser" default:"malefic"`
	TlsConfig        *TlsConfig              `config:"tls"`
	EncryptionConfig types.EncryptionsConfig `config:"encryption"`
	Headers          map[string][]string     `config:"headers"`
	ErrorPage        string                  `config:"error_page"`
	BodyPrefix       string                  `config:"body_prefix"`
	BodySuffix       string                  `config:"body_suffix"`
}

func (http *HttpPipelineConfig) ToProtobuf(lisId string) (*clientpb.Pipeline, error) {
	tls, err := http.TlsConfig.ReadCert()
	if err != nil {
		return nil, err
	}
	// 如果指定了错误页面，读取文件内容
	var errorPageContent string
	if http.ErrorPage != "" {
		content, err := os.ReadFile(http.ErrorPage)
		if err != nil {
			return nil, fmt.Errorf("failed to read error page file: %v", err)
		}
		errorPageContent = string(content)
	}

	// 序列化额外参数
	params := types.PipelineParams{
		Headers:    http.Headers,
		ErrorPage:  errorPageContent,
		BodyPrefix: http.BodyPrefix,
		BodySuffix: http.BodySuffix,
	}
	paramsJson, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pipeline params: %v", err)
	}

	return &clientpb.Pipeline{
		Name:       http.Name,
		ListenerId: lisId,
		Enable:     http.Enable,
		Parser:     http.Parser,
		Type:       consts.HTTPPipeline,
		Body: &clientpb.Pipeline_Http{
			Http: &clientpb.HTTPPipeline{
				Host:   http.Host,
				Port:   uint32(http.Port),
				Params: string(paramsJson),
			},
		},
		Tls:        tls.ToProtobuf(),
		Encryption: http.EncryptionConfig.ToProtobuf(),
	}, nil
}

type REMConfig struct {
	Enable  bool   `config:"enable" default:"false"`
	Name    string `config:"name" default:"default-rem"`
	Console string `config:"console" default:""`
}

func (r *REMConfig) ToProtobuf(lisId string) (*clientpb.Pipeline, error) {
	return &clientpb.Pipeline{
		Name:       r.Name,
		Enable:     r.Enable,
		Type:       consts.RemPipeline,
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
	File string `config:"file"`
	Path string `config:"path"`
	Type string `config:"type" default:"raw"`
	//EncryptionConfig *EncryptionConfig `config:"encryption"`
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
		File:    content.File,
		Path:    content.Path,
		Size:    uint64(len(data)),
		Type:    content.Type,
		Content: data,
		//Encryption: content.EncryptionConfig.ToProtobuf(),
	}, nil
}

type CertConfig struct {
	*types.CertConfig
	Ca     *types.CertConfig
	Enable bool `yaml:"enable"`
}

func (t *CertConfig) ToProtobuf() *clientpb.TLS {
	if t.CertConfig == nil {
		return &clientpb.TLS{
			Enable: t.Enable,
		}
	}
	var ca *clientpb.Cert
	if t.Ca != nil {
		ca = &clientpb.Cert{
			Cert: t.Ca.Cert,
		}
	}
	return &clientpb.TLS{
		Cert: &clientpb.Cert{
			Cert: t.Cert,
			Key:  t.Key,
		},
		Ca:     ca,
		Enable: t.Enable,
		Acme:   t.Enable,
	}
}

type TlsConfig struct {
	Enable   bool   `config:"enable"`
	CertFile string `config:"cert_file"`
	KeyFile  string `config:"key_file"`
	CAFile   string `config:"ca_file"`
	//Acme     bool   `config:"acme"`
	//Domain   string `config:"domain"`
	Name     string `config:"name"`
	CN       string `config:"CN"`
	O        string `config:"O"`
	C        string `config:"C"`
	L        string `config:"L"`
	OU       string `config:"OU"`
	ST       string `config:"ST"`
	Validity string `config:"validity"`
}

func (t *TlsConfig) ToPkix() *pkix.Name {
	if t.Name == "" && t.CN == "" && t.O == "" && t.C == "" && t.L == "" && t.OU == "" && t.ST == "" {
		return nil
	}
	return &pkix.Name{
		CommonName:         t.Name,
		Organization:       []string{t.O},
		Country:            []string{t.C},
		Locality:           []string{t.L},
		OrganizationalUnit: []string{t.OU},
		Province:           []string{t.ST},
	}
}

func (t *TlsConfig) ReadCert() (*types.TlsConfig, error) {
	if t == nil {
		return &types.TlsConfig{
			Enable: false,
		}, nil
	}
	if t.CertFile != "" && t.KeyFile != "" && t.CAFile != "" {
		cert, err := os.ReadFile(t.CertFile)
		if err != nil {
			return nil, err
		}
		key, err := os.ReadFile(t.KeyFile)
		if err != nil {
			return nil, err
		}
		caCert, err := os.ReadFile(t.CAFile)
		if err != nil {
			return nil, err
		}
		return &types.TlsConfig{
			Cert: &types.CertConfig{
				Cert: string(cert),
				Key:  string(key),
			},
			CA: &types.CertConfig{
				Cert: string(caCert),
			},
			Enable: t.Enable,
			//Acme:   t.Acme,
			//Domain: t.Domain,
		}, nil
	}
	return &types.TlsConfig{
		Enable: t.Enable,
		//Acme:    t.Acme,
		//Domain:  t.Domain,
		Subject: t.ToPkix(),
	}, nil
}

func NewCrypto(es []*clientpb.Encryption) ([]cryptostream.Cryptor, error) {
	var cryptos []cryptostream.Cryptor
	for _, e := range es {
		iv := slices.Clone([]byte(e.Key))
		slices.Reverse(iv)
		c, err := cryptostream.NewCryptor(e.Type, []byte(e.Key), cryptostream.PKCS7Pad(iv, 16))
		if err != nil {
			return nil, err
		}
		cryptos = append(cryptos, c)
	}

	return cryptos, nil
}

// JoinStringSlice Helper function to join string slices
func JoinStringSlice(slice []string) string {
	if len(slice) > 0 {
		return slice[0] // Just return the first element for simplicity
	}
	return ""
}
