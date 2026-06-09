package forwardrpc

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/chainreactors/IoM-go/consts"
	mtls "github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const forwardClientCommonName = "server-forward-client"

func ServerOptions(authPath string) ([]grpc.ServerOption, error) {
	clientConf, err := mtls.ReadConfig(authPath)
	if err != nil {
		return nil, fmt.Errorf("read forward listener auth: %w", err)
	}
	tlsConfig, err := ServerTLSConfig(clientConf)
	if err != nil {
		return nil, err
	}
	return []grpc.ServerOption{
		grpc.Creds(credentials.NewTLS(tlsConfig)),
		grpc.MaxRecvMsgSize(consts.ServerMaxMessageSize),
		grpc.MaxSendMsgSize(consts.ServerMaxMessageSize),
	}, nil
}

func DialOptions(serverName string) ([]grpc.DialOption, error) {
	tlsConfig, err := ClientTLSConfig(serverName)
	if err != nil {
		return nil, err
	}
	return []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(consts.ServerMaxMessageSize)),
	}, nil
}

func ServerTLSConfig(clientConf *mtls.ClientConfig) (*tls.Config, error) {
	certPEM, caPEM, keyPEM, err := forwardAuthPEM(clientConf)
	if err != nil {
		return nil, err
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse forward listener certificate: %w", err)
	}
	if err := requireCertificateUsage(certPEM, x509.ExtKeyUsageServerAuth, "forward listener auth"); err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("parse forward listener CA certificate")
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

func ClientTLSConfig(serverName string) (*tls.Config, error) {
	if serverName == "" {
		return nil, fmt.Errorf("forward listener server name is empty")
	}
	certPEM, keyPEM, err := certutils.GetOrCreateForwardClientCert(forwardClientCommonName)
	if err != nil {
		return nil, err
	}
	if err := requireCertificateUsage(certPEM, x509.ExtKeyUsageClientAuth, "forward server client cert"); err != nil {
		return nil, err
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse forward client certificate: %w", err)
	}
	caCert, _, err := certutils.GetCertificateAuthority()
	if err != nil {
		return nil, fmt.Errorf("load forward client CA: %w", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCert)
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		ServerName:   serverName,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			return verifyPeerCertificate(caCertPool, rawCerts, x509.ExtKeyUsageServerAuth)
		},
		MinVersion: tls.VersionTLS13,
	}, nil
}

func forwardAuthPEM(clientConf *mtls.ClientConfig) ([]byte, []byte, []byte, error) {
	if clientConf == nil {
		return nil, nil, nil, fmt.Errorf("forward listener auth config is nil")
	}
	certPEM := []byte(clientConf.Certificate)
	caPEM := []byte(clientConf.CACertificate)
	keyPEM := []byte(clientConf.PrivateKey)
	if len(certPEM) == 0 || len(caPEM) == 0 || len(keyPEM) == 0 {
		return nil, nil, nil, fmt.Errorf("forward listener auth config missing certificate, CA, or private key")
	}
	return certPEM, caPEM, keyPEM, nil
}

func requireCertificateUsage(certPEM []byte, usage x509.ExtKeyUsage, label string) error {
	cert, err := parseLeafCertificate(certPEM)
	if err != nil {
		return err
	}
	if hasExtKeyUsage(cert.ExtKeyUsage, usage) {
		return nil
	}
	return fmt.Errorf("%s certificate missing required extended key usage %s", label, extKeyUsageName(usage))
}

func verifyPeerCertificate(roots *x509.CertPool, rawCerts [][]byte, usage x509.ExtKeyUsage) error {
	if len(rawCerts) == 0 {
		return fmt.Errorf("peer did not provide a certificate")
	}
	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return fmt.Errorf("parse peer certificate: %w", err)
	}
	intermediates := x509.NewCertPool()
	for _, raw := range rawCerts[1:] {
		intermediate, err := x509.ParseCertificate(raw)
		if err != nil {
			return fmt.Errorf("parse peer intermediate certificate: %w", err)
		}
		intermediates.AddCert(intermediate)
	}
	_, err = cert.Verify(x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{usage},
	})
	if err != nil {
		return fmt.Errorf("verify peer certificate: %w", err)
	}
	return nil
}

func parseLeafCertificate(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("decode certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}
	return cert, nil
}

func hasExtKeyUsage(usages []x509.ExtKeyUsage, want x509.ExtKeyUsage) bool {
	for _, usage := range usages {
		if usage == want {
			return true
		}
	}
	return false
}

func extKeyUsageName(usage x509.ExtKeyUsage) string {
	switch usage {
	case x509.ExtKeyUsageClientAuth:
		return "clientAuth"
	case x509.ExtKeyUsageServerAuth:
		return "serverAuth"
	default:
		return fmt.Sprintf("%d", usage)
	}
}
