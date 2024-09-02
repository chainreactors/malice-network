package certs

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/chainreactors/files"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/helper"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"os"
	"path"
)

const (
	OperatorCA = iota + 1
	ListenerCA
	ImplantCA
	RootCA
)

const (
	// RSAKey - Namespace for RSA keys
	RSAKey            = "rsa"
	RootName          = "Root"
	ListenerNamespace = "listener" // Listener servers
	rootCert          = "root_ca.pem"
	rootKey           = "root_key.pem"
	serverCert        = "server_crt.pem"
	serverKey         = "server_key.pem"
)

var certsLog = logs.Log

// RemoveCertificate - Remove a certificate from the cert store
func RemoveCertificate(caType int, keyType string, commonName string) error {
	var err error
	if keyType != RSAKey {
		return fmt.Errorf("invalid key type '%s'", keyType)
	}
	dbSession := db.Session()
	if caType == ListenerCA {
		err = dbSession.Where(&models.Listener{
			Name: commonName,
		}).Delete(&models.Listener{}).Error
		commonName = "listener." + commonName
	} else {
		err = dbSession.Where(&models.Operator{
			Name: commonName,
		}).Delete(&models.Operator{}).Error
	}
	err = dbSession.Where(&models.Certificate{
		CAType:     caType,
		KeyType:    keyType,
		CommonName: commonName,
	}).Delete(&models.Certificate{}).Error
	if err != nil {
		return err
	}

	return err
}

// --------------------------------
//  Generic Certificate Functions
// --------------------------------

