package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"

	"github.com/chainreactors/malice-network/server/internal/core"
)

func (rpc *Server) RegisterPipeline(ctx context.Context, req *clientpb.Pipeline) (*implantpb.Empty, error) {
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

func (rpc *Server) ListTcpPipelines(ctx context.Context, req *clientpb.ListenerName) (*clientpb.Pipelines, error) {
	var result []*clientpb.Pipeline
	pipelines, err := db.ListPipelines(req.Name, "tcp")
	if err != nil {
		return &clientpb.Pipelines{}, err
	}
	for _, pipeline := range pipelines {
		tcp := clientpb.TCPPipeline{
			Name:   pipeline.Name,
			Host:   pipeline.Host,
			Port:   uint32(pipeline.Port),
			Enable: pipeline.Enable,
		}
		result = append(result, &clientpb.Pipeline{
			Body: &clientpb.Pipeline_Tcp{
				Tcp: &tcp,
			},
		})
	}
	return &clientpb.Pipelines{Pipelines: result}, nil
}

func (rpc *Server) StartTcpPipeline(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	pipelineDB, err := db.FindPipeline(req.Name, req.ListenerId)
	if err != nil {
		return &clientpb.Empty{}, err
	}
	pipeline := models.ToProtobuf(&pipelineDB)
	listener := core.Listeners.Get(req.ListenerId)
	listener.Pipelines.Pipelines = append(listener.Pipelines.Pipelines, pipeline)
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

func (rpc *Server) StopTcpPipeline(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	pipelineDB, err := db.FindPipeline(req.Name, req.ListenerId)
	if err != nil {
		return &clientpb.Empty{}, err
	}
	pipeline := models.ToProtobuf(&pipelineDB)
	listener := core.Listeners.Get(req.ListenerId)
	for i, p := range listener.Pipelines.Pipelines {
		if p.GetTcp().Name == req.Name {
			listener.Pipelines.Pipelines = append(listener.Pipelines.Pipelines[:i], listener.Pipelines.Pipelines[i+1:]...)
		}
	}
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

func (rpc *Server) ListJobs(ctx context.Context, req *clientpb.Empty) (*clientpb.Pipelines, error) {
	var pipelines []*clientpb.Pipeline
	for _, job := range core.Jobs.All() {
		pipeline, ok := job.Message.(*clientpb.Pipeline)
		if !ok {
			continue
		}
		if pipeline.GetTcp() != nil {
			pipelines = append(pipelines, job.Message.(*clientpb.Pipeline))
		}
	}
	return &clientpb.Pipelines{Pipelines: pipelines}, nil
}
