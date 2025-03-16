package rpc

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/samber/lo"
)

func (rpc *Server) RegisterRem(ctx context.Context, req *clientpb.Pipeline) (*clientpb.Empty, error) {
	lns, err := core.Listeners.Get(req.ListenerId)
	if err != nil {
		return nil, err
	}
	req.Ip = lns.IP
	_, err = db.SavePipeline(models.FromPipelinePb(req))
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) ListRems(ctx context.Context, req *clientpb.Listener) (*clientpb.Pipelines, error) {
	var result []*clientpb.Pipeline
	ctxs, err := db.NewContextQuery().ByType(consts.ContextPivoting).Find()
	if err != nil {
		return nil, err
	}
	ctxMap := lo.GroupBy(ctxs, func(item *models.Context) string {
		return item.PipelineID
	})
	for pid, pivots := range ctxMap {
		pipe, ok := core.Listeners.Find(pid)
		if !ok {
			continue
		}
		pipe.GetRem().Agents = make(map[string]*clientpb.REMAgent)
		for _, c := range pivots {
			piv := c.Context.(*output.PivotingContext)
			pipe.GetRem().Agents[piv.RemID] = piv.ToRemAgent()
		}
		result = append(result, pipe)
	}
	return &clientpb.Pipelines{Pipelines: result}, nil
}

func (rpc *Server) StartRem(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	remDB, err := db.FindPipeline(req.Name)
	if err != nil {
		return nil, err
	}
	err = db.EnablePipeline(remDB.Name)
	if err != nil {
		return nil, err
	}
	rem := remDB.ToProtobuf()
	lns, err := core.Listeners.Get(remDB.ListenerId)
	if err != nil {
		return nil, err
	}
	job := &core.Job{
		ID:       core.NextJobID(),
		Pipeline: rem,
		Name:     rem.Name,
	}
	core.Jobs.Add(job)
	lns.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlRemStart,
		Job:  job.ToProtobuf(),
	})

	return &clientpb.Empty{}, nil
}

