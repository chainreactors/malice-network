package certutils

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
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
		err = dbSession.Where(&models.Operator{
			Name: commonName,
			Type: mtls.Listener,
		}).Delete(&models.Operator{}).Error
		commonName = "listener." + commonName
	} else {
		err = dbSession.Where(&models.Operator{
			Name: commonName,
			Type: mtls.Client,
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
	if fileutils.Exist(rootCertPath) && fileutils.Exist(rootKeyPath) {
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

func GenerateServerCert(name string) ([]byte, []byte, error) {
	certPath := path.Join(configs.CertsPath, serverCert)
	certKeyPath := path.Join(configs.CertsPath, serverKey)
	if fileutils.Exist(certPath) && fileutils.Exist(certKeyPath) {
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
	cert, key, err := certs.GenerateChildCert(name, false, ca, caKey)
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
		Host:          host,
		Port:          port,
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
		Host:          host,
		Port:          port,
		Type:          mtls.Listener,
		CACertificate: string(caCert),
		PrivateKey:    string(key),
		Certificate:   string(cert),
	}, nil
}

func GenerateTlsCert(name, listenerID string) (string, string, error) {
	cert, key, err := db.FindPipelineCert(name, listenerID)
	if err != nil && !errors.Is(err, db.ErrRecordNotFound) {
		return "", "", err
	}
	if cert != "" && key != "" {
		return cert, key, nil
	}
	caCertByte, caKeyByte, err := certs.GenerateCACert(name)
	if err != nil {
		return "", "", err
	}
	caCert, caKey, err := ParseCertificateAuthority(caCertByte, caKeyByte)
	if err != nil {
		return "", "", err
	}
	certByte, keyByte, err := certs.GenerateChildCert(name, true, caCert, caKey)
	return string(certByte), string(keyByte), nil
}
