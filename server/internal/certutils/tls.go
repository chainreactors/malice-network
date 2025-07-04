package certutils

import (
	"crypto/tls"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"net"

	"github.com/chainreactors/malice-network/server/internal/configs"
)

func WrapWithTls(lsn net.Listener, config *configs.CertConfig) (net.Listener, error) {
	pair, err := tls.X509KeyPair([]byte(config.Cert), []byte(config.Key))
	if err != nil {
		return nil, err
	}

	return tls.NewListener(lsn, TlsConfig(pair)), nil
}

func WrapToTlsConfig(config *configs.CertConfig) (*tls.Config, error) {
	if !config.Enable {
		return nil, nil
	}
	pair, err := tls.X509KeyPair([]byte(config.Cert), []byte(config.Key))
	if err != nil {
		return nil, err
	}

	return TlsConfig(pair), nil
}

func GetTlsConfig(tlsConfig *types.CertConfig) (*tls.Config, error) {
	cert, err := tls.X509KeyPair([]byte(tlsConfig.Cert), []byte(tlsConfig.Key))
	if err != nil {
		return nil, err
	}

	return TlsConfig(cert), nil
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

func SaveTlsCert(tls *types.TlsConfig, piplineName, listenerID string) (*types.TlsConfig, string, error) {
	var certModel *models.Certificate
	var err error
	name := codenames.GetCodename()
	if tls.Cert == nil && !tls.AutoCert {
		tls, err = GenerateSelfTLS(piplineName, listenerID)
		if err != nil {
			return tls, "", err
		}
		certModel = &models.Certificate{
			Name:      name,
			Type:      certs.SelfSigned,
			CACertPEM: tls.CA.Cert,
			CAKeyPEM:  tls.CA.Key,
			CertPEM:   tls.Cert.Cert,
			KeyPEM:    tls.Cert.Key,
		}
	} else if tls.Cert != nil && !tls.AutoCert {
		certModel = &models.Certificate{
			Name:      name,
			Type:      certs.Imported,
			CertPEM:   tls.Cert.Cert,
			KeyPEM:    tls.Cert.Key,
			CACertPEM: tls.CA.Cert,
		}
	} else if tls.Cert != nil && tls.AutoCert {
		certModel = &models.Certificate{
			Name:    tls.Domain,
			Type:    certs.AutoCert,
			CertPEM: tls.Cert.Cert,
			KeyPEM:  tls.Cert.Key,
			Domain:  tls.Domain,
			//CACertPEM: pipelineModel.Tls.CA.Cert,
		}
		name = tls.Domain
	}
	err = db.SaveCertificate(certModel)
	if err != nil {
		return tls, name, err
	}
	return tls, name, nil
}
