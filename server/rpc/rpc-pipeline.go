package rpc

import (
	"context"
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/server/internal/certutils"
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
	var pipelineModel *models.Pipeline
	pipelineModel, err = db.FindPipeline(req.Name)
	if err != nil && !errors.Is(err, db.ErrRecordNotFound) {
		return nil, err
	} else if err != nil && errors.Is(err, db.ErrRecordNotFound) {
		pipelineModel = models.FromPipelinePb(req)
		if req.CertName == "" && pipelineModel.Tls.Enable {
			tls, certName, err := certutils.SaveTlsCert(pipelineModel.Tls, pipelineModel.Name, pipelineModel.ListenerId)
			if err != nil {
				return nil, err
			}
			pipelineModel.Tls = tls
			pipelineModel.CertName = certName
		} else if pipelineModel.Tls.Enable {
			certModel, err := db.FindCertificate(req.CertName)
			if err != nil {
				return nil, err
			}
			pipelineModel.Tls = &types.TlsConfig{
				Cert: &types.CertConfig{
					Cert: certModel.CertPEM,
					Key:  certModel.KeyPEM,
				},
				Domain: certModel.Domain,
				Enable: req.Enable,
			}
			pipelineModel.CertName = req.CertName
		}
		_, err = db.SavePipeline(pipelineModel)
		if err != nil {
			return nil, err
		}
	}
	var profileReq *clientpb.Profile
	if req.Parser == consts.ImplantPulse {
		profileReq = &clientpb.Profile{
			Name:            req.Name + "_default",
			PipelineId:      req.BeaconPipeline,
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
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventJob,
			Op:        consts.CtrlAutoCert,
			Important: true,
			Job:       job.ToProtobuf(),
		})
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
		pipelineDB.Tls.Cert.Cert = tls.CertPEM
		pipelineDB.Tls.Cert.Key = tls.KeyPEM
		pipelineDB.Tls.CA.Cert = tls.CACertPEM
		pipelineDB.Tls.CA.Key = tls.CAKeyPEM
	}
	lns, err := core.Listeners.Get(pipelineDB.ListenerId)
	if err != nil {
		return nil, err
	}
	pipelineProto := pipelineDB.ToProtobuf()
	pipelineProto.Target = req.Target
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
