package mtls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
)

// VerifyCertificate - Verify a certificate
func VerifyCertificate(caCertificate []byte, rawCerts [][]byte) error {
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(caCertificate)
	if !ok {
		log.Printf("Failed to parse root certificate")
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
		log.Printf("no root certificate")
	}
	if _, err := cert.Verify(options); err != nil {
		log.Printf("Failed to verify certificate: " + err.Error())
		return err
	}

	return nil
}

func GetGrpcOptions(caCertificate []byte, certificate []byte, privateKey []byte, servername string) ([]grpc.DialOption, error) {
	certPEM, err := tls.X509KeyPair(certificate, privateKey)
	if err != nil {
		log.Printf("Cannot parse client certificate: %v", err)
		return nil, err
	}

	// Load CA cert
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCertificate)

	// Setup config with custom certificate validation routine
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{certPEM},
		RootCAs:            caCertPool,
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			return VerifyCertificate(caCertificate, rawCerts)
		},
	}
	tlsConfig.ServerName = servername
	transportCreds := credentials.NewTLS(tlsConfig)
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(transportCreds),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(consts.ClientMaxReceiveMessageSize)),
	}
	return options, nil
}

func Connect(config *ClientConfig) (*grpc.ClientConn, error) {
	options, err := GetGrpcOptions([]byte(config.CACertificate), []byte(config.Certificate), []byte(config.PrivateKey), config.Type)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), consts.DefaultDuration)
	defer cancel()
	connection, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", config.LHost, config.LPort), options...)
	if err != nil {
		return nil, err
	}
	return connection, nil
}
