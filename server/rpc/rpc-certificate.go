package rpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"sort"
	"strings"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/types"
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
	certName := strings.TrimSpace(req.Cert.Name)
	if certName == "" {
		return nil, fmt.Errorf("certificate name is required")
	}

	fields := make(map[string]interface{})
	certPEM := req.Cert.Cert
	keyPEM := req.Cert.Key
	if strings.TrimSpace(certPEM) != "" || strings.TrimSpace(keyPEM) != "" {
		if strings.TrimSpace(certPEM) == "" || strings.TrimSpace(keyPEM) == "" {
			return nil, fmt.Errorf("cert and key must be provided together")
		}
		if _, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM)); err != nil {
			return nil, fmt.Errorf("invalid certificate key pair: %w", err)
		}
		fields["cert_pem"] = certPEM
		fields["key_pem"] = keyPEM
	}
	if certType := strings.TrimSpace(req.Cert.Type); certType != "" {
		fields["type"] = certType
	}
	if req.Cert.Comment != "" {
		fields["comment"] = req.Cert.Comment
	}
	if req.Ca != nil {
		fields["ca_cert_pem"] = req.Ca.Cert
		if req.Ca.Cert == "" {
			fields["ca_key_pem"] = ""
		} else if req.Ca.Key != "" {
			fields["ca_key_pem"] = req.Ca.Key
		}
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("no certificate fields were provided")
	}

	if err := db.UpdateCertFields(certName, fields); err != nil {
		return nil, err
	}
	publishCertificateLifecycleEvent(consts.CtrlCertUpdate, certName, "")
	return &clientpb.Empty{}, nil
}

func (rpc *Server) ApplyCertificate(ctx context.Context, req *clientpb.CertificateApplyRequest) (*clientpb.CertificateApplyResult, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	certName := strings.TrimSpace(req.GetCertName())
	if certName == "" {
		return nil, fmt.Errorf("cert_name is required")
	}
	cert, err := db.FindCertificate(certName)
	if err != nil {
		return nil, err
	}
	if _, err := tls.X509KeyPair([]byte(cert.CertPEM), []byte(cert.KeyPEM)); err != nil {
		return nil, fmt.Errorf("certificate %s is invalid: %w", certName, err)
	}

	query := db.NewPipelineQuery().WhereCertName(certName)
	if req.GetListenerId() != "" {
		query = query.WhereListenerID(req.GetListenerId())
	}
	if req.GetPipelineName() != "" {
		query = query.WhereName(req.GetPipelineName())
	}
	references, err := query.Find()
	if err != nil {
		return nil, err
	}
	sort.Slice(references, func(i, j int) bool {
		left := references[i].ListenerId + ":" + references[i].Name
		right := references[j].ListenerId + ":" + references[j].Name
		return left < right
	})

	result := &clientpb.CertificateApplyResult{CertName: certName}
	for _, reference := range references {
		target := &clientpb.CertificateApplyTarget{
			ListenerId:   reference.ListenerId,
			PipelineName: reference.Name,
			PipelineType: reference.Type,
		}
		var applyErr error
		switch reference.Type {
		case consts.WebsitePipeline:
			_, applyErr = rpc.UpdateWebsiteTLS(ctx, &clientpb.PipelineTLSUpdate{
				ListenerId: reference.ListenerId,
				Name:       reference.Name,
				Mode:       clientpb.TLSUpdateMode_TLS_UPDATE_MODE_EXISTING_CERT,
				CertName:   certName,
			})
		case consts.HTTPPipeline, consts.TCPPipeline:
			_, applyErr = rpc.UpdatePipelineTLS(ctx, &clientpb.PipelineTLSUpdate{
				ListenerId: reference.ListenerId,
				Name:       reference.Name,
				Mode:       clientpb.TLSUpdateMode_TLS_UPDATE_MODE_EXISTING_CERT,
				CertName:   certName,
			})
		default:
			applyErr = fmt.Errorf("pipeline type %s does not support certificate reload", reference.Type)
		}
		if applyErr != nil {
			target.Error = applyErr.Error()
		} else {
			target.Applied = true
		}
		result.Targets = append(result.Targets, target)
	}
	return result, nil
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
		err = db.UpdateCertFields(req.Domain, map[string]interface{}{
			"cert_pem": string(certPEM),
			"key_pem":  string(keyPEM),
			"type":     certs.Acme,
		})
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
