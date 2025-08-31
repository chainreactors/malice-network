package certutils

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

func ParseCertificateAuthority(certPEM, keyPEM []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	certBlock, _ := pem.Decode(certPEM)
	var err error
	if certBlock == nil {
		certsLog.Error("Failed to parse certificate PEM")
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		certsLog.Errorf("Failed to parse certificate: %v", err)
		return nil, nil, err
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		certsLog.Error("Failed to parse certificate PEM")
		return nil, nil, err
	}
	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		logs.Log.Errorf("Failed to parse EC private key: %v", err)
		return nil, nil, err
	}

	return cert, key, nil
}

// GetCertificateAuthority - Get the current CA certificate
func GetCertificateAuthority() (*x509.Certificate, *rsa.PrivateKey, error) {
	certPEM, keyPEM, err := GetCertificateAuthorityPEM(path.Join(configs.GetCertDir(), certs.RootCert), path.Join(configs.GetCertDir(), certs.RootKey))
	if err != nil {
		return nil, nil, err
	}
	return ParseCertificateAuthority(certPEM, keyPEM)
}

// GetCertificateAuthorityPEM - Get PEM encoded CA cert/key
func GetCertificateAuthorityPEM(caCertPath, caKeyPath string) ([]byte, []byte, error) {
	certPEM, err := os.ReadFile(caCertPath)
	if err != nil {
		certsLog.Error(err.Error())
		return nil, nil, err
	}

	keyPEM, err := os.ReadFile(caKeyPath)
	if err != nil {
		certsLog.Error(err.Error())
		return nil, nil, err
	}
	return certPEM, keyPEM, nil
}

// SaveCertificateAuthority - Save the certificate and the key to the filesystem
// doesn't return an error because errors are fatal. If we can't generate CAs,
// then we can't secure communication and we should die a horrible death.
func SaveCertificateAuthority(caType int, cert []byte, key []byte) {

	storageDir := configs.GetCertDir()
	if _, err := os.Stat(storageDir); os.IsNotExist(err) {
		os.MkdirAll(storageDir, 0700)
	}

	// CAs get written to the filesystem since we control the names and makes them
	// easier to move around/backup
	certFilePath := filepath.Join(storageDir, fmt.Sprintf("%s-ca-cert.pem", caType))
	keyFilePath := filepath.Join(storageDir, fmt.Sprintf("%s-ca-key.pem", caType))

	err := ioutil.WriteFile(certFilePath, cert, 0600)
	if err != nil {
		certsLog.Errorf("Failed write certificate data to: %s", certFilePath)
	}

	err = ioutil.WriteFile(keyFilePath, key, 0600)
	if err != nil {
		certsLog.Errorf("Failed write certificate data to: %s", keyFilePath)
	}
}
