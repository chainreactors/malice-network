package configs

import (
	"crypto/x509/pkix"
	"fmt"
	"os"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/implanttypes"

	cryptostream "github.com/chainreactors/malice-network/server/internal/stream"
	"golang.org/x/exp/slices"
)

var ListenerConfigFileName = "listener.yaml"

type ListenerConfig struct {
	Enable bool   `config:"enable" default:"true" yaml:"enable"`
	Name   string `config:"name" default:"listener" yaml:"name"`
	Auth   string `config:"auth" default:"listener.auth" yaml:"auth"`
	//Server             string                `config:"server" default:"127.0.0.1"`
	IP                 string                `config:"ip" yaml:"ip"`
	TcpPipelines       []*TcpPipelineConfig  `config:"tcp" yaml:"tcp"`
	BindPipelineConfig []*BindPipelineConfig `config:"bind" yaml:"bind"`
	HttpPipelines      []*HttpPipelineConfig `config:"http" yaml:"http"`
	Websites           []*WebsiteConfig      `config:"website" yaml:"website"`
	REMs               []*REMConfig          `config:"rem" yaml:"rem"`
	AutoBuildConfig    *AutoBuildConfig      `config:"auto_build" yaml:"auto_build"`
}

type TcpPipelineConfig struct {
	Enable           bool                           `config:"enable" default:"true" yaml:"enable"`
	Name             string                         `config:"name" default:"tcp" yaml:"name"`
	Host             string                         `config:"host" default:"0.0.0.0" yaml:"host"`
	Port             uint16                         `config:"port" default:"5001" yaml:"port"`
	Parser           string                         `config:"parser" default:"malefic" yaml:"parser"`
	TlsConfig        *TlsConfig                     `config:"tls" yaml:"tls"`
	EncryptionConfig implanttypes.EncryptionsConfig `config:"encryption" yaml:"encryption"`
	SecureConfig     *implanttypes.SecureConfig     `config:"secure" yaml:"secure"` // Age 密码学安全配置
	PacketLength     int                            `config:"packet_length" yaml:"packet_length"`
}

type AutoBuildConfig struct {
	Enable     bool     `config:"enable" default:"false" yaml:"enable"`
	BuildPulse bool     `config:"build_pulse" default:"false" yaml:"build_pulse"`
	Target     []string `config:"target" default:"" yaml:"target"`
	Pipeline   []string `config:"pipeline" default:"" yaml:"pipeline"`
}

func (tcp *TcpPipelineConfig) ToProtobuf(lisId string) (*clientpb.Pipeline, error) {
	tls, err := tlsToProtobuf(tcp.TlsConfig)
	if err != nil {
		return nil, err
	}
	return &clientpb.Pipeline{
		Name:       tcp.Name,
		ListenerId: lisId,
		Enable:     tcp.Enable,
		Parser:     tcp.Parser,
		Type:       consts.TCPPipeline,
		PacketLength: uint32(tcp.PacketLength),
		Body: &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Host: tcp.Host,
				Port: uint32(tcp.Port),
			},
		},
		Tls:        tls,
		Encryption: tcp.EncryptionConfig.ToProtobuf(),
		Secure:     tcp.SecureConfig.ToProtobuf(),
	}, nil
}

type BindPipelineConfig struct {
	Enable           bool                           `config:"enable" default:"true" yaml:"enable"`
	Name             string                         `config:"name" default:"bind" yaml:"name"`
	TlsConfig        *TlsConfig                     `config:"tls" yaml:"tls"`
	EncryptionConfig implanttypes.EncryptionsConfig `config:"encryption" yaml:"encryption"`
	PacketLength     int                            `config:"packet_length" yaml:"packet_length"`
}

func (pipeline *BindPipelineConfig) ToProtobuf(lisId string) (*clientpb.Pipeline, error) {
	tls, err := tlsToProtobuf(pipeline.TlsConfig)
	if err != nil {
		return nil, err
	}
	return &clientpb.Pipeline{
		Name:       pipeline.Name,
		Enable:     pipeline.Enable,
		ListenerId: lisId,
		Parser:     consts.ImplantMalefic,
		PacketLength: uint32(pipeline.PacketLength),
		Body: &clientpb.Pipeline_Bind{
			Bind: &clientpb.BindPipeline{},
		},
		Tls:        tls,
		Encryption: pipeline.EncryptionConfig.ToProtobuf(),
	}, nil
}

type HttpPipelineConfig struct {
	Enable           bool                           `config:"enable" default:"true" yaml:"enable"`
	Name             string                         `config:"name" default:"http" yaml:"name"`
	Host             string                         `config:"host" default:"0.0.0.0" yaml:"host"`
	Port             uint16                         `config:"port" default:"8080" yaml:"port"`
	Parser           string                         `config:"parser" default:"malefic" yaml:"parser"`
	TlsConfig        *TlsConfig                     `config:"tls" yaml:"tls"`
	EncryptionConfig implanttypes.EncryptionsConfig `config:"encryption" yaml:"encryption"`
	SecureConfig     *implanttypes.SecureConfig     `config:"secure" yaml:"secure"` // Age 密码学安全配置
	Headers          map[string][]string            `config:"headers" yaml:"headers"`
	ErrorPage        string                         `config:"error_page" yaml:"error_page"`
	BodyPrefix       string                         `config:"body_prefix" yaml:"body_prefix"`
	BodySuffix       string                         `config:"body_suffix" yaml:"body_suffix"`
	PacketLength     int                            `config:"packet_length" yaml:"packet_length"`
}

