package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
	"strings"

	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func (rpc *Server) RegisterRem(ctx context.Context, req *clientpb.Pipeline) (*clientpb.Empty, error) {
	ip := getRemoteAddr(ctx)
	ip = strings.Split(ip, ":")[0]
	remModel := models.FromPipelinePb(req, ip)

	err := db.SavePipeline(remModel)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) ListRems(ctx context.Context, req *clientpb.Listener) (*clientpb.Pipelines, error) {
	var result []*clientpb.Pipeline
	rems, err := db.ListPipelines(req.Id)
	if err != nil {
		return nil, err
	}
	for _, rem := range rems {
		if rem.Type == consts.RemPipeline {
			result = append(result, rem.ToProtobuf())
		}
	}
	return &clientpb.Pipelines{Pipelines: result}, nil
}

func (rpc *Server) StartRem(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	remDB, err := db.FindPipeline(req.Name)
	if err != nil {
		return nil, err
	}
	remDB.Enable = true
	rem := remDB.ToProtobuf()
	ok := core.Listeners.AddPipeline(rem)
	if !ok {
		return nil, errs.ErrNotFoundListener
	}
	core.Jobs.Add(&core.Job{
		ID:      core.CurrentJobID(),
		Message: rem,
		Name:    rem.Name,
	})
	core.Jobs.Ctrl <- &clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlRemStart,
		Job: &clientpb.Job{
			Id:       core.NextJobID(),
			Pipeline: rem,
		},
	}
	err = db.EnablePipeline(remDB)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) StopRem(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	remDB, err := db.FindPipeline(req.Name)
	if err != nil {
		return &clientpb.Empty{}, err
	}
	rem := remDB.ToProtobuf()
	ok := core.Listeners.RemovePipeline(rem)
	if !ok {
		return nil, errs.ErrNotFoundListener
	}
	core.Jobs.Ctrl <- &clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlRemStop,
		Job: &clientpb.Job{
			Id:       core.NextJobID(),
			Pipeline: rem,
		},
	}
	err = db.DisablePipeline(remDB)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) DeleteRem(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	remDB, err := db.FindPipeline(req.Name)
	if err != nil {
		return &clientpb.Empty{}, err
	}
	rem := remDB.ToProtobuf()
	ok := core.Listeners.RemovePipeline(rem)
	if !ok {
		return nil, errs.ErrNotFoundListener
	}
	core.Jobs.Ctrl <- &clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlRemStop,
		Job: &clientpb.Job{
			Id:       core.NextJobID(),
			Pipeline: rem,
		},
	}
	err = db.DeletePipeline(req.Name)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) NewRemCallback(ctx context.Context, req *clientpb.Pipeline) (*clientpb.Empty, error) {
	err := db.SavePipeline(models.FromPipelinePb(req, ""))
	if err != nil {
		return nil, err
	}
	ok := core.Listeners.UpdatePipeline(req)
	if !ok {
		return nil, errs.ErrNotFoundListener
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) RemDial(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go greq.HandlerResponse(ch, types.MsgResponse)
	return greq.Task.ToProtobuf(), nil
}
