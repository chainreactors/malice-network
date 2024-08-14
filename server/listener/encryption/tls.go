package encryption

import (
	"crypto/tls"
	"github.com/chainreactors/malice-network/server/internal/certs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"net"
)

func WrapWithTls(lsn net.Listener, config *configs.TlsConfig) (net.Listener, error) {
	cert, key, err := certs.GenerateListenerCertificate(config)
	if err != nil {
		return nil, err
	}
	pair, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{pair}}
	return tls.NewListener(lsn, tlsConfig), nil
}

func WrapToTlsConfig(config *configs.TlsConfig) (*tls.Config, error) {
	cert, key, err := certs.GenerateListenerCertificate(config)
	if err != nil {
		return nil, err
	}
	pair, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{pair}}
	return tlsConfig, nil
}
