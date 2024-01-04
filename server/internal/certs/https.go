package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/helper"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"io/ioutil"
	"os"
	"path"
)

const (
	// SERVERCA - Directory containing operator certificates
	SERVERCA = "root"
	// CLIENTCA - Directory containing client certificates
	CLIENTCA = "client"
)

// InitRSACertificate - Generate a server certificate signed with a given CA
func InitRSACertificate(host, user string, isCA, isClient bool) ([]byte, []byte, error) {
	certsLog.Debugf("Generating TLS certificate (RSA) for '%s' ...", host)

	var privateKey interface{}
	var err error
	var caType string

	// Generate private key
	privateKey, err = rsa.GenerateKey(rand.Reader, RsaKeySize())
	if err != nil {
		certsLog.Errorf("Failed to generate private key %v", err)
		return nil, nil, err
	}
	subject := randomSubject(host)
	switch isClient {
	case true:
		caType = CLIENTCA
	case false:
		caType = SERVERCA
	}
	cert, key := generateCertificate(caType, (*subject), isCA, isClient, privateKey)
	err = saveCertificate(caType, RSAKey, fmt.Sprintf("%s.%s", host, user), cert, key)
	// 保存到文件
	if isCA {
		filename := fmt.Sprintf(configs.CertsPath+"/%s_%s", host, user)
		if certErr := os.WriteFile(filename+"_crt.pem", cert, 0o777); certErr != nil {
			return nil, nil, certErr
		}
		if keyErr := os.WriteFile(filename+"_key.pem", key, 0o777); keyErr != nil {
			return nil, nil, keyErr
		}
	}
	return cert, key, nil
}

// MtlsListenerGenerateRsaCertificate -
func MtlsListenerGenerateRsaCertificate(name string, isRoot bool) ([]byte, []byte, error) {
	if isRoot {
		certsPath := path.Join(configs.ServerRootPath, "certs")
		// check if listenerCert exist
		listenerCertPath := path.Join(certsPath, "listener_default_crt.pem")
		listenerKeyPath := path.Join(certsPath, "listener_default_key.pem")
		if helper.FileExists(listenerCertPath) && helper.FileExists(listenerKeyPath) {
			logs.Log.Info("Listener server CA certificates already exist.")
			certBytes, err := ioutil.ReadFile(listenerCertPath)
			if err != nil {
				logs.Log.Errorf("Error reading Listener certificate file: %s", err)
				return nil, nil, err
			}
			keyBytes, err := ioutil.ReadFile(listenerKeyPath)
			if err != nil {
				logs.Log.Errorf("Error reading Listener key file: %s", err)
				return nil, nil, err
			}
			err = saveCertificate(ListenerCA, RSAKey, fmt.Sprintf("%s", name), certBytes, keyBytes)
			return certBytes, keyBytes, nil
		}
	}
	cert, key := GenerateRSACertificate(OperatorCA, name, false, true)
	err := saveCertificate(ListenerCA, RSAKey, fmt.Sprintf("%s", name), cert, key)
	filename := fmt.Sprintf(configs.CertsPath+"/%s_%s", ListenerCA, name)
	if certErr := os.WriteFile(filename+"_crt.pem", cert, 0o777); certErr != nil {
		return nil, nil, certErr
	}
	if keyErr := os.WriteFile(filename+"_key.pem", key, 0o777); keyErr != nil {
		return nil, nil, keyErr
	}
	return cert, key, err
}
