package generate

import (
	"encoding/pem"
	"fmt"
	"github.com/chainreactors/files"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/helper"
	"github.com/chainreactors/malice-network/server/internal/certs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path"
)

const (
	defaultClient = "test_localhost_5004"
)

// GenerateRootCA - Initialize the root CA
func GenerateRootCA() {
	certsPath := path.Join(configs.ServerRootPath, "certs")
	err := os.MkdirAll(certsPath, 0744)
	if err != nil {
		logs.Log.Errorf("Failed to generate file paths: %v", err)
	}
	// 检查是否已存在证书
	rootCertPath := path.Join(certsPath, "localhost_root_crt.pem")
	rootKeyPath := path.Join(certsPath, "localhost_root_key.pem")
	if helper.FileExists(rootCertPath) && helper.FileExists(rootKeyPath) {
		logs.Log.Info("Root CA certificates already exist.")
		return
	}
	_, _, err = certs.InitRSACertificate("localhost", "root", true, false)
	if err != nil {
		logs.Log.Errorf("Failed to generate server certificate: %v", err)
	}
}

// GenerateClientCA - Initialize the client CA
func GenerateClientCA(host, user string) ([]byte, []byte, error) {
	cert, key, err := certs.InitRSACertificate(host, user, false, true)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, err
}

// ServerInitUserCert - Initialize the client cert by server
func ServerInitUserCert(name string) error {
	if files.IsExist(path.Join(assets.GetConfigDir(), fmt.Sprintf("%s.yaml", defaultClient))) {
		logs.Log.Info("Client certificate already exist.")
		return nil
	}
	cert, key, err := certs.InitRSACertificate("localhost", name, false, true)
	if err != nil {
		return err
	}
	ca, _, caErr := certs.GetCertificateAuthority(certs.SERVERCA)
	caCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Raw})
	if caErr != nil {
		return caErr
	}
	token := models.GenerateOperatorToken()
	var client = &assets.ClientConfig{
		Operator:      name,
		LHost:         "localhost",
		LPort:         5004,
		Token:         token,
		CACertificate: string(caCert),
		Certificate:   string(cert),
		PrivateKey:    string(key),
	}
	// save config as yaml file
	configDir := assets.GetConfigDir()
	configFile := path.Join(configDir, fmt.Sprintf("%s_%s_%d.yaml", name, client.LHost, client.LPort))

	// 使用 YAML 库将 config 结构体序列化为 YAML 数据
	yamlData, err := yaml.Marshal(client)
	if err != nil {
		return err
	}
	// 将 YAML 数据写入文件
	err = ioutil.WriteFile(configFile, yamlData, 0644)
	if err != nil {
		return err
	}
	dbSession := db.Session()
	err = dbSession.Create(&models.Operator{
		Name:  name,
		Token: token,
	}).Error
	if err != nil {
		return err
	}
	return nil
}