func (rpc *Server) StopRem(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	job, err := core.Jobs.Get(req.Name)
	if err != nil {
		return nil, err
	}
	ok := core.Listeners.RemovePipeline(job.Pipeline)
	if !ok {
		return nil, errs.ErrNotFoundListener
	}
	core.Listeners.PushCtrl(consts.CtrlRemStop, job.Pipeline)
	err = db.DisablePipeline(job.Name)
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
	lns, err := core.Listeners.Get(req.ListenerId)
	if err != nil {
		return nil, err
	}
	lns.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlRemStop,
		Job: &clientpb.Job{
			Pipeline: rem,
		},
	})
	err = db.DeletePipeline(req.Name)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) RemDial(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	pid := req.Params["pipeline_id"]
	if pid == "" {
		return nil, errs.ErrNotFoundPipeline
	}
	req.Params = nil
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go greq.HandlerResponse(ch, types.MsgResponse, func(spite *implantpb.Spite) {
		pipe, ok := core.Listeners.Find(pid)
		if !ok {
			logs.Log.Warnf("pipeline %s not found", pid)
			return
		}
		lns, _ := core.Listeners.Get(pipe.ListenerId)
		i := lns.PushCtrl(&clientpb.JobCtrl{
			Ctrl: consts.CtrlPipelineSync,
			Job: &clientpb.Job{
				Name:     pipe.Name,
				Pipeline: pipe,
			},
		})
		lns.WaitCtrl(i)

		event := core.Event{
			EventType: consts.EventPivot,
			Session:   greq.Session.ToProtobufLite(),
			Important: true,
			Spite:     spite,
		}
		pipe, ok = core.Listeners.Find(pid)
		if !ok {
			logs.Log.Warnf("pipeline %s not found", pid)
			return
		}

		if remOpt, ok := pipe.GetRem().Agents[spite.GetResponse().Output]; ok {
			pivot := output.NewPivotingWithRem(remOpt)
			pivot.Pipeline = pipe.Name
			pivot.Listener = pipe.ListenerId
			event.Op = "pivot_" + pivot.Mod
			event.Message = pivot.Abstract()
			lns, err := core.Listeners.Get(pipe.ListenerId)
			if err != nil {
				return
			}
			_, err = db.SaveContext(&clientpb.Context{
				Session:  greq.Session.ToProtobufLite(),
				Task:     greq.Task.ToProtobuf(),
				Listener: lns.ToProtobuf(),
				Pipeline: pipe,
				Type:     consts.ContextPivoting,
				Value:    output.MarshalContext(pivot),
			})
			if err != nil {
				return
			}
			core.EventBroker.Publish(event)
		}
	})
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) RemAgentCtrl(ctx context.Context, req *clientpb.REMAgent) (*clientpb.Empty, error) {
	pipe, ok := core.Listeners.Find(req.PipelineId)
	if !ok {
		return nil, errs.ErrNotFoundListener
	}
	lns, err := core.Listeners.Get(pipe.ListenerId)
	if err != nil {
		return nil, err
	}
	i := lns.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlRemAgentCtrl,
		Job: &clientpb.Job{
			Name:     pipe.Name,
			Pipeline: pipe,
			Body: &clientpb.Job_RemAgent{
				RemAgent: req,
			},
		},
	})
	status := lns.WaitCtrl(i)
	if status.Status == consts.CtrlStatusSuccess {
		agent := status.Job.GetRemAgent()
		pivot := output.NewPivotingWithRem(agent)
		pivot.Pipeline = pipe.Name
		pivot.Listener = pipe.ListenerId
		_, err = db.SaveContext(&clientpb.Context{
			Listener: lns.ToProtobuf(),
			Pipeline: pipe,
			Type:     consts.ContextPivoting,
			Value:    output.MarshalContext(pivot),
		})
		if err != nil {
			return nil, err
		}
		pipe.GetRem().Agents[agent.Id] = agent
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventPivot,
			Op:        "pivot_" + pivot.Mod,
			Message:   pivot.Abstract(),
		})
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) RemAgentLog(ctx context.Context, req *clientpb.REMAgent) (*clientpb.RemLog, error) {
	pipe, ok := core.Listeners.Find(req.PipelineId)
	if !ok {
		return nil, errs.ErrNotFoundListener
	}
	lns, err := core.Listeners.Get(pipe.ListenerId)
	if err != nil {
		return nil, err
	}
	i := lns.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlRemAgentLog,
		Job: &clientpb.Job{
			Name:     pipe.Name,
			Pipeline: pipe,
			Body: &clientpb.Job_RemAgent{
				RemAgent: req,
			},
		},
	})

	job := lns.WaitCtrl(i)
	return job.Job.GetRemLog(), nil
}

func (rpc *Server) RemAgentStop(ctx context.Context, req *clientpb.REMAgent) (*clientpb.Empty, error) {
	pipe, ok := core.Listeners.Find(req.PipelineId)
	if !ok {
		return nil, errs.ErrNotFoundListener
	}
	lns, err := core.Listeners.Get(pipe.ListenerId)
	if err != nil {
		return nil, err
	}
	lns.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlRemAgentStop,
		Job: &clientpb.Job{
			Name:     pipe.Name,
			Pipeline: pipe,
			Body: &clientpb.Job_RemAgent{
				RemAgent: req,
			},
		},
	})
	return &clientpb.Empty{}, nil
}

func (rpc *Server) LoadRem(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
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

func (rpc *Server) HealthCheckRem(ctx context.Context, req *clientpb.Pipeline) (*clientpb.Empty, error) {
	_, err := db.SavePipeline(models.FromPipelinePb(req))
	if err != nil {
		return nil, err
	}

	ctxs, err := db.NewContextQuery().ByType(consts.ContextPivoting).ByPipeline(req.Name).Find()
	if err != nil {
		return nil, err
	}

	agents := req.GetRem().Agents
	for _, c := range ctxs {
		if _, ok := agents[c.ID.String()]; !ok {
			t, err := output.ParseContext(consts.ContextPivoting, c.Value)
			if err != nil {
				return nil, err
			}
			piv := t.(*output.PivotingContext)
			if piv.Enable {
				piv.Enable = false
				c.Value = piv.Marshal()
				_, err = db.SaveContext(c.ToProtobuf())
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return &clientpb.Empty{}, nil
}
