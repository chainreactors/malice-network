package rpc

import (
	"context"
	"fmt"
	"strings"

	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"

	"github.com/chainreactors/malice-network/server/internal/core"
)

func (rpc *Server) RegisterPipeline(ctx context.Context, req *clientpb.Pipeline) (*clientpb.Empty, error) {
	ip := getRemoteAddr(ctx)
	ip = strings.Split(ip, ":")[0]
	pipelineModel := models.FromPipelinePb(req, ip)
	var err error
	if pipelineModel.Tls.Enable && pipelineModel.Tls.Cert == "" && pipelineModel.Tls.Key == "" {
		pipelineModel.Tls.Cert, pipelineModel.Tls.Key, err = certutils.GenerateTlsCert(pipelineModel.Name, pipelineModel.ListenerID)
		if err != nil {
			return nil, err
		}
	}
	err = db.CreatePipeline(pipelineModel)
	if err != nil {
		return nil, err
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
		result = append(result, models.ToPipelinePB(pipeline))
	}
	return &clientpb.Pipelines{Pipelines: result}, nil
}

func (rpc *Server) StartPipeline(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	pipelineDB, err := db.FindPipeline(req.Name)
	if err != nil {
		return nil, err
	}
	pipelineDB.Enable = true
	pipeline := models.ToPipelinePB(pipelineDB)
	listener := core.Listeners.Get(pipeline.ListenerId)
	if listener == nil {
		return nil, fmt.Errorf("listener %s not found", req.ListenerId)
	}
	listener.AddPipeline(pipeline)
	core.Jobs.Add(&core.Job{
		ID:      core.CurrentJobID(),
		Message: pipeline,
		Name:    pipeline.Name,
	})
	core.Jobs.Ctrl <- &clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlPipelineStart,
		Job: &clientpb.Job{
			Id:       core.NextJobID(),
			Pipeline: pipeline,
		},
	}
	err = db.EnablePipeline(pipelineDB)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) StopPipeline(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	pipelineDB, err := db.FindPipeline(req.Name)
	if err != nil {
		return &clientpb.Empty{}, err
	}
	pipeline := models.ToPipelinePB(pipelineDB)
	listener := core.Listeners.Get(pipeline.ListenerId)
	if listener == nil {
		return nil, fmt.Errorf("listener %s not found", req.ListenerId)
	}
	listener.RemovePipeline(pipeline)
	core.Jobs.Ctrl <- &clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlPipelineStop,
		Job: &clientpb.Job{
			Id:       core.NextJobID(),
			Pipeline: pipeline,
		},
	}
	err = db.DisablePipeline(pipelineDB)
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
	pipeline := models.ToPipelinePB(pipelineDB)
	listener := core.Listeners.Get(pipeline.ListenerId)
	if listener == nil {
		return nil, fmt.Errorf("listener %s not found", req.ListenerId)
	}
	listener.RemovePipeline(pipeline)
	core.Jobs.Ctrl <- &clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlPipelineStop,
		Job: &clientpb.Job{
			Id:       core.NextJobID(),
			Pipeline: pipeline,
		},
	}
	err = db.DeletePipeline(req.Name)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}
