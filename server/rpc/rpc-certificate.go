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
			return rpc.publishCertEvent(consts.CtrlCertCreate, certModel)
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
		return rpc.publishCertEvent(consts.CtrlCertCreate, certModel)
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
		return rpc.publishCertEvent(consts.CtrlCertCreate, certModel)
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

	return rpc.publishCertEvent(consts.CtrlCertCreate, certModel)
}

func (rpc *Server) DeleteCertificate(ctx context.Context, req *clientpb.Cert) (*clientpb.Empty, error) {
	err := db.DeleteCertificate(req.Name)
	if err != nil {
		return nil, err
	}
	publishCertificateLifecycleEvent(consts.CtrlCertDelete, req.Name, "")
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
	if req == nil || req.Cert == nil {
		return nil, fmt.Errorf("certificate is required")
	}

	caPEM := ""
	if req.Ca != nil {
		caPEM = req.Ca.Cert
	}

	err := db.UpdateCert(req.Cert.Name, req.Cert.Cert, req.Cert.Key, caPEM, req.Cert.Comment)
	if err != nil {
		return nil, err
	}
	publishCertificateLifecycleEvent(consts.CtrlCertUpdate, req.Cert.Name, "")
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
	operation := consts.CtrlCertCreate
	if existing != nil {
		operation = consts.CtrlCertUpdate
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
		return rpc.publishCertEvent(operation, certModel)
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

func (rpc *Server) publishCertEvent(operation string, certModel *models.Certificate) (*clientpb.Empty, error) {
	msg, err := certs.FormatSubject(certModel.Name, certModel.Type, certModel.CertPEM)
	if err != nil {
		return nil, err
	}
	publishCertificateLifecycleEvent(operation, certModel.Name, msg)
	return &clientpb.Empty{}, nil
}

func publishCertificateLifecycleEvent(operation, name, message string) {
	if message == "" {
		message = fmt.Sprintf("certificate %s changed", name)
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventCert,
		Op:        operation,
		IsNotify:  false,
		Message:   message,
		Important: true,
	})
}
