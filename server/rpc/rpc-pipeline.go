package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"

	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func (rpc *Server) RegisterPipeline(ctx context.Context, req *lispb.Pipeline) (*implantpb.Empty, error) {
	ch := make(chan bool)

	job := &core.Job{
		ID:      core.NextJobID(),
		Message: req,
		JobCtrl: ch,
	}
	core.Jobs.Add(job)
	return &implantpb.Empty{}, nil
}

func (rpc *Server) StartTcpPipeline(ctx context.Context, req *lispb.TCPPipeline) (*clientpb.Empty, error) {
	ctrl := clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlPipelineStart,
		Job: &clientpb.Job{
			Id:       core.NextJobID(),
			Pipeline: types.BuildPipeline(req),
		},
	}
	core.Jobs.Ctrl <- &ctrl

	return &clientpb.Empty{}, nil
}
