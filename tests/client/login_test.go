package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
	"os"
	"path"
	"testing"
)

func TestLogin(t *testing.T) {
	t.Log(path.Join(assets.GetConfigDir(), "test_localhost_5004.yaml"))
	clientConfig, err := assets.ReadConfig(path.Join(assets.GetConfigDir(), "test_localhost_5004.yaml"))
	if err != nil {
		t.Log(err)
	}
	tlsConfig, err := getTLSConfig(clientConfig.CACertificate, clientConfig.Certificate, clientConfig.PrivateKey)
	if err != nil {
		t.Log(err)
	}
	transportCreds := credentials.NewTLS(tlsConfig)
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(transportCreds),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(consts.ClientMaxReceiveMessageSize)),
	}
	conn, err := grpc.Dial("localhost:5004", options...)
	if err != nil {
		fmt.Println(err)
	}
	t.Log("Dialing")
	client := clientrpc.NewMaliceRPCClient(conn)
	regReq := &clientpb.LoginReq{
		Host: "localhost",
		Port: 30009,
		Name: "test",
	}
	t.Log("Calling")
	// 调用服务器的 Register 方法并等待响应
	res, err := client.LoginClient(context.Background(), regReq)
	if err != nil {
		t.Log(err)
	}
	fmt.Println(res)
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
func getTLSConfig(caCertificate string, certificate string, privateKey string) (*tls.Config, error) {

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
