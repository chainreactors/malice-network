package configs

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/chainreactors/files"
	"github.com/chainreactors/logs"
	"github.com/goccy/go-yaml"
	"github.com/gookit/config/v2"
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
	TempPath                    = path.Join(ServerRootPath, "temp")
	PluginPath                  = path.Join(ServerRootPath, "plugins")
	AuditPath                   = path.Join(ServerRootPath, "audit")
	CachePath                   = path.Join(TempPath, "cache")
	ErrNoConfig                 = errors.New("no config found")
	WebsitePath                 = path.Join(ServerRootPath, "web")
)

func InitConfig() error {
	perm := os.FileMode(0o700)
	err := os.MkdirAll(ServerRootPath, perm)
	if err != nil {
		return err
	}
	os.MkdirAll(LogPath, perm)
	os.MkdirAll(CertsPath, perm)
	os.MkdirAll(TempPath, perm)
	//os.MkdirAll(PluginPath, perm)
	os.MkdirAll(AuditPath, perm)
	os.MkdirAll(CachePath, perm)
	os.MkdirAll(WebsitePath, perm)
	return nil
}

func GetServerConfig() *ServerConfig {
	s := &ServerConfig{}
	err := config.MapStruct("server", s)
	if err != nil {
		logs.Log.Errorf("Failed to map server config %s", err)
		return nil
	}
	return s
}

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

func GetConfig(key string) any {
	return config.Get("server.config." + key)
}

func LoadConfig(filename string, v interface{}) error {
	err := config.LoadFiles(filename)
	if err != nil {
		return err
	}
	err = config.Decode(v)
	if err != nil {
		return err
	}
	return nil
}

type ServerConfig struct {
	GRPCPort     uint16        `config:"grpc_port" default:"5004"`
	GRPCHost     string        `config:"grpc_host" default:"0.0.0.0"`
	DaemonConfig *DaemonConfig `config:"daemon"`
	LogConfig    *LogConfig    `config:"log" default:""`
	MiscConfig   *MiscConfig   `config:"config" default:""`
}

func (c *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.GRPCHost, c.GRPCPort)
}

func (c *ServerConfig) Save() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	err = os.WriteFile(CurrentServerConfigFilename, data, 0o600)
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
	Level              int  `json:"level" default:"20"`
	GRPCUnaryPayloads  bool `json:"grpc_unary_payloads"`
	GRPCStreamPayloads bool `json:"grpc_stream_payloads"`
	TLSKeyLogger       bool `json:"tls_key_logger"`
}

// DaemonConfig - Configure daemon mode
type DaemonConfig struct {
	Host string `json:"host" default:"0.0.0.0"`
	Port int    `json:"port" default:"5001"`
}

type MiscConfig struct {
	PacketLength int    `config:"packet_length" default:"4194304"`
	Certificate  string `config:"certificate" default:""`
	PrivateKey   string `config:"certificate_key" default:""`
}

func LoadMiscConfig() ([]byte, []byte, error) {
	var opt ServerConfig
	// load config
	err := LoadConfig(ServerConfigFileName, &opt)
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
