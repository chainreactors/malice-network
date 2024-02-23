package encryption

import (
	"crypto/tls"
	"github.com/chainreactors/malice-network/server/internal/certs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"net"
)

func WrapWithTls(lsn net.Listener, config *configs.TlsConfig) (net.Listener, error) {
	cert, key := certs.GenerateRSACertificate(0, "", false, false, config)
	pair, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{pair}}
	return tls.NewListener(lsn, tlsConfig), nil
}
