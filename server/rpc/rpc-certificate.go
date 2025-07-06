package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func (rpc *Server) GenerateSelfCertificate(ctx context.Context, req *clientpb.TLS) (*clientpb.Empty, error) {
	var certModel *models.Certificate
	if req.Cert != nil {
		certModel = &models.Certificate{
			Name:    codenames.GetCodename(),
			Type:    certs.Imported,
			CertPEM: req.Cert.Cert,
			KeyPEM:  req.Cert.Key,
		}

	} else if !req.Acme {
		subject := certutils.CertificateSubjectToPkixName(req.CertSubject)
		tls, err := certutils.GenerateSelfTLS("", subject)
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
	}
	err := db.SaveCertificate(certModel)
	if err != nil {
		return nil, err
	}
	subject, err := certs.ExtractCertificateSubject(certModel.CertPEM)
	if err != nil {
		return nil, err
	}
	if subject != nil {
		ouStr := ""
		if len(subject.OrganizationalUnit) > 0 {
			ouStr = subject.OrganizationalUnit[0]
		}
		stStr := ""
		if len(subject.Province) > 0 {
			stStr = subject.Province[0]
		}
		msg := fmt.Sprintf("cert %s (type: %s) generate sucess, CN: %s, O: %s, C: %s, L: %s, OU: %s, ST: %s",
			certModel.Name, certModel.Type, subject.CommonName, subject.Organization[0], subject.Country[0], subject.Locality[0],
			ouStr, stStr)
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventCert,
			IsNotify:  false,
			Message:   msg,
			Important: true,
		})
	}
	return &clientpb.Empty{}, nil
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

func (rpc *Server) UpdateCertificate(ctx context.Context, req *clientpb.Cert) (*clientpb.Empty, error) {
	err := db.UpdateCert(req.Name, req.Cert, req.Key)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (rpc *Server) GenerateAcmeCert(ctx context.Context, req *clientpb.TLS) (*clientpb.Empty, error) {
	pipelineDB, err := db.FindPipeline(req.PipelineName)
	if err != nil {
		return nil, err
	}
	lns, err := core.Listeners.Get(pipelineDB.ListenerId)
	if err != nil {
		return nil, err
	}
	pipelineProto := pipelineDB.ToProtobuf()
	job := &core.Job{
		ID:       core.NextJobID(),
		Pipeline: pipelineProto,
		Name:     req.PipelineName,
	}
	lns.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlAcme,
		Job:  job.ToProtobuf()})
	return &clientpb.Empty{}, nil
}