// GetOperatorServerMTLSConfig - Get the TLS config for the operator server
func GetOperatorServerMTLSConfig(host string) *tls.Config {
	caCert, _, err := GetCertificateAuthority()
	if err != nil {
		logs.Log.Errorf("Failed to load CA %s", err)
		return nil
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCert)
	certPEM, keyPEM, err := GenerateServerCert(host)
	if err != nil {
		logs.Log.Errorf("Failed to load certificate %s", err)
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		logs.Log.Errorf("Error loading server certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	return tlsConfig
}

func GenerateRootCert() error {
	rootCertPath := path.Join(configs.CertsPath, rootCert)
	rootKeyPath := path.Join(configs.CertsPath, rootKey)
	if files.IsExist(rootCertPath) && files.IsExist(rootKeyPath) {
		return nil
	}
	cert, key, err := certs.GenerateCACert(RootName)
	if err != nil {
		return err
	}
	err = certs.SaveToPEMFile(rootCertPath, cert)
	if err != nil {
		return err
	}
	err = certs.SaveToPEMFile(rootKeyPath, key)
	if err != nil {
		return err
	}
	err = db.AddCertificate(RootCA, RSAKey, RootName, cert, key)
	if err != nil {
		return err
	}
	return nil
}

func GeneratePipelineCert(config *configs.TlsConfig) ([]byte, []byte, error) {
	pipelineCAPath := path.Join(configs.CertsPath, fmt.Sprintf("%s_ca_cert.pem", config.Name))
	pipelineCertPath := path.Join(configs.ListenerPath, fmt.Sprintf("%s_crt.pem", config.Name))
	pipelineKeyPath := path.Join(configs.ListenerPath, fmt.Sprintf("%s_key.pem", config.Name))
	var cert, key []byte
	var caCertByte, caKeyByte []byte
	var err error
	if helper.FileExists(pipelineCAPath) && helper.FileExists(pipelineCertPath) && helper.FileExists(pipelineKeyPath) {
		cert, err = os.ReadFile(pipelineCertPath)
		if err != nil {
			return nil, nil, err
		}
		key, err = os.ReadFile(pipelineKeyPath)
		if err != nil {
			return nil, nil, err
		}
		return cert, key, nil
	}
	if config.CertFile != "" && config.KeyFile != "" {
		cert = []byte(config.CertFile)
		key = []byte(config.KeyFile)
		err = os.WriteFile(pipelineCertPath, cert, 0644)
		if err != nil {
			return nil, nil, err
		}
		err = os.WriteFile(pipelineKeyPath, key, 0644)
		if err != nil {
			return nil, nil, err
		}
		err = db.AddCertificate(ImplantCA, RSAKey, config.Name, cert, key)
		if err != nil {
			return nil, nil, err
		}
		return cert, key, nil
	} else {
		caCertByte, caKeyByte, err = certs.GenerateCACert(config.Name)
		if err != nil {
			return nil, nil, err
		}
		caCert, caKey, err := ParseCertificateAuthority(caCertByte, caKeyByte)
		if err != nil {
			return nil, nil, err
		}
		cert, key, err = certs.GenerateChildCert(config.Name, true, caCert, caKey)
		err = certs.SaveToPEMFile(pipelineCertPath, caCertByte)
		if err != nil {
			return nil, nil, err
		}
		err = certs.SaveToPEMFile(pipelineCertPath, cert)
		if err != nil {
			return nil, nil, err
		}
		err = certs.SaveToPEMFile(pipelineKeyPath, key)
		if err != nil {
			return nil, nil, err
		}
	}
	err = db.AddCertificate(RootCA, RSAKey, config.Name, caCertByte, caKeyByte)
	if err != nil {
		return nil, nil, err
	}
	err = db.AddCertificate(ImplantCA, RSAKey, config.Name, cert, key)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

func GenerateServerCert(name string) ([]byte, []byte, error) {
	certPath := path.Join(configs.CertsPath, serverCert)
	certKeyPath := path.Join(configs.CertsPath, serverKey)
	if helper.FileExists(certPath) && helper.FileExists(certKeyPath) {
		certBytes, err := os.ReadFile(certPath)
		if err != nil {
			return nil, nil, err
		}
		keyBytes, err := os.ReadFile(certKeyPath)
		if err != nil {
			return nil, nil, err
		}
		return certBytes, keyBytes, nil
	}
	ca, caKey, err := GetCertificateAuthority()
	if err != nil {
		return nil, nil, err
	}
	cert, key, err := certs.GenerateChildCert(name, true, ca, caKey)
	if err != nil {
		return nil, nil, err
	}
	err = certs.SaveToPEMFile(certPath, cert)
	if err != nil {
		return nil, nil, err
	}
	err = certs.SaveToPEMFile(certKeyPath, key)
	if err != nil {
		return nil, nil, err
	}
	err = db.AddCertificate(OperatorCA, RSAKey, name, cert, key)
	if err != nil {
		return nil, nil, err
	}
	//err = db.CreateOperator(name)
	return cert, key, nil
}

func GenerateClientCert(host, name string, port int) (*mtls.ClientConfig, error) {
	ca, caKey, err := GetCertificateAuthority()
	caCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Raw})
	if err != nil {
		return nil, err
	}
	cert, key, err := certs.GenerateChildCert(name, true, ca, caKey)
	if err != nil {
		return nil, err
	}
	err = db.AddCertificate(OperatorCA, RSAKey, name, cert, key)
	if err != nil {
		return nil, err
	}
	return &mtls.ClientConfig{
		Operator:      name,
		LHost:         host,
		LPort:         port,
		Type:          mtls.Client,
		CACertificate: string(caCert),
		PrivateKey:    string(key),
		Certificate:   string(cert),
	}, nil
}

func GenerateListenerCert(host, name string, port int) (*mtls.ClientConfig, error) {
	var cert, key []byte
	ca, caKey, err := GetCertificateAuthority()
	caCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Raw})
	if err != nil {
		return nil, err
	}
	cert, key, err = certs.GenerateChildCert(host, true, ca, caKey)
	if err != nil {
		return nil, err
	}
	err = db.AddCertificate(ListenerCA, RSAKey, name, cert, key)
	if err != nil {
		return nil, err
	}
	return &mtls.ClientConfig{
		Operator:      name,
		LHost:         host,
		LPort:         port,
		Type:          mtls.Listener,
		CACertificate: string(caCert),
		PrivateKey:    string(key),
		Certificate:   string(cert),
	}, nil
}
