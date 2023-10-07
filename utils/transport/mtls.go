package transport

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/chainreactors/malice-network/utils/certs"
	"net"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	kb = 1024
	mb = kb * 1024
	gb = mb * 1024

	// ServerMaxMessageSize - Server-side max GRPC message size
	ServerMaxMessageSize = 2 * gb
)

//var (
//	mtlsLog = log.NamedLogger("transport", "mtls")
//)

// StartMtlsClientListener - Start a mutual TLS listener
func StartMtlsClientListener(host string, port uint16) (*grpc.Server, net.Listener, error) {
	// TODO - log Starting gRPC/mtls  listener on
	//mtlsLog.Infof("Starting gRPC/mtls  listener on %s:%d", host, port)

	tlsConfig := getOperatorServerTLSConfig("multiplayer")

	creds := credentials.NewTLS(tlsConfig)
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		// TODO - log mtls error
		//mtlsLog.Error(err)
		return nil, nil, err
	}
	options := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.MaxRecvMsgSize(ServerMaxMessageSize),
		grpc.MaxSendMsgSize(ServerMaxMessageSize),
	}
	options = append(options, initMiddleware(true)...)
	grpcServer := grpc.NewServer(options...)
	// TODO - RegisterSliverRPCServer
	//rpcpb.RegisterSliverRPCServer(grpcServer, rpc.NewServer())
	go func() {
		panicked := true
		defer func() {
			if panicked {
				// TODO - log stacktrace from panic
				fmt.Println(debug.Stack())
				//mtlsLog.Errorf("stacktrace from panic: %s", string(debug.Stack()))
			}
		}()
		if err := grpcServer.Serve(ln); err != nil {
			// TODO - log gRPC server exited with error
			//mtlsLog.Warnf("gRPC server exited with error: %v", err)
		} else {
			panicked = false
		}
	}()
	return grpcServer, ln, nil
}

// getOperatorServerTLSConfig - Generate the TLS configuration, we do now allow the end user
// to specify any TLS paramters, we choose sensible defaults instead
func getOperatorServerTLSConfig(host string) *tls.Config {
	caCertPtr, _, err := certs.GetCertificateAuthority(certs.OperatorCA)
	if err != nil {
		// TODO - log Invalid ca type (%s): %v
		//mtlsLog.Fatalf("Invalid ca type (%s): %v", certs.OperatorCA, host)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCertPtr)

	_, _, err = certs.OperatorServerGetCertificate(host)
	if err == certs.ErrCertDoesNotExist {
		certs.OperatorServerGenerateCertificate(host)
	}

	certPEM, keyPEM, err := certs.OperatorServerGetCertificate(host)
	if err != nil {
		// TODO - log Failed to generate or fetch certificate
		//mtlsLog.Errorf("Failed to generate or fetch certificate %s", err)
		return nil
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		// TODO - log Error loading server certificate
		//mtlsLog.Fatalf("Error loading server certificate: %v", err)
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
