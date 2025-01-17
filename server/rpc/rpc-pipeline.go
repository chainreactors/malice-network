package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
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
	pipelineModel := models.FromPipelinePb(req)
	if pipelineModel.Tls.Enable && pipelineModel.Tls.Cert == "" && pipelineModel.Tls.Key == "" {
		pipelineModel.Tls.Cert, pipelineModel.Tls.Key, err = certutils.GenerateTlsCert(pipelineModel.Name, pipelineModel.ListenerID)
		if err != nil {
			return nil, err
		}
	}
	_, err = db.SavePipeline(pipelineModel)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) SyncPipeline(ctx context.Context, req *clientpb.Pipeline) (*clientpb.Empty, error) {
	pipe, err := db.SavePipeline(models.FromPipelinePb(req))
	if err != nil {
		return nil, err
	}
	ok := core.Listeners.UpdatePipeline(pipe.ToProtobuf())
	if !ok {
		return nil, errs.ErrNotFoundListener
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) ListPipelines(ctx context.Context, req *clientpb.Listener) (*clientpb.Pipelines, error) {
	var result []*clientpb.Pipeline
	pipelines, err := db.ListPipelines(req.Id)
	if err != nil {
		return nil, err
	}
	for _, pipeline := range pipelines {
		if pipeline.Type == consts.TCPPipeline || pipeline.Type == consts.BindPipeline {
			result = append(result, pipeline.ToProtobuf())
		}
	}
	return &clientpb.Pipelines{Pipelines: result}, nil
}

func (rpc *Server) StartPipeline(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	pipelineDB, err := db.FindPipeline(req.Name)
	if err != nil {
		return nil, err
	}
	pipeline := pipelineDB.ToProtobuf()
	pipeline.Target = req.Target
	pipeline.BeaconPipeline = req.BeaconPipeline
	lns, err := core.Listeners.Get(req.ListenerId)
	if err != nil {
		return nil, err
	}
	lns.AddPipeline(pipeline)
	job := &core.Job{
		ID:       core.NextJobID(),
		Pipeline: pipeline,
		Name:     pipeline.Name,
	}
	core.Jobs.Add(job)
	lns.PushCtrl(&clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlPipelineStart,
		Job:  job.ToProtobuf()})
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
	lns, err := core.Listeners.Get(req.ListenerId)
	if err != nil {
		return nil, err
	}
	lns.RemovePipeline(job.Pipeline)
	lns.PushCtrl(&clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
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
		return &clientpb.Empty{}, err
	}
	pipeline := pipelineDB.ToProtobuf()
	lns, err := core.Listeners.Get(req.ListenerId)
	if err != nil {
		return nil, err
	}
	lns.RemovePipeline(pipeline)
	lns.PushCtrl(&clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
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