func (http *HttpPipelineConfig) ToProtobuf(lisId string) (*clientpb.Pipeline, error) {
	tls, err := tlsToProtobuf(http.TlsConfig)
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
	params := &implanttypes.PipelineParams{
		Headers:    http.Headers,
		ErrorPage:  errorPageContent,
		BodyPrefix: http.BodyPrefix,
		BodySuffix: http.BodySuffix,
	}

	return &clientpb.Pipeline{
		Name:       http.Name,
		ListenerId: lisId,
		Enable:     http.Enable,
		Parser:     http.Parser,
		Type:       consts.HTTPPipeline,
		PacketLength: uint32(http.PacketLength),
		Body: &clientpb.Pipeline_Http{
			Http: &clientpb.HTTPPipeline{
				Host:   http.Host,
				Port:   uint32(http.Port),
				Params: params.String(),
			},
		},
		Tls:        tls,
		Encryption: http.EncryptionConfig.ToProtobuf(),
		Secure:     http.SecureConfig.ToProtobuf(),
	}, nil
}

type REMConfig struct {
	Enable  bool   `config:"enable" default:"false" yaml:"enable"`
	Name    string `config:"name" default:"default-rem" yaml:"name"`
	Console string `config:"console" default:"" yaml:"console"`
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
	Enable      bool          `config:"enable" default:"false" yaml:"enable"`
	RootPath    string        `config:"root" default:"." yaml:"root"`
	WebsiteName string        `config:"name" default:"web" yaml:"name"`
	Port        uint16        `config:"port" default:"443" yaml:"port"`
	Auth        string        `config:"auth" default:"" yaml:"auth"` // website-level default auth "user:pass"
	WebContents []*WebContent `config:"content" default:"" yaml:"content"`
	TlsConfig   *TlsConfig    `config:"tls" yaml:"tls"`
}

type WebContent struct {
	File string `config:"file" yaml:"file"`
	Path string `config:"path" yaml:"path"`
	Type string `config:"type" default:"raw" yaml:"type"`
	Auth string `config:"auth" default:"" yaml:"auth"` // per-path auth "user:pass", "none" = skip
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

type TlsConfig struct {
	Enable   bool   `config:"enable" yaml:"enable"`
	MTLS     bool   `config:"mtls" yaml:"mtls"`
	CertFile string `config:"cert_file" yaml:"cert_file"`
	KeyFile  string `config:"key_file" yaml:"key_file"`
	CAFile   string `config:"ca_file" yaml:"ca_file"`
	//Acme     bool   `config:"acme"`
	//Domain   string `config:"domain"`
	//Name     string `config:"name"`
	CN string `config:"CN" yaml:"CN"`
	O  string `config:"O" yaml:"O"`
	C  string `config:"C" yaml:"C"`
	L  string `config:"L" yaml:"L"`
	OU string `config:"OU" yaml:"OU"`
	ST string `config:"ST" yaml:"ST"`
	//Validity string `config:"validity"`
}

func (t *TlsConfig) ToPkix() *pkix.Name {
	if t.CN == "" && t.O == "" && t.C == "" && t.L == "" && t.OU == "" && t.ST == "" {
		return nil
	}
	return &pkix.Name{
		CommonName:         t.CN,
		Organization:       []string{t.O},
		Country:            []string{t.C},
		Locality:           []string{t.L},
		OrganizationalUnit: []string{t.OU},
		Province:           []string{t.ST},
	}
}

func (t *TlsConfig) ReadCert() (*implanttypes.TlsConfig, error) {
	// 处理nil情况
	if t == nil {
		return &implanttypes.TlsConfig{Enable: false}, nil
	}
	// 创建基础TLS配置
	tls := &implanttypes.TlsConfig{
		Enable:  t.Enable,
		MTLS:    t.MTLS,
		Subject: t.ToPkix(),
	}
	// 如果没有证书文件，直接返回基础配置
	if t.CertFile == "" || t.KeyFile == "" {
		return tls, nil
	}
	// 读取证书文件
	cert, err := os.ReadFile(t.CertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cert file: %s", err)
	}
	key, err := os.ReadFile(t.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %s", err)
	}
	// 设置证书配置
	tls.Cert = &implanttypes.CertConfig{
		Cert: string(cert),
		Key:  string(key),
	}
	// 读取CA证书（如果存在）
	if t.CAFile != "" {
		caCert, err := os.ReadFile(t.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %s", err)
		}
		tls.CA = &implanttypes.CertConfig{
			Cert: string(caCert),
		}
	}
	return tls, nil
}

func tlsToProtobuf(config *TlsConfig) (*clientpb.TLS, error) {
	tls, err := config.ReadCert()
	if err != nil {
		return nil, err
	}
	return tls.ToProtobuf(), nil
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
