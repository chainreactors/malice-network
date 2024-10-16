package configs

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/chainreactors/files"
	"github.com/chainreactors/logs"
	crConfig "github.com/chainreactors/malice-network/helper/utils/config"
	"gopkg.in/yaml.v3"
	"io"
	insecureRand "math/rand"
	"os"
	"path"
	"time"
)

var (
	ServerConfigFileName        = "config.yaml"
	ServerRootPath              = files.GetExcPath() + ".malice"
	CurrentServerConfigFilename = "config.yaml"
	LogPath                     = path.Join(ServerRootPath, "logs")
	CertsPath                   = path.Join(ServerRootPath, "certs")
	ListenerPath                = path.Join(ServerRootPath, "listener")
	TempPath                    = path.Join(ServerRootPath, "temp")
	PluginPath                  = path.Join(ServerRootPath, "plugins")
	AuditPath                   = path.Join(ServerRootPath, "audit")
	CachePath                   = path.Join(TempPath, "cache")
	ErrNoConfig                 = errors.New("no config found")
	WebsitePath                 = path.Join(ServerRootPath, "web")
)

func NewFileLog(filename string) *logs.Logger {
	logger := logs.NewLogger(logs.Info)
	logger.SetFile(path.Join(LogPath, fmt.Sprintf("%s.log", filename)))
	logger.SetOutput(io.Discard)
	logger.Init()
	return logger
}

func NewDebugLog(filename string) *logs.Logger {
	logger := logs.NewLogger(logs.Debug)
	logger.SetFile(path.Join(LogPath, fmt.Sprintf("%s.log", filename)))
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

func getRandomID() string {
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
	err := crConfig.LoadConfig(ServerConfigFileName, &opt)
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
	Enable   bool `config:"enable" default:"true"`
	Telegram struct {
		Enable bool   `config:"enable" default:"false"`
		APIKey string `config:"api_key"`
		ChatID int64  `config:"chat_id"`
	} `config:"telegram"`
	DingTalk struct {
		Enable bool   `config:"enable" default:"false"`
		Secret string `config:"secret"`
		Token  string `config:"token"`
	} `config:"dingtalk"`
	Lark struct {
		Enable     bool   `config:"enable" default:"false"`
		WebHookUrl string `config:"webhook_url"`
	} `config:"lark"`
	ServerChan struct {
		Enable bool   `config:"enable" default:"false"`
		URL    string `config:"url"`
	} `config:"serverchan"`
}
