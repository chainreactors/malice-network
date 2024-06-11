package assets

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/helper"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"
)

var (
	MaliceDirName = ".config/malice"
	ConfigDirName = "configs"
)

type ClientConfig struct {
	Operator      string `json:"operator"` // This value is actually ignored for the most part (cert CN is used instead)
	LHost         string `json:"lhost"`
	LPort         int    `json:"lport"`
	Type          string `json:"type"`
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

// NewConfig - new config and save in local file
func NewConfig(host, user string, port int, certs, privateKey, ca []byte) error {
	// new config
	config := &ClientConfig{
		Operator:      user,
		LHost:         host,
		LPort:         port,
		CACertificate: string(ca),
		PrivateKey:    string(privateKey),
		Certificate:   string(certs),
	}
	// save config as yaml file
	configDir := GetConfigDir()
	configFile := path.Join(configDir, fmt.Sprintf("%s_%s_%d.yaml", user, host, port))

	// 使用 YAML 库将 config 结构体序列化为 YAML 数据
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		logs.Log.Errorf("marshal config to yaml failed: %v", err)
		return err
	}
	// 将 YAML 数据写入文件
	err = ioutil.WriteFile(configFile, yamlData, 0644)
	if err != nil {
		logs.Log.Errorf("write config to file failed: %v", err)
		return err
	}
	return nil
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

func MvConfig(oldPath string) error {
	fileName := filepath.Base(oldPath)
	newPath := filepath.Join(GetConfigDir(), fileName)
	err := helper.CopyFile(oldPath, newPath)
	if err != nil {
		return err
	}
	err = helper.RemoveFile(oldPath)
	if err != nil {
		return err
	}
	return nil
}
