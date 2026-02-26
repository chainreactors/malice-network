package configs

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/utils/configutil"
	"gopkg.in/yaml.v3"
	"io"
	insecureRand "math/rand"
	"os"
	"path/filepath"
	"time"
)

var (
	ServerConfigFileName        = "config.yaml"
	ServerRootPath              = filepath.Join(GetWorkDir(), ".malice")
	CurrentServerConfigFilename = "config.yaml"
	ContextPath                 = filepath.Join(ServerRootPath, "context")
	LogPath                     = filepath.Join(ServerRootPath, "log")
	CertsPath                   = filepath.Join(ServerRootPath, "certs")
	ListenerPath                = filepath.Join(ServerRootPath, "listener")
	TempPath                    = filepath.Join(ServerRootPath, "temp")
	PluginPath                  = filepath.Join(ServerRootPath, "plugins")
	AuditPath                   = filepath.Join(ServerRootPath, "audit")
	ErrNoConfig                 = errors.New("no config found")
	WebsitePath                 = filepath.Join(ServerRootPath, "web")
	ProfilePath                 = filepath.Join(ServerRootPath, "profile")
	// variables for implant build
	BuildPath       = filepath.Join(GetWorkDir(), "..", "malefic", "build")
	BinPath         = filepath.Join(ServerRootPath, "bin")
	SourceCodePath  = filepath.Join(BuildPath, "src")
	ResourcePath    = filepath.Join(SourceCodePath, "resources")
	TargetPath      = filepath.Join(SourceCodePath, "target")
	CargoCachePath  = filepath.Join(BuildPath, "cache")
	BuildOutputPath = filepath.Join(BuildPath, "output")
)

func NewFileLog(filename string) *logs.Logger {
	logger := logs.NewLogger(logs.InfoLevel)
	logger.SetFile(filepath.Join(LogPath, fmt.Sprintf("%s.log", filename)))
	logger.SetOutput(io.Discard)
	logger.Init()
	return logger
}

func NewDebugLog(filename string) *logs.Logger {
	logger := logs.NewLogger(logs.DebugLevel)
	logger.SetFile(filepath.Join(LogPath, fmt.Sprintf("%s.log", filename)))
	logger.Init()
	return logger
}

type ServerConfig struct {
	Enable        bool          `config:"enable" default:"true" yaml:"enable"`
	GRPCPort      uint16        `config:"grpc_port" default:"5004" yaml:"grpc_port"`
	GRPCHost      string        `config:"grpc_host" default:"0.0.0.0" yaml:"grpc_host"`
	IP            string        `config:"ip" default:"" yaml:"ip"`
	DaemonConfig  bool          `config:"daemon" default:"false" yaml:"daemon"`
	EncryptionKey string        `config:"encryption_key" default:"maliceofinternal" yaml:"encryption_key"`
	LogConfig     *LogConfig    `config:"log" yaml:"log"`
	MiscConfig    *MiscConfig   `config:"config" yaml:"config"`
	NotifyConfig  *NotifyConfig `config:"notify" yaml:"notify"`
	GithubConfig  *GithubConfig `config:"github" yaml:"github"`
	SaasConfig    *SaasConfig   `config:"saas" yaml:"saas"`
}

func (c *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.IP, c.GRPCPort)
}

func (c *ServerConfig) Save() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	err = os.WriteFile(CurrentServerConfigFilename, data, 0600)
	if err != nil {
		// TODO - log failed to write config
		logs.Log.Errorf("Failed to write config %s", err)
		return err
	}
	return nil
}

func GetRandomID() string {
	seededRand := insecureRand.New(insecureRand.NewSource(time.Now().UnixNano()))
	buf := make([]byte, 32)
	seededRand.Read(buf)
	return hex.EncodeToString(buf)
}

// LogConfig - Server logging config
type LogConfig struct {
	Level int `json:"level" default:"20" config:"level" yaml:"level"`
	//GRPCUnaryPayloads  bool `json:"grpc_unary_payloads"`
	//GRPCStreamPayloads bool `json:"grpc_stream_payloads"`
	//TLSKeyLogger       bool `json:"tls_key_logger"`
}

type MiscConfig struct {
	PacketLength int    `config:"packet_length" default:"4194304" yaml:"packet_length"`
	Certificate  string `config:"cert" default:"" yaml:"cert"`
	PrivateKey   string `config:"key" default:"" yaml:"key"`
}

func LoadMiscConfig() ([]byte, []byte, error) {
	var opt ServerConfig
	// load config
	err := configutil.LoadConfig(ServerConfigFileName, &opt)
	if err != nil {
		logs.Log.Errorf("Failed to load config: %s", err)
		return nil, nil, err
	}
	if opt.MiscConfig.Certificate != "" && opt.MiscConfig.PrivateKey != "" {
		return []byte(opt.MiscConfig.Certificate), []byte(opt.MiscConfig.PrivateKey), nil
	} else {
		return nil, nil, ErrNoConfig
	}
}

type NotifyConfig struct {
	Enable     bool              `config:"enable" default:"true" yaml:"enable"`
	Telegram   *TelegramConfig   `config:"telegram" yaml:"telegram"`
	DingTalk   *DingTalkConfig   `config:"dingtalk" yaml:"dingtalk"`
	Lark       *LarkConfig       `config:"lark" yaml:"lark"`
	ServerChan *ServerChanConfig `config:"serverchan" yaml:"serverchan"`
	PushPlus   *PushPlusConfig   `config:"pushplus" yaml:"pushplus"`
}

type TelegramConfig struct {
	Enable bool   `config:"enable" default:"false" yaml:"enable"`
	APIKey string `config:"api_key" yaml:"api_key"`
	ChatID int64  `config:"chat_id" yaml:"chat_id"`
}

type DingTalkConfig struct {
	Enable bool   `config:"enable" default:"false" yaml:"enable"`
	Secret string `config:"secret" yaml:"secret"`
	Token  string `config:"token" yaml:"token"`
}

type LarkConfig struct {
	Enable     bool   `config:"enable" default:"false" yaml:"enable"`
	WebHookUrl string `config:"webhook_url" yaml:"webhook_url"`
	Secret     string `config:"secret" yaml:"secret"`
}

type ServerChanConfig struct {
	Enable bool   `config:"enable" default:"false" yaml:"enable"`
	URL    string `config:"url" yaml:"url"`
}

type PushPlusConfig struct {
	Enable  bool   `config:"enable" default:"false" yaml:"enable"`
	Token   string `config:"token" yaml:"token"`
	Topic   string `config:"topic" yaml:"topic"`
	Channel string `config:"channel" yaml:"channel"`
}

type GithubConfig struct {
	Repo     string `config:"repo" default:"malefic" yaml:"repo"`
	Owner    string `config:"owner" default:"" yaml:"owner"`
	Token    string `config:"token" default:"" yaml:"token"`
	Workflow string `config:"workflow" default:"generate.yaml" yaml:"workflow"`
}

func (g *GithubConfig) ToProtobuf() *clientpb.GithubActionBuildConfig {
	return &clientpb.GithubActionBuildConfig{
		Owner:      g.Owner,
		Repo:       g.Repo,
		Token:      g.Token,
		WorkflowId: g.Workflow,
	}
}

type SaasConfig struct {
	Enable bool   `config:"enable" yaml:"enable"`
	Url    string `config:"url" default:"" yaml:"url"`
	Token  string `config:"token" default:"" yaml:"token"`
}
