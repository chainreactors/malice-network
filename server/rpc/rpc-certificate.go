package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func (rpc *Server) AddCertificate(ctx context.Context, req *clientpb.TLS) (*clientpb.Empty, error) {
	var certModel *models.Certificate
	if req.Cert != nil {
		certModel = &models.Certificate{
			Name:    codenames.GetCodename(),
			Type:    certs.Imported,
			CertPEM: req.Cert.Cert,
			KeyPEM:  req.Cert.Key,
			//CACertPEM: req.Ca.Cert
		}
	} else if !req.AutoCert {
		tls, err := certutils.GenerateSelfTLS("", "")
		if err != nil {
			return nil, err
		}
		certModel = &models.Certificate{
			Name:      codenames.GetCodename(),
			Type:      certs.SelfSigned,
			CertPEM:   tls.Cert.Cert,
			KeyPEM:    tls.Cert.Key,
			CACertPEM: tls.CA.Cert,
			CAKeyPEM:  tls.CA.Key,
		}
	} else {
		tls, err := certutils.GetAutoCertTls(req)
		if err != nil {
			return nil, err
		}
		certModel = &models.Certificate{
			Name:    req.Domain,
			Type:    certs.AutoCert,
			CertPEM: tls.Cert.Cert,
			KeyPEM:  tls.Cert.Key,
			//CACertPEM: tls.Ca.Cert,
		}
	}
	err := db.SaveCertificate(certModel)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (rpc *Server) DeleteCertificate(ctx context.Context, req *clientpb.Cert) (*clientpb.Empty, error) {
	err := db.DeleteCertificate(req.Name)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (rpc *Server) GetAllCertificates(ctx context.Context, req *clientpb.Empty) (*clientpb.Certs, error) {
	var results *clientpb.Certs
	certModels, err := db.GetAllCertificates()
	if err != nil {
		return nil, err
	}
	for _, cert := range certModels {
		results.Certs = append(results.Certs, cert.ToProtobuf())
	}
	return results, nil
}

func (rpc *Server) UpdateCertificate(ctx context.Context, req *clientpb.Cert) (*clientpb.Empty, error) {
	err := db.UpdateCert(req.Name, req.Cert, req.Key)
	if err != nil {
		return nil, err
	}
	return nil, nil
}
