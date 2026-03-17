package certutils

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"net"
)

func WrapWithTls(lsn net.Listener, cert *implanttypes.CertConfig) (net.Listener, error) {
	pair, err := tls.X509KeyPair([]byte(cert.Cert), []byte(cert.Key))
	if err != nil {
		return nil, err
	}

	return tls.NewListener(lsn, TlsConfig(pair)), nil
}

func GetTlsConfig(config *implanttypes.CertConfig) (*tls.Config, error) {
	cert, err := tls.X509KeyPair([]byte(config.Cert), []byte(config.Key))
	if err != nil {
		return nil, err
	}

	return TlsConfig(cert), nil
}

// GetMTlsConfig creates a TLS config that requires and verifies client certificates
// against the given CA. This enables mutual TLS for pipeline connections.
func GetMTlsConfig(serverCert *implanttypes.CertConfig, caCert *implanttypes.CertConfig) (*tls.Config, error) {
	cert, err := tls.X509KeyPair([]byte(serverCert.Cert), []byte(serverCert.Key))
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(caCert.Cert))

	tlsCfg := TlsConfig(cert)
	tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
	tlsCfg.ClientCAs = caCertPool
	return tlsCfg, nil
}

func TlsConfig(cert tls.Certificate) *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		},
	}
}
