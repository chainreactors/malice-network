package web

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/server/configs"
	"github.com/chainreactors/malice-network/server/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net"
	"runtime/debug"
)

var webLogger = configs.SetLogFilePath("web", logs.Warn)

func StartMtlsClientListener(host string, port uint16) (*grpc.Server, net.Listener, error) {
	webLogger.Infof("Starting gRPC/mtls  listener on %s:%d", host, port)
	tlsConfg := getOperatorServerTLSConfig(host)

	creds := credentials.NewTLS(tlsConfg)
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		logs.Log.Errorf(err.Error())
		return nil, nil, err
	}
	options := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.MaxRecvMsgSize(consts.ServerMaxMessageSize),
		grpc.MaxSendMsgSize(consts.ServerMaxMessageSize),
	}
	options = append(options)
	grpcServer := grpc.NewServer(options...)
	clientrpc.RegisterMaliceRPCServer(grpcServer, rpc.NewServer())
	go func() {
		panicked := true
		defer func() {
			if panicked {
				logs.Log.Errorf("stacktrace from panic: %s", string(debug.Stack()))
			}
		}()
		if err := grpcServer.Serve(ln); err != nil {
			logs.Log.Warnf("gRPC server exited with error: %v", err)
		} else {
			panicked = false
		}
	}()
	return grpcServer, ln, nil
}

// getOperatorServerTLSConfig - Get the TLS config for the operator server
func getOperatorServerTLSConfig(host string) *tls.Config {
	caCert, _, err := certs.GetCertificateAuthority(certs.OperatorCA)
	if err != nil {
		logs.Log.Errorf("Failed to load CA %s", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCert)

	_, _, err = certs.OperatorServerGetCertificate(host)
	if errors.Is(err, certs.ErrCertDoesNotExist) {
		certs.OperatorServerGenerateCertificate(host)
	}
	certPEM, keyPEM, err := certs.OperatorServerGetCertificate(host)
	if err != nil {
		webLogger.Errorf("Failed to load certificate %s", err)
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		webLogger.Errorf("Error loading server certificate: %v", err)
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
