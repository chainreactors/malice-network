package mtls

import (
	"encoding/hex"
	"fmt"
	"github.com/chainreactors/logs"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"
)

var (
	MaliceDirName = ".config/malice"
	ConfigDirName = "configs"
	host          = "localhost"
)

type ClientConfig struct {
	Operator      string `json:"operator"` // This value is actually ignored for the most part (cert CN is used instead)
	LHost         string `json:"lhost"`
	LPort         int    `json:"lport"`
	Token         string `json:"token"`
	CACertificate string `json:"ca_certificate"`
	PrivateKey    string `json:"private_key"`
	Certificate   string `json:"certificate"`
}

func GetConfigDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	dir := filepath.Join(rootDir, ConfigDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}
	return dir
}

func GetRootAppDir() string {
	user, _ := user.Current()
	dir := filepath.Join(user.HomeDir, MaliceDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			logs.Log.Error(err.Error())
		}
	}
	return dir
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
//		confs[fmt.Sprintf("%s@%s (%x)", conf.Operator, conf.LHost, digest[:8])] = conf
//	}
//	return confs
//}

// ReadConfig - Load config into struct
func ReadConfig(confFilePath string) (*ClientConfig, error) {
	confFile, err := os.Open(confFilePath)
	if err != nil {
		log.Printf("Open failed %v", err)
		return nil, err
	}
	defer confFile.Close()
	data, err := ioutil.ReadAll(confFile)
	if err != nil {
		log.Printf("Read failed %v", err)
		return nil, err
	}
	conf := &ClientConfig{}
	err = yaml.Unmarshal(data, conf)
	if err != nil {
		log.Printf("Parse failed %v", err)
		return nil, err
	}
	return conf, nil
}

func CheckConfigIsExist(name string) error {
	configDir, _ := os.Getwd()
	configPath := path.Join(configDir, fmt.Sprintf("%s.yaml", name))
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logs.Log.Debug("Config file not exist")
		return os.ErrNotExist
	}
	return nil
}

func NewListenerConfig(user string, certs, privateKey, ca []byte) error {
	// save config as yaml file
	configDir, _ := os.Getwd()
	configFile := path.Join(configDir, fmt.Sprintf("%s.yaml", user))

	config := &ClientConfig{
		Operator:      user,
		LHost:         "localhost",
		LPort:         5004,
		Token:         generateOperatorToken(),
		CACertificate: string(ca),
		PrivateKey:    string(privateKey),
		Certificate:   string(certs),
	}

	yamlData, err := yaml.Marshal(config)
	if err != nil {
		logs.Log.Errorf("marshal config to yaml failed: %v", err)
		return err
	}
	err = ioutil.WriteFile(configFile, yamlData, 0644)
	if err != nil {
		logs.Log.Errorf("write config to file failed: %v", err)
		return err
	}
	return nil
}

// NewClientConfig - new config and save in local file
func NewClientConfig(host, user string, port int, certs, privateKey, ca []byte) (string, error) {
	token := generateOperatorToken()
	// new config
	config := &ClientConfig{
		Operator:      user,
		LHost:         host,
		LPort:         port,
		Token:         token,
		CACertificate: string(ca),
		PrivateKey:    string(privateKey),
		Certificate:   string(certs),
	}
	// save config as yaml file
	configDir := GetConfigDir()
	configFile := path.Join(configDir, fmt.Sprintf("%s_%s.yaml", user, host))

	yamlData, err := yaml.Marshal(config)
	if err != nil {
		logs.Log.Errorf("marshal config to yaml failed: %v", err)
		return token, err
	}

	err = ioutil.WriteFile(configFile, yamlData, 0644)
	if err != nil {
		logs.Log.Errorf("write config to file failed: %v", err)
		return token, err
	}
	return token, nil
}

func GetConfigs() ([]string, error) {
	var files []string

	// Traverse all files in the specified directory.
	err := filepath.Walk(GetConfigDir(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(info.Name(), ".yaml") || strings.HasSuffix(info.Name(), ".yml")) {
			files = append(files, info.Name())
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// generateOperatorToken - Generate a new operator auth token
func generateOperatorToken() string {
	buf := make([]byte, 32)
	n, err := rand.Read(buf)
	if err != nil || n != len(buf) {
		logs.Log.Error("Failed to generate random token")
	}
	return hex.EncodeToString(buf)
}

func RemoveConfig(name string, caType int) error {
	var configPath string
	if caType == 2 {
		configPath = fmt.Sprintf("%s.yaml", name)
	} else if caType == 1 {
		configDir := GetConfigDir()
		configPath = path.Join(configDir, fmt.Sprintf("%s_%s.yaml", name, host))
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logs.Log.Debug("Config file not exist")
		return os.ErrNotExist
	}
	err := os.Remove(configPath)
	if err != nil {
		logs.Log.Errorf("remove config file failed: %v", err)
		return err
	}
	return nil
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
