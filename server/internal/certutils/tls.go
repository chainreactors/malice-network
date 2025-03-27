package certutils

import (
	"crypto/tls"
	"net"

	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
)

func WrapWithTls(lsn net.Listener, config *configs.CertConfig) (net.Listener, error) {
	pair, err := tls.X509KeyPair([]byte(config.Cert), []byte(config.Key))
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{pair}}
	return tls.NewListener(lsn, tlsConfig), nil
}

func WrapToTlsConfig(config *configs.CertConfig) (*tls.Config, error) {
	if !config.Enable {
		return nil, nil
	}
	pair, err := tls.X509KeyPair([]byte(config.Cert), []byte(config.Key))
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{pair}}
	return tlsConfig, nil
}

func GetTlsConfig(tlsConfig *clientpb.TLS) (*tls.Config, error) {
	cert, err := tls.X509KeyPair([]byte(tlsConfig.Cert), []byte(tlsConfig.Key))
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}
