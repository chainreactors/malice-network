package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"

	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func (rpc *Server) RegisterPipeline(ctx context.Context, req *lispb.Pipeline) (*implantpb.Empty, error) {
	err := db.CreatePipeline(req)
	if err != nil {
		return &implantpb.Empty{}, err
	}
	return &implantpb.Empty{}, nil
}

func (rpc *Server) StartTcpPipeline(ctx context.Context, req *lispb.CtrlPipeline) (*clientpb.Empty, error) {
	pipelineDB, err := db.FindPipeline(req.Name, req.ListenerId)
	if err != nil {
		return &clientpb.Empty{}, err
	}
	pipeline := models.ToProtobuf(&pipelineDB)
	ch := make(chan bool)
	job := &core.Job{
		ID:      core.CurrentJobID(),
		Message: pipeline,
		JobCtrl: ch,
	}
	core.Jobs.Add(job)
	ctrl := clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlPipelineStart,
		Job: &clientpb.Job{
			Id:       core.NextJobID(),
			Pipeline: pipeline,
		},
	}
	core.Jobs.Ctrl <- &ctrl
	return &clientpb.Empty{}, nil
}

func (rpc *Server) StopTcpPipeline(ctx context.Context, req *lispb.CtrlPipeline) (*clientpb.Empty, error) {
	pipelineDB, err := db.FindPipeline(req.Name, req.ListenerId)
	if err != nil {
		return &clientpb.Empty{}, err
	}
	pipeline := models.ToProtobuf(&pipelineDB)
	ctrl := clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlPipelineStop,
		Job: &clientpb.Job{
			Id:       core.NextJobID(),
			Pipeline: pipeline,
		},
	}
	core.Jobs.Ctrl <- &ctrl
	return &clientpb.Empty{}, nil
}

func (rpc *Server) ListPipelines(ctx context.Context, req *lispb.ListenerName) (*lispb.Pipelines, error) {
	var pipelines []*lispb.Pipeline
	for _, job := range core.Jobs.All() {
		pipeline, ok := job.Message.(*lispb.Pipeline)
		if !ok {
			continue
		}
		if pipeline.GetTcp() != nil {
			pipelines = append(pipelines, job.Message.(*lispb.Pipeline))
		}
	}
	return &lispb.Pipelines{Pipelines: pipelines}, nil
}
