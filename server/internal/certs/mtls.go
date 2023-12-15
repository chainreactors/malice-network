package certs

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/helper"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"log"
	"os"
	"path"
)

const (
	// MtlsImplantCA - Directory containing HTTPS server certificates
	MtlsImplantCA = "mtls-implant"
	MtlsServerCA  = "mtls-server"
)

// MtlsC2ServerGenerateECCCertificate - Generate a server certificate signed with a given CA
func MtlsC2ServerGenerateECCCertificate(host string) ([]byte, []byte, error) {
	cert, key := GenerateECCCertificate(MtlsServerCA, host, false, false)
	err := saveCertificate(MtlsServerCA, ECCKey, host, cert, key)
	return cert, key, err
}

// MtlsC2ImplantGenerateECCCertificate - Generate a server certificate signed with a given CA
func MtlsC2ImplantGenerateECCCertificate(name string) ([]byte, []byte, error) {
	cert, key := GenerateECCCertificate(MtlsImplantCA, name, false, true)
	err := saveCertificate(MtlsImplantCA, ECCKey, name, cert, key)
	return cert, key, err
}

// MtlsListenerGenerateRsaCertificate -
func MtlsListenerGenerateRsaCertificate(name string, isRoot bool) ([]byte, []byte, error) {
	if isRoot {
		var listenerCert *models.Certificate
		certsPath := path.Join(configs.ServerRootPath, "certs")
		// 检查是否已存在证书
		listenerCertPath := path.Join(certsPath, "listener_default_crt.pem")
		listenerKeyPath := path.Join(certsPath, "listener_default_key.pem")
		if helper.FileExists(listenerCertPath) && helper.FileExists(listenerKeyPath) {
			logs.Log.Info("Listener server CA certificates already exist.")
			dbSession := db.Session()
			result := dbSession.Where(
				&models.Certificate{
					CommonName: fmt.Sprintf("%s", name)}).First(&listenerCert)
			if result.Error != nil {
				logs.Log.Errorf("Failed to load CA %v", result.Error)
				return nil, nil, result.Error
			}
			return []byte(listenerCert.CertificatePEM), []byte(listenerCert.PrivateKeyPEM), nil
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

// VerifyCertificate - Verify a certificate
func VerifyCertificate(caCertificate string, rawCerts [][]byte) error {
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(caCertificate))
	if !ok {
		log.Printf("Failed to parse root certificate")
		os.Exit(3)
	}

	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		log.Printf("Failed to parse certificate: " + err.Error())
		return err
	}

	options := x509.VerifyOptions{
		Roots: roots,
	}
	if options.Roots == nil {
		panic("no root certificate")
	}
	if _, err := cert.Verify(options); err != nil {
		log.Printf("Failed to verify certificate: " + err.Error())
		return err
	}

	return nil
}

// GetTLSConfig - Get the TLS config for the operator server
func GetTLSConfig(caCertificate string, certificate string, privateKey string) (*tls.Config, error) {

	certPEM, err := tls.X509KeyPair([]byte(certificate), []byte(privateKey))
	if err != nil {
		log.Printf("Cannot parse client certificate: %v", err)
		return nil, err
	}

	// Load CA cert
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(caCertificate))

	// Setup config with custom certificate validation routine
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{certPEM},
		RootCAs:            caCertPool,
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			return VerifyCertificate(caCertificate, rawCerts)
		},
	}
	return tlsConfig, nil
}
