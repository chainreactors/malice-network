package configs

import (
	"encoding/hex"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/goccy/go-yaml"
	"github.com/gookit/config/v2"
	insecureRand "math/rand"
	"os"
	"path/filepath"
	"time"
)

var (
	ServerConfigFileName        = "config.yaml"
	ServerRootPath              = ".malice"
	CurrentServerConfigFilename = "config.yaml"
)

func GetServerConfig() *ServerConfig {
	s := &ServerConfig{}
	err := config.MapStruct("server", s)
	if err != nil {
		logs.Log.Errorf("Failed to map server config %s", err)
		return nil
	}
	return s
}

func GetLogPath() string {
	dir := filepath.Join(ServerRootPath, "logs")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			logs.Log.Errorf("Failed to create logs dir %s", err)
		}
	}
	return dir
}

func SetLogFilePath(stream string, level logs.Level) *logs.Logger {
	logger := logs.NewLogger(level)
	logger.SetFile(filepath.Join(GetLogPath(), fmt.Sprintf("%s.log", stream)))
	return logger
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
}

func (c *ServerConfig) String() string {
	return fmt.Sprintf("%s:%d", c.GRPCHost, c.GRPCPort)
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

// // AddMultiplayerJob - Add Job Configs
//
//	func (c *ServerConfig) AddMultiplayerJob(config *MultiplayerJobConfig) error {
//		if c.Jobs == nil {
//			c.Jobs = &JobConfig{}
//		}
//		config.JobID = getRandomID()
//		c.Jobs.Multiplayer = append(c.Jobs.Multiplayer, config)
//		return c.Save()
//	}
//
// // AddMTLSJob - Add Job Configs
//
//	func (c *ServerConfig) AddMTLSJob(config *MTLSJobConfig) error {
//		if c.Jobs == nil {
//			c.Jobs = &JobConfig{}
//		}
//		config.JobID = getRandomID()
//		c.Jobs.MTLS = append(c.Jobs.MTLS, config)
//		return c.Save()
//	}
//
// // AddWGJob - Add Job Configs
//
//	func (c *ServerConfig) AddWGJob(config *WGJobConfig) error {
//		if c.Jobs == nil {
//			c.Jobs = &JobConfig{}
//		}
//		config.JobID = getRandomID()
//		c.Jobs.WG = append(c.Jobs.WG, config)
//		return c.Save()
//	}
//
// // AddDNSJob - Add a persistent DNS job
//
//	func (c *ServerConfig) AddDNSJob(config *DNSJobConfig) error {
//		if c.Jobs == nil {
//			c.Jobs = &JobConfig{}
//		}
//		config.JobID = getRandomID()
//		c.Jobs.DNS = append(c.Jobs.DNS, config)
//		return c.Save()
//	}
//
// // AddHTTPJob - Add a persistent job
//
//	func (c *ServerConfig) AddHTTPJob(config *HTTPJobConfig) error {
//		if c.Jobs == nil {
//			c.Jobs = &JobConfig{}
//		}
//		config.JobID = getRandomID()
//		c.Jobs.HTTP = append(c.Jobs.HTTP, config)
//		return c.Save()
//	}
//
// // RemoveJob - Remove Job by ID
//
//	func (c *ServerConfig) RemoveJob(jobID string) {
//		if c.Jobs == nil {
//			return
//		}
//		defer c.Save()
//		for i, j := range c.Jobs.MTLS {
//			if j.JobID == jobID {
//				c.Jobs.MTLS = append(c.Jobs.MTLS[:i], c.Jobs.MTLS[i+1:]...)
//				return
//			}
//		}
//		for i, j := range c.Jobs.DNS {
//			if j.JobID == jobID {
//				c.Jobs.DNS = append(c.Jobs.DNS[:i], c.Jobs.DNS[i+1:]...)
//				return
//			}
//		}
//		for i, j := range c.Jobs.HTTP {
//			if j.JobID == jobID {
//				c.Jobs.HTTP = append(c.Jobs.HTTP[:i], c.Jobs.HTTP[i+1:]...)
//				return
//			}
//		}
//	}
//

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
