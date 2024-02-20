package certs

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/helper"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"os"
	"path"
)

const (
	// OperatorCA - Directory containing operator certificates
	OperatorCA      = "operator"
	HTTPSCA         = "host"
	clientNamespace = "client" // Operator clients
	serverNamespace = "server" // Operator servers
	ListenerCA      = "listener"
)

var certsLog = logs.Log

// OperatorClientGenerateCertificate - Generate a certificate signed with a given CA
func OperatorClientGenerateCertificate(operator string) ([]byte, []byte, error) {
	cert, key := GenerateRSACertificate(OperatorCA, operator, false, true, nil)
	err := saveCertificate(OperatorCA, RSAKey, fmt.Sprintf("%s.%s", "root", operator), cert, key)
	filename := fmt.Sprintf(configs.CertsPath+"/%s_%s", OperatorCA, SERVERCA)
	if certErr := os.WriteFile(filename+"_crt.pem", cert, 0o777); certErr != nil {
		return nil, nil, certErr
	}
	if keyErr := os.WriteFile(filename+"_key.pem", key, 0o777); keyErr != nil {
		return nil, nil, keyErr
	}
	return cert, key, err
}

// OperatorClientGetCertificate - Helper function to fetch a client cert
func OperatorClientGetCertificate(operator string) ([]byte, []byte, error) {
	return GetECCCertificate(OperatorCA, fmt.Sprintf("%s.%s", clientNamespace, operator))
}

// OperatorClientRemoveCertificate - Helper function to remove a client cert
func OperatorClientRemoveCertificate(operator string) error {
	return RemoveCertificate(OperatorCA, ECCKey, fmt.Sprintf("%s.%s", clientNamespace, operator))
}

// OperatorServerGetCertificate - Helper function to fetch a server cert
func OperatorServerGetCertificate(hostname string) ([]byte, []byte, error) {
	return GetRSACertificate(OperatorCA, fmt.Sprintf("%s.%s", serverNamespace, hostname))
}

// OperatorServerGenerateCertificate - Generate a certificate signed with a given CA
func OperatorServerGenerateCertificate(hostname string) ([]byte, []byte, error) {
	certsPath := path.Join(configs.ServerRootPath, "certs")
	// check if listenerCert exist
	serverCertPath := path.Join(certsPath, "server_operator_crt.pem")
	serverKeyPath := path.Join(certsPath, "server_operator_key.pem")
	if helper.FileExists(serverCertPath) && helper.FileExists(serverKeyPath) {
		logs.Log.Debug("Mtls server CA certificates already exist.")
		certBytes, keyBytes, err := CheckCertIsExist(serverCertPath, serverKeyPath, OperatorName)
		if err != nil {
			return certBytes, keyBytes, err
		}
		return certBytes, keyBytes, nil
	}
	cert, key := GenerateRSACertificate(OperatorCA, hostname, false, false, nil)
	err := saveCertificate(OperatorCA, RSAKey, fmt.Sprintf("%s.%s", serverNamespace, hostname), cert, key)
	filename := fmt.Sprintf(configs.CertsPath+"/%s_%s", serverNamespace, OperatorCA)
	if certErr := os.WriteFile(filename+"_crt.pem", cert, 0o777); certErr != nil {
		return nil, nil, certErr
	}
	if keyErr := os.WriteFile(filename+"_key.pem", key, 0o777); keyErr != nil {
		return nil, nil, keyErr
	}
	return cert, key, err
}

// OperatorClientListCertificates - Get all client certificates
func OperatorClientListCertificates() []*x509.Certificate {
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
