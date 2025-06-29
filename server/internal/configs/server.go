package configs

import (
	"encoding/hex"
	"errors"
	"fmt"
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
	// variables for implant build
	BuildPath       = filepath.Join(GetWorkDir(), "..", "malefic", "build")
	BinPath         = filepath.Join(ServerRootPath, "bin")
	SourceCodePath  = filepath.Join(BuildPath, "src")
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
	Enable       bool          `config:"enable" default:"true"`
	GRPCPort     uint16        `config:"grpc_port" default:"5004"`
	GRPCHost     string        `config:"grpc_host" default:"0.0.0.0"`
	IP           string        `config:"ip" default:""`
	DaemonConfig bool          `config:"daemon" default:"false"`
	LogConfig    *LogConfig    `config:"log"`
	MiscConfig   *MiscConfig   `config:"config"`
	NotifyConfig *NotifyConfig `config:"notify"`
	GithubConfig *GithubConfig `config:"github"`
	SassConfig   *SaasConfig   `config:"saas"`
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
	Level int `json:"level" default:"20" config:"level"`
	//GRPCUnaryPayloads  bool `json:"grpc_unary_payloads"`
	//GRPCStreamPayloads bool `json:"grpc_stream_payloads"`
	//TLSKeyLogger       bool `json:"tls_key_logger"`
}

type MiscConfig struct {
	PacketLength int    `config:"packet_length" default:"4194304"`
	Certificate  string `config:"cert" default:""`
	PrivateKey   string `config:"key" default:""`
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
	Enable     bool              `config:"enable" default:"true"`
	Telegram   *TelegramConfig   `config:"telegram"`
	DingTalk   *DingTalkConfig   `config:"dingtalk"`
	Lark       *LarkConfig       `config:"lark"`
	ServerChan *ServerChanConfig `config:"serverchan"`
}

type TelegramConfig struct {
	Enable bool   `config:"enable" default:"false"`
	APIKey string `config:"api_key"`
	ChatID int64  `config:"chat_id"`
}

type DingTalkConfig struct {
	Enable bool   `config:"enable" default:"false"`
	Secret string `config:"secret"`
	Token  string `config:"token"`
}

type LarkConfig struct {
	Enable     bool   `config:"enable" default:"false"`
	WebHookUrl string `config:"webhook_url"`
}

type ServerChanConfig struct {
	Enable bool   `config:"enable" default:"false"`
	URL    string `config:"url"`
}

type GithubConfig struct {
	Repo     string `config:"repo" default:"malefic"`
	Owner    string `config:"owner" default:""`
	Token    string `config:"token" default:""`
	Workflow string `config:"workflow" default:"generate.yml"`
}

type SaasConfig struct {
	Enable bool   `config:"enable" default:"true"`
	Host   string `config:"host" default:""`
	Port   uint16 `config:"port" default:""`
	Token  string `config:"token" default:""`
}
