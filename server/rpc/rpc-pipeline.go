package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"

	"github.com/chainreactors/malice-network/helper/proto/listener/lispb"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func (rpc *Server) RegisterPipeline(ctx context.Context, req *lispb.Pipeline) (*implantpb.Empty, error) {
	if req.GetTls().Enable && req.GetTls().Cert == "" && req.GetTls().Key == "" {
		cert, key, err := certutils.GenerateTlsCert(req.GetTcp().Name, req.GetTcp().ListenerId)
		if err != nil {
			return &implantpb.Empty{}, err
		}
		req.GetTls().Cert = cert
		req.GetTls().Key = key
	}
	err := db.CreatePipeline(req)
	if err != nil {
		return &implantpb.Empty{}, err
	}
	return &implantpb.Empty{}, nil
}

func (rpc *Server) ListTcpPipelines(ctx context.Context, req *lispb.ListenerName) (*lispb.Pipelines, error) {
	var result []*lispb.Pipeline
	pipelines, err := db.ListPipelines(req.Name, "tcp")
	if err != nil {
		return &lispb.Pipelines{}, err
	}
	for _, pipeline := range pipelines {
		tcp := lispb.TCPPipeline{
			Name:   pipeline.Name,
			Host:   pipeline.Host,
			Port:   uint32(pipeline.Port),
			Enable: pipeline.Enable,
		}
		result = append(result, &lispb.Pipeline{
			Body: &lispb.Pipeline_Tcp{
				Tcp: &tcp,
			},
		})
	}
	return &lispb.Pipelines{Pipelines: result}, nil
}

func (rpc *Server) StartTcpPipeline(ctx context.Context, req *lispb.CtrlPipeline) (*clientpb.Empty, error) {
	pipelineDB, err := db.FindPipeline(req.Name, req.ListenerId)
	if err != nil {
		return &clientpb.Empty{}, err
	}
	pipeline := models.ToProtobuf(&pipelineDB)
	pipeline.GetTcp().Enable = true
	job := &core.Job{
		ID:      core.CurrentJobID(),
		Message: pipeline,
		Name:    pipeline.GetTcp().Name,
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
	err = db.EnablePipeline(pipelineDB)
	if err != nil {
		return nil, err
	}
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
	err = db.UnEnablePipeline(pipelineDB)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) ListJobs(ctx context.Context, req *clientpb.Empty) (*lispb.Pipelines, error) {
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
