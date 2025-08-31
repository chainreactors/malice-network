package mtls

import (
	"encoding/hex"
	"fmt"
	"github.com/chainreactors/logs"
	"gopkg.in/yaml.v3"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

var (
	Listener = "listener"
	Client   = "client"
)

type ClientConfig struct {
	Operator      string `json:"operator" yaml:"operator"` // This value is actually ignored for the most part (cert CN is used instead)
	Host          string `json:"host" yaml:"host"`
	Port          int    `json:"port" yaml:"port"`
	Type          string `json:"type" yaml:"type"`
	CACertificate string `json:"ca" yaml:"ca"`
	PrivateKey    string `json:"key" yaml:"key"`
	Certificate   string `json:"cert" yaml:"cert"`
}

func (c *ClientConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

//func GetConfigs() map[string]*ClientConfig {
//	configDir := GetConfigDir()
//	configFiles, err := ioutil.ReadDir(configDir)
//	if err != nil {
//		log.Printf("No configs found %v", err)
//		return map[string]*ClientConfig{}
//	}
//
//	confs := map[string]*ClientConfig{}
//	for _, confFile := range configFiles {
//		confFilePath := path.Join(configDir, confFile.Name())
//		log.Printf("Parsing config %s", confFilePath)
//
//		conf, err := ReadConfig(confFilePath)
//		if err != nil {
//			continue
//		}
//		digest := sha256.Sum256([]byte(conf.Certificate))
//		confs[fmt.Sprintf("%s@%s (%x)", conf.Operator, conf.Host, digest[:8])] = conf
//	}
//	return confs
//}

// ReadConfig - Load config into struct
func ReadConfig(confFilePath string) (*ClientConfig, error) {
	confFile, err := os.Open(confFilePath)
	if err != nil {
		return nil, err
	}
	defer confFile.Close()
	data, err := io.ReadAll(confFile)
	if err != nil {
		return nil, err
	}
	conf := &ClientConfig{}
	err = yaml.Unmarshal(data, conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

// NewClientConfig - new config and save in local file
//func NewClientConfig(host, user string, port int, caType string, certs, privateKey, ca []byte) *ClientConfig {
//	// new config
//	config := &ClientConfig{
//		Operator:      user,
//		Host:         host,
//		Port:         port,
//		Type:          caType,
//		CACertificate: string(ca),
//		PrivateKey:    string(privateKey),
//		Certificate:   string(certs),
//	}
//
//	return config
//}

// save config as yaml file
//func WriteConfig(clientConfig *ClientConfig, clientType, name string) error {
//	configDir, _ := os.Getwd()
//	var configFile string
//	if clientType == listener {
//		configFile = path.Join(configDir, fmt.Sprintf("%s.yaml", name))
//	} else {
//		configFile = path.Join(configDir, fmt.Sprintf("%s_%s.yaml", name, clientConfig.Host))
//	}
//	data, err := yaml.Marshal(clientConfig)
//	if err != nil {
//		return err
//	}
//	err = os.WriteFile(configFile, data, 0644)
//	if err != nil {
//		logs.Log.Errorf("write config to file failed: %v", err)
//		return err
//	}
//	return nil
//}

// generateOperatorToken - Generate a new operator auth token
func generateOperatorToken() string {
	buf := make([]byte, 32)
	n, err := rand.Read(buf)
	if err != nil || n != len(buf) {
		logs.Log.Error("Failed to generate random token")
	}
	return hex.EncodeToString(buf)
}

func GetListeners() ([]string, error) {
	var files []string
	configPath, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(configPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			baseName := filepath.Base(path)
			if strings.HasSuffix(baseName, ".yaml") {
				if !strings.HasPrefix(baseName, "config.yaml") {
					fileName := strings.TrimSuffix(baseName, filepath.Ext(baseName))
					files = append(files, fileName)
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return files, nil
}
