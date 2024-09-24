package common

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/utils/mtls"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
)

var (
	serverRootPath    = ".malice"
	certsPath         = path.Join(serverRootPath, "certs")
	defaultConfigName = "default_localhost.yaml"
)

func getCertDir() string {
	//rootDir := assets.GetRootAppDir()
	// test
	if _, err := os.Stat(certsPath); os.IsNotExist(err) {
		err := os.MkdirAll(certsPath, 0700)
		if err != nil {
			logs.Log.Errorf("Failed to create cert dir: %v", err)
		}
	}
	return certsPath
}

// GetCertificateAuthorityPEM - Get PEM encoded CA cert/key
func GetCertificateAuthorityPEM(caType string) ([]byte, []byte, error) {
	caType = path.Base(caType)
	caCertPath := path.Join(getCertDir(), "localhost_root_crt.pem")
	caKeyPath := path.Join(getCertDir(), "localhost_root_key.pem")

	certPEM, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		logs.Log.Error(err.Error())
		return nil, nil, err
	}

	keyPEM, err := ioutil.ReadFile(caKeyPath)
	if err != nil {
		logs.Log.Error(err.Error())
		return nil, nil, err
	}
	return certPEM, keyPEM, nil
}

// GetCertificateAuthority - Get the current CA certificate
func GetCertificateAuthority(caType string) (*x509.Certificate, *rsa.PrivateKey, error) {
	certPEM, keyPEM, err := GetCertificateAuthorityPEM(caType)
	if err != nil {
		return nil, nil, err
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		logs.Log.Error("Failed to parse certificate PEM")
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		logs.Log.Errorf("Failed to parse certificate: %v", err)
		return nil, nil, err
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		logs.Log.Error("Failed to parse certificate PEM")
		return nil, nil, err
	}
	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		logs.Log.Errorf("Failed to parse EC private key: %v", err)
		return nil, nil, err
	}

	return cert, key, nil
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
			return verifyCertificate(caCertificate, rawCerts)
		},
		ServerName: "client",
	}
	return tlsConfig, nil
}

// VerifyCertificate - Verify a certificate
func verifyCertificate(caCertificate string, rawCerts [][]byte) error {
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

func RpcOptions() []grpc.DialOption {
	var options []grpc.DialOption
	//caCertX509, _, err := GetCertificateAuthority("root")
	//caCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertX509.Raw})
	//if err != nil {
	//	panic(err)
	//}
	configFile := filepath.Join(assets.GetConfigDir(), defaultConfigName)
	config, err := mtls.ReadConfig(configFile)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	tlsConfig, err := GetTLSConfig(config.CACertificate, config.Certificate, config.PrivateKey)
	transportCreds := credentials.NewTLS(tlsConfig)
	options = []grpc.DialOption{
		grpc.WithTransportCredentials(transportCreds),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(consts.ClientMaxReceiveMessageSize)),
	}
	return options
}
