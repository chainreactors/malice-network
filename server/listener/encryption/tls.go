package encryption

import (
	"crypto/tls"
	"github.com/chainreactors/malice-network/server/internal/certs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"net"
)

func WrapWithTls(conn net.Conn, config *configs.TlsConfig) (net.Conn, error) {
	cert, key := certs.GenerateRSACertificate("pipeline", "", false, false, config)
	pair, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{pair}}
	return tls.Server(conn, tlsConfig), nil
}
