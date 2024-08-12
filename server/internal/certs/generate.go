package certs

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/chainreactors/files"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/helper"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"os"
	"path"
	"path/filepath"
)

const (
	OperatorCA = iota + 1
	ListenerCA
	ImplantCA
	RootCA
)

const (
	clientNamespace   = "client"   // Operator clients
	serverNamespace   = "server"   // Operator servers
	ListenerNamespace = "listener" // Listener servers
	rootCert          = "root_ca.pem"
	rootKey           = "root_key.pem"
	implantCa         = "implant_ca.pem"
	implantKey        = "implant_key.pem"
	serverCert        = "server_crt.pem"
	serverKey         = "server_key.pem"
)

var certsLog = logs.Log

// ClientGenerateCertificate - Generate a certificate signed with a given CA
func ClientGenerateCertificate(host, name string, port int, clientType int) (*mtls.ClientConfig, error) {
	ca, _, caErr := GetCertificateAuthority()
	caCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Raw})
	if caErr != nil {
		return nil, caErr
	}
	if clientType == OperatorCA {
		cert, key := GenerateRSACertificate(OperatorCA, name, false, true, nil)
		err := saveCertificate(OperatorCA, RSAKey,
			fmt.Sprintf("%s", name), cert, key)
		if err != nil {
			return nil, err
		}
		clientConfig := mtls.NewClientConfig(host, name, port, clientType, cert, key, caCert)
		err = db.CreateOperator(name)
		if err != nil {
			return nil, err
		}
		return clientConfig, nil
	} else {
		var certBytes, keyBytes []byte
		var err error
		certsPath := path.Join(configs.ServerRootPath, "certs")
		// check if listenerCert exist
		cwd, _ := os.Getwd()
		if files.IsExist(filepath.Join(cwd, name+".yaml")) {
			logs.Log.Debug("Listener server CA certificates already exist.")
			certBytes, keyBytes, err = CheckCertIsExist(certsPath, "", name, ListenerCA)
			if err != nil {
				return nil, err
			}
		} else {
			certBytes, keyBytes = GenerateRSACertificate(ListenerCA, name, false, true, nil)
			err = saveCertificate(ListenerCA, RSAKey,
				fmt.Sprintf("%s.%s", ListenerNamespace, name), certBytes, keyBytes)
			if err != nil {
				return nil, err
			}
			err = db.CreateListener(name)
			if err != nil {
				return nil, err
			}
		}
		clientConfig := mtls.NewClientConfig(host, name, port, clientType, certBytes, keyBytes, caCert)
		if err != nil {
			return nil, err
		}
		return clientConfig, err
	}
}

// ClientGetCertificate - Helper function to fetch a client cert
func ClientGetCertificate(operator string) ([]byte, []byte, error) {
	return GetECCCertificate(OperatorCA, fmt.Sprintf("%s.%s", clientNamespace, operator))
}

// ClientRemoveCertificate - Helper function to remove a client cert
func ClientRemoveCertificate(operator string) error {
	return RemoveCertificate(OperatorCA, ECCKey, fmt.Sprintf("%s.%s", clientNamespace, operator))
}

// ServerGetCertificate - Helper function to fetch a server cert
func ServerGetCertificate(hostname string) ([]byte, []byte, error) {
	return GetRSACertificate(OperatorCA, fmt.Sprintf("%s.%s", serverNamespace, hostname))
}

// ServerGenerateCertificate - Generate a certificate signed with a given CA
func ServerGenerateCertificate(name string, isCA bool, cfgPath string) ([]byte, []byte, error) {
	if isCA {
		certPath := path.Join(configs.CertsPath, rootCert)
		certKey := path.Join(configs.CertsPath, rootKey)
		if helper.FileExists(certPath) && helper.FileExists(certKey) {
			logs.Log.Debug("Root CA certificates already exist.")
			return nil, nil, nil
		} else {
			cert, key := GenerateRSACertificate(RootCA, name, isCA, false, nil)
			err := removeOldCerts(cfgPath)
			if err != nil {
				return cert, key, err
			}
			if certErr := os.WriteFile(certPath, cert, 0777); certErr != nil {
				return nil, nil, certErr
			}
			if keyErr := os.WriteFile(certKey, key, 0777); keyErr != nil {
				return nil, nil, keyErr
			}
			return cert, key, err
		}
	} else {
		certPath := path.Join(configs.CertsPath, serverCert)
		certKey := path.Join(configs.CertsPath, serverKey)
		if helper.FileExists(certPath) && helper.FileExists(certKey) {
			logs.Log.Debug("Mtls server CA certificates already exist.")
			certBytes, keyBytes, err := CheckCertIsExist(certPath, certKey, name, OperatorCA)
			if err != nil {
				return certBytes, keyBytes, err
			}
			return certBytes, keyBytes, nil
		}
		cert, key, err := configs.LoadMiscConfig()
		if err != nil && errors.Is(err, configs.ErrNoConfig) {
			cert, key = GenerateRSACertificate(OperatorCA, name, false, false, nil)
			err := saveCertificate(OperatorCA, RSAKey, fmt.Sprintf("%s", name), cert, key)
			if certErr := os.WriteFile(certPath, cert, 0o777); certErr != nil {
				return nil, nil, certErr
			}
			if keyErr := os.WriteFile(certKey, key, 0o777); keyErr != nil {
				return nil, nil, keyErr
			}
			return cert, key, err
		} else if err != nil {
			return nil, nil, err
		}
		return cert, key, nil
	}
}

// ClientListCertificates - Get all client certificates
func ClientListCertificates() []*x509.Certificate {
	operatorCerts := []*models.Certificate{}
	dbSession := db.Session()
	result := dbSession.Where(&models.Certificate{CAType: OperatorCA}).Find(&operatorCerts)
	if result.Error != nil {
		certsLog.Error(result.Error.Error())
		return []*x509.Certificate{}
	}

	certsLog.Infof("Found %d operator certs ...", len(operatorCerts))

	certs := []*x509.Certificate{}
	for _, operator := range operatorCerts {
		block, _ := pem.Decode([]byte(operator.CertificatePEM))
		if block == nil {
			certsLog.Warn("failed to parse certificate PEM")
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			certsLog.Warnf("failed to parse x.509 certificate %v", err)
			continue
		}
		certs = append(certs, cert)
	}
	return certs
}
