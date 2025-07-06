package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func (rpc *Server) RegisterPipeline(ctx context.Context, req *clientpb.Pipeline) (*clientpb.Empty, error) {
	lns, err := core.Listeners.Get(req.ListenerId)
	if err != nil {
		return nil, err
	}
	req.Ip = lns.IP
	err = db.SavePipelineByRegister(req)
	if err != nil {
		return nil, err
	}
	var profileReq *clientpb.Profile
	if req.Parser == consts.ImplantPulse {
		profileReq = &clientpb.Profile{
			Name:            req.Name + "_default",
			PulsePipelineId: req.Name,
		}
	} else {
		profileReq = &clientpb.Profile{
			Name:       req.Name + "_default",
			PipelineId: req.Name,
		}
	}
	err = db.NewProfile(profileReq)
	if err != nil {
		logs.Log.Errorf("new profile %s failed %v", req.Name, err)
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) SyncPipeline(ctx context.Context, req *clientpb.Pipeline) (*clientpb.Empty, error) {
	_, err := db.SavePipeline(models.FromPipelinePb(req))
	if err != nil {
		return nil, err
	}
	job := core.Jobs.AddPipeline(req)
	if req.Tls != nil && req.Tls.Domain != "" {
		err := db.UpdateCert(req.Tls.Domain, req.Tls.Cert.Cert, req.Tls.Cert.Key)
		if err != nil {
			return nil, err
		}
		subject, err := certs.ExtractCertificateSubject(req.Tls.Cert.Cert)
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
				req.Tls.Domain, certs.Acme, subject.CommonName, subject.Organization[0], subject.Country[0], subject.Locality[0],
				ouStr, stStr)
			core.EventBroker.Publish(core.Event{
				EventType: consts.EventCert,
				IsNotify:  false,
				Message:   msg,
				Important: true,
			})
		}
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventJob,
		Op:        consts.CtrlPipelineSync,
		Important: true,
		Job:       job.ToProtobuf(),
	})
	return &clientpb.Empty{}, nil
}

func (rpc *Server) ListPipelines(ctx context.Context, req *clientpb.Listener) (*clientpb.Pipelines, error) {
	var result []*clientpb.Pipeline
	if req.Id != "" {
		pipe, err := core.Listeners.Get(req.Id)
		if err != nil {
			return nil, err
		}
		result = append(result, pipe.AllPipelines()...)
	} else {
		core.Listeners.Range(func(key, value any) bool {
			lns := value.(*core.Listener)
			result = append(result, lns.AllPipelines()...)
			return true
		})
	}
	return &clientpb.Pipelines{Pipelines: result}, nil
}

func (rpc *Server) StartPipeline(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	pipelineDB, err := db.FindPipeline(req.Name)
	if err != nil {
		return nil, err
	}
	if req.CertName != "" {
		tls, err := db.FindCertificate(req.CertName)
		if err != nil {
			return nil, err
		}
		pipelineDB.Tls.Cert = &types.CertConfig{
			Cert: tls.CertPEM,
			Key:  tls.KeyPEM,
		}
		pipelineDB.Tls.CA = &types.CertConfig{
			Cert: tls.CACertPEM,
			Key:  tls.CAKeyPEM,
		}
		pipelineDB.Tls.Enable = true
		pipelineDB.CertName = req.CertName
		_, err = db.SavePipeline(pipelineDB)
		if err != nil {
			return nil, err
		}
	}
	lns, err := core.Listeners.Get(pipelineDB.ListenerId)
	if err != nil {
		return nil, err
	}
	pipelineProto := pipelineDB.ToProtobuf()
	job := &core.Job{
		ID:       core.NextJobID(),
		Pipeline: pipelineProto,
		Name:     req.Name,
	}

	lns.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlPipelineStart,
		Job:  job.ToProtobuf()})
	pipeline := pipelineDB.ToProtobuf()
	err = db.EnablePipeline(pipeline.Name)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) StopPipeline(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	job, err := core.Jobs.Get(req.Name)
	if err != nil {
		return nil, err
	}
	lns, err := core.Listeners.Get(job.Pipeline.ListenerId)
	if err != nil {
		return nil, err
	}
	lns.RemovePipeline(job.Pipeline)
	lns.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlPipelineStop,
		Job:  job.ToProtobuf(),
	})
	err = db.DisablePipeline(job.Pipeline.Name)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) DeletePipeline(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	pipelineDB, err := db.FindPipeline(req.Name)
	if err != nil {
		return nil, err
	}
	pipeline := pipelineDB.ToProtobuf()
	lns, err := core.Listeners.Get(pipelineDB.ListenerId)
	if err != nil {
		return nil, err
	}
	lns.RemovePipeline(pipeline)
	lns.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlPipelineStop,
		Job: &clientpb.Job{
			Pipeline: pipeline,
		},
	})
	err = db.DeletePipeline(req.Name)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}
