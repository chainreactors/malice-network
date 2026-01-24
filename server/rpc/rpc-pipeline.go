package rpc

import (
	"context"
	"fmt"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/implanttypes"
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
	pipelineModel := models.FromPipelinePb(req)
	_, err = db.SavePipeline(pipelineModel)
	if err != nil {
		return nil, err
	}
	profileReq := &clientpb.Profile{
		Name:       req.Name + "_default",
		PipelineId: req.Name,
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
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventJob,
		Op:        consts.CtrlPipelineSync,
		Important: true,
		Job:       job.ToProtobuf(),
	})
	return &clientpb.Empty{}, nil
}

func (rpc *Server) ListPipelines(ctx context.Context, req *clientpb.Listener) (*clientpb.Pipelines, error) {
	pipelines, err := db.ListPipelinesByListener(req.Id)
	if err != nil {
		return nil, err
	}
	return pipelines.ToProtobuf(), nil
}

func (rpc *Server) StartPipeline(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	listenerID := req.GetListenerId()
	if listenerID == "" && req.Pipeline != nil {
		listenerID = req.Pipeline.ListenerId
	}
	if listenerID == "" {
		return nil, fmt.Errorf("listener_id required")
	}

	pipelineDB, err := db.FindPipelineByListener(req.Name, listenerID)
	if err != nil {
		return nil, err
	}
	if pipelineDB.PipelineParams == nil {
		pipelineDB.PipelineParams = &implanttypes.PipelineParams{}
	}
	if req.CertName != "" {
		_, err := db.FindCertificate(req.CertName)
		if err != nil {
			return nil, err
		}
		pipelineDB, err = db.UpdatePipelineCert(req.CertName, pipelineDB)
		if err != nil {
			return nil, err
		}
	} else if req.Pipeline != nil && req.Pipeline.Tls != nil {
		if req.Pipeline.Tls.Cert != nil && req.Pipeline.Tls.Cert.Cert != "" && req.Pipeline.Tls.Cert.Key != "" {
			pipelineDB.PipelineParams.Tls = implanttypes.FromTls(req.Pipeline.Tls)
		}
	}
	lns, err := core.Listeners.Get(listenerID)
	if err != nil {
		return nil, err
	}

	if existing := lns.GetPipeline(req.Name); existing != nil && existing.Enable {
		_ = db.EnablePipelineByListener(req.Name, listenerID)
		return &clientpb.Empty{}, nil
	}

	pipelineProto := pipelineDB.ToProtobuf()
	job := &core.Job{
		ID:       core.NextJobID(),
		Pipeline: pipelineProto,
		Name:     req.Name,
	}

	ctrlID := lns.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlPipelineStart,
		Job:  job.ToProtobuf()})

	status := lns.WaitCtrl(ctrlID)
	if status == nil || status.Status != consts.CtrlStatusSuccess {
		_ = db.DisablePipelineByListener(pipelineDB.Name, listenerID)
		if status != nil && status.Error != "" {
			return nil, fmt.Errorf("start pipeline %s failed: %s", req.Name, status.Error)
		}
		return nil, fmt.Errorf("start pipeline %s failed: unknown error", req.Name)
	}

	pipeline := pipelineDB.ToProtobuf()
	if err := db.EnablePipelineByListener(pipeline.Name, listenerID); err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) StopPipeline(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	listenerID := req.GetListenerId()
	if listenerID == "" && req.Pipeline != nil {
		listenerID = req.Pipeline.ListenerId
	}
	if listenerID == "" {
		return nil, fmt.Errorf("listener_id required")
	}

	lns, err := core.Listeners.Get(listenerID)
	if err != nil {
		return nil, err
	}

	var pipe *clientpb.Pipeline
	if existing := lns.GetPipeline(req.Name); existing != nil {
		pipe = existing
	} else {
		pipelineDB, err := db.FindPipelineByListener(req.Name, listenerID)
		if err != nil {
			return nil, err
		}
		pipe = pipelineDB.ToProtobuf()
	}

	job := &core.Job{
		ID:       core.NextJobID(),
		Name:     req.Name,
		Pipeline: pipe,
	}
	lns.RemovePipeline(pipe)
	lns.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlPipelineStop,
		Job:  job.ToProtobuf(),
	})
	err = db.DisablePipelineByListener(req.Name, listenerID)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) DeletePipeline(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	listenerID := req.GetListenerId()
	if listenerID == "" && req.Pipeline != nil {
		listenerID = req.Pipeline.ListenerId
	}
	if listenerID == "" {
		return nil, fmt.Errorf("listener_id required")
	}

	pipelineDB, err := db.FindPipelineByListener(req.Name, listenerID)
	if err != nil {
		return nil, err
	}
	pipeline := pipelineDB.ToProtobuf()
	lns, err := core.Listeners.Get(listenerID)
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
	err = db.DeletePipelineByListener(req.Name, listenerID)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}
