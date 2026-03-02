package rpc

import (
	"context"
	"fmt"
	"strings"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func (rpc *Server) GenerateSelfCert(ctx context.Context, req *clientpb.Pipeline) (*clientpb.Empty, error) {
	if req == nil {
		return nil, fmt.Errorf("pipeline is nil")
	}

	if req.Tls == nil {
		return nil, fmt.Errorf("pipeline %s tls config is nil", req.Name)
	}

	pipelineName := strings.TrimSpace(req.Name)
	attachToPipeline := pipelineName != ""
	// Standalone certificate management: allow generating/importing certs without binding to a pipeline.
	if !attachToPipeline {
		if req.Tls.Cert != nil && req.Tls.Cert.Cert != "" {
			certModel, err := db.SaveCertFromTLS(req.Tls, "", "")
			if err != nil {
				return nil, err
			}
			return rpc.publishCertEvent(certModel)
		}

		tls, err := certutils.GenerateSelfTLS("", req.Tls.CertSubject)
		if err != nil {
			return nil, err
		}
		req.Tls = tls

		certModel, err := db.SaveCertFromTLS(req.Tls, "", "")
		if err != nil {
			return nil, err
		}
		return rpc.publishCertEvent(certModel)
	}

	// Pipeline-bound certificate generation: only act when TLS is enabled.
	if !req.Tls.Enable {
		return &clientpb.Empty{}, nil
	}

	if req.Tls.Cert != nil && req.Tls.Cert.Cert != "" {
		certModel, err := db.SaveCertFromTLS(req.Tls, pipelineName, req.ListenerId)
		if err != nil {
			return nil, err
		}
		return rpc.publishCertEvent(certModel)
	}

	certModel, err := db.FindPipelineCert(pipelineName, req.ListenerId)
	if err != nil {
		return nil, err
	}
	if certModel != nil {
		req.Tls = certModel.ToProtobuf()
		return &clientpb.Empty{}, nil
	}

	tls, err := certutils.GenerateSelfTLS("", req.Tls.CertSubject)
	if err != nil {
		return nil, err
	}
	req.Tls = tls

	certModel, err = db.SaveCertFromTLS(req.Tls, pipelineName, req.ListenerId)
	if err != nil {
		return nil, err
	}

	return rpc.publishCertEvent(certModel)
}

func (rpc *Server) DeleteCertificate(ctx context.Context, req *clientpb.Cert) (*clientpb.Empty, error) {
	err := db.DeleteCertificate(req.Name)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) GetAllCertificates(ctx context.Context, req *clientpb.Empty) (*clientpb.Certs, error) {
	certs := &clientpb.Certs{}
	certModels, err := db.GetAllCertificates()
	if err != nil {
		return nil, err
	}
	for _, cert := range certModels {
		certs.Certs = append(certs.Certs, cert.ToProtobuf())
	}
	return certs, nil
}

func (rpc *Server) UpdateCertificate(ctx context.Context, req *clientpb.TLS) (*clientpb.Empty, error) {
	err := db.UpdateCert(req.Cert.Name, req.Cert.Cert, req.Cert.Key, req.Ca.Cert)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) GenerateAcmeCert(ctx context.Context, req *clientpb.Pipeline) (*clientpb.Empty, error) {
	return nil, fmt.Errorf("deprecated: use ObtainAcmeCert with DNS-01 challenge instead")
}

func (rpc *Server) ObtainAcmeCert(ctx context.Context, req *clientpb.AcmeRequest) (*clientpb.Empty, error) {
	certPEM, keyPEM, err := certutils.ObtainCert(
		req.Domain,
		req.Provider,
		req.Email,
		req.CaUrl,
		req.Credentials,
	)
	if err != nil {
		return nil, fmt.Errorf("ACME certificate obtain failed: %w", err)
	}

	// Check if cert already exists, update or create
	existing, _ := db.FindCertificate(req.Domain)
	if existing != nil {
		err = db.UpdateCert(req.Domain, string(certPEM), string(keyPEM), "")
		if err != nil {
			return nil, fmt.Errorf("failed to update certificate: %w", err)
		}
	} else {
		certModel := &models.Certificate{
			Name:    req.Domain,
			Type:    "acme",
			Domain:  req.Domain,
			CertPEM: string(certPEM),
			KeyPEM:  string(keyPEM),
		}
		err = db.SaveCertificate(certModel)
		if err != nil {
			return nil, fmt.Errorf("failed to save certificate: %w", err)
		}
	}

	certModel, _ := db.FindCertificate(req.Domain)
	if certModel != nil {
		return rpc.publishCertEvent(certModel)
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) DownloadCertificate(ctx context.Context, req *clientpb.Cert) (*clientpb.TLS, error) {
	certificate, err := db.FindCertificate(req.Name)
	if err != nil {
		return nil, err
	}
	return certificate.ToProtobuf(), nil
}

func (rpc *Server) SaveAcmeCert(ctx context.Context, req *clientpb.Pipeline) (*clientpb.Empty, error) {
	return nil, fmt.Errorf("deprecated: use ObtainAcmeCert with DNS-01 challenge instead")
}

func (rpc *Server) publishCertEvent(certModel *models.Certificate) (*clientpb.Empty, error) {
	msg, err := certs.FormatSubject(certModel.Name, certModel.Type, certModel.CertPEM)
	if err != nil {
		return nil, err
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventCert,
		IsNotify:  false,
		Message:   msg,
		Important: true,
	})
	return &clientpb.Empty{}, nil
}
