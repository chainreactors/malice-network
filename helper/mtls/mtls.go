<<<<<<<< HEAD:client/utils/mtls.go
package utils
========
package mtls
>>>>>>>> 6b39ff6 (refactor bubble cli and mtls func):helper/mtls/mtls.go

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
	"os"
)

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

<<<<<<<< HEAD:client/utils/mtls.go
func MTLSConnect(config *assets.ClientConfig) (clientrpc.RootRPCClient, *grpc.ClientConn, error) {
	tlsConfig, err := getTLSConfig(config.CACertificate, config.Certificate, config.PrivateKey)
========
func Connect(config *assets.ClientConfig) (*grpc.ClientConn, error) {
	tlsConfig, err := GetTLSConfig(config.CACertificate, config.Certificate, config.PrivateKey)
>>>>>>>> 6b39ff6 (refactor bubble cli and mtls func):helper/mtls/mtls.go
	if err != nil {
		return nil, nil, err
	}
	transportCreds := credentials.NewTLS(tlsConfig)
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(transportCreds),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(consts.ClientMaxReceiveMessageSize)),
	}
	ctx, cancel := context.WithTimeout(context.Background(), consts.DefaultTimeout)
	defer cancel()
	connection, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", config.LHost, config.LPort), options...)
	if err != nil {
		return nil, nil, err
	}
	return clientrpc.NewRootRPCClient(connection), connection, nil
}
