package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
)

func (rpc *Server) GenerateSelfCert(ctx context.Context, req *clientpb.Pipeline) (*clientpb.Empty, error) {
	if req.Name != "" {
		certModel, err := db.FindPipelineCert(req.Name, req.ListenerId)
		if err != nil {
			return nil, err
		}
		if certModel != nil {
			return &clientpb.Empty{}, nil
		}
	}

	if req.Tls.Cert == nil {
		tls, err := certutils.GenerateSelfTLS("", req.Tls.CertSubject)
		if err != nil {
			return nil, err
		}
		req.Tls = tls
	}
	certModel, err := db.SaveCertFromTLS(req.Tls, req.Name)
	if err != nil {
		return nil, err
	}

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

func (rpc *Server) GenerateAcmeCert(ctx context.Context, req *clientpb.Pipeline) (*clientpb.Empty, error) {
	lns, err := core.Listeners.Get(req.ListenerId)
	if err != nil {
		return nil, err
	}
	job := &core.Job{
		ID:       core.NextJobID(),
		Pipeline: req,
		Name:     req.Name,
	}
	lns.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlAcme,
		Job:  job.ToProtobuf()})
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
	certModel, err := db.SaveCertFromTLS(req.Tls, req.Name)
	if err != nil {
		return nil, err
	}
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
