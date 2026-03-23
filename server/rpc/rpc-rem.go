package rpc

import (
	"context"
	"errors"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
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

	_, err = db.FindPipelineByListener(req.Name, req.ListenerId)
	if err != nil {
		if req.GetRem().Console == "" {
			req.GetRem().Console = "tcp://127.0.0.1:12345"
		}
		_, err = db.SavePipeline(models.FromPipelinePb(req))
		if err != nil {
			return nil, err
		}
	}

	return &clientpb.Empty{}, nil
}

func (rpc *Server) ListRems(ctx context.Context, req *clientpb.Listener) (*clientpb.Pipelines, error) {
	var result []*clientpb.Pipeline
	ctxs, err := db.NewContextQuery().WhereType(consts.ContextPivoting).Find()
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
			pipe.GetRem().Agents[piv.RemAgentID] = piv.ToRemAgent()
		}
		result = append(result, pipe)
	}
	return &clientpb.Pipelines{Pipelines: result}, nil
}

func (rpc *Server) StartRem(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	listenerID, err := resolveListenerID(req)
	if err != nil {
		return nil, err
	}

	remDB, err := db.FindPipelineByListener(req.Name, listenerID)
	if err != nil {
		return nil, err
	}
	lns, err := core.Listeners.Get(listenerID)
	if err != nil {
		return nil, err
	}

	if existing := lns.GetPipeline(req.Name); existing != nil && existing.Enable {
		_ = db.EnablePipelineByListener(req.Name, listenerID)
		return &clientpb.Empty{}, nil
	}

	rem := remDB.ToProtobuf()
	job := &core.Job{
		ID:       core.NextJobID(),
		Pipeline: rem,
		Name:     rem.Name,
	}

	ctrlID := lns.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlRemStart,
		Job:  job.ToProtobuf(),
	})

	status := lns.WaitCtrl(ctrlID)
	if err := waitForCtrlStatus("start rem", req.Name, status); err != nil {
		_ = db.DisablePipelineByListener(remDB.Name, listenerID)
		return nil, err
	}

	if err := db.EnablePipelineByListener(rem.Name, listenerID); err != nil {
		return nil, err
	}
	// Do not call core.Jobs.AddPipeline(rem) here: the listener's
	// handleStartRem already invoked SyncPipeline with the runtime-
	// populated pipeline (Link, Subscribe, Port). Calling AddPipeline
	// again with the stale DB snapshot would overwrite those values.

	return &clientpb.Empty{}, nil
}

func (rpc *Server) DeleteRem(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	listenerID, err := resolveListenerID(req)
	if err != nil {
		return nil, err
	}

	if _, err := db.FindPipelineByListener(req.Name, listenerID); err != nil {
		return &clientpb.Empty{}, err
	}
	lns, err := core.Listeners.Get(listenerID)
	if err != nil {
		return nil, err
	}

	if existing := lns.GetPipeline(req.Name); existing != nil {
		ctrlID := lns.PushCtrl(&clientpb.JobCtrl{
			Ctrl: consts.CtrlRemStop,
			Job: &clientpb.Job{
				Id:       core.NextJobID(),
				Name:     req.Name,
				Pipeline: existing,
			},
		})
		status := lns.WaitCtrl(ctrlID)
		if err := waitForCtrlStatus("delete rem", req.Name, status); err != nil {
			return nil, err
		}
		lns.RemovePipeline(existing)
	}

	err = db.DeletePipelineByListener(req.Name, listenerID)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) StopRem(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	listenerID, err := resolveListenerID(req)
	if err != nil {
		return nil, err
	}

	lns, err := core.Listeners.Get(listenerID)
	if err != nil {
		return nil, err
	}

	if _, err := db.FindPipelineByListener(req.Name, listenerID); err != nil {
		return nil, err
	}

	pipe := lns.GetPipeline(req.Name)
	if pipe != nil {
		job := &core.Job{
			ID:       core.NextJobID(),
			Name:     req.Name,
			Pipeline: pipe,
		}

		ctrlID := lns.PushCtrl(&clientpb.JobCtrl{
			Ctrl: consts.CtrlRemStop,
			Job:  job.ToProtobuf(),
		})
		status := lns.WaitCtrl(ctrlID)
		if err := waitForCtrlStatus("stop rem", req.Name, status); err != nil {
			return nil, err
		}
	}

	if err := db.DisablePipelineByListener(req.Name, listenerID); err != nil {
		return nil, err
	}
	if pipe != nil {
		lns.RemovePipeline(pipe)
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) RemDial(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	err := types.AssertRequestName(req, consts.ModuleRemDial)
	if err != nil {
		return nil, err
	}
	pid := req.Params["pipeline_id"]
	if pid == "" {
		return nil, types.ErrNotFoundPipeline
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

	greq.HandlerResponse(ch, types.MsgResponse, func(spite *implantpb.Spite) {
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
			pivot := output.NewPivotingWithRem(remOpt, pipe)
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
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	pipe, ok := core.Listeners.Find(req.PipelineId)
	if !ok {
		return nil, types.ErrNotFoundListener
	}
	lns, err := core.Listeners.Get(pipe.ListenerId)
	if err != nil {
		return nil, err
	}

	// Route reconfigure requests to dedicated handler (no pivot side-effects).
	if len(req.Args) > 0 && req.Args[0] == "reconfigure" {
		lns.PushCtrl(&clientpb.JobCtrl{
			Ctrl: consts.CtrlRemAgentReconfigure,
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
	if err := waitForCtrlStatus("rem agent ctrl", req.Id, status); err != nil {
		return nil, err
	}
	agent := status.GetJob().GetRemAgent()
	if agent == nil {
		return nil, errors.New("rem agent ctrl response missing agent")
	}
	pivot := output.NewPivotingWithRem(agent, pipe)
	_, err = db.SaveContext(&clientpb.Context{
		Listener: lns.ToProtobuf(),
		Pipeline: pipe,
		Type:     consts.ContextPivoting,
		Value:    output.MarshalContext(pivot),
	})
	if err != nil {
		return nil, err
	}
	if pipe.GetRem().Agents == nil {
		pipe.GetRem().Agents = make(map[string]*clientpb.REMAgent)
	}
	pipe.GetRem().Agents[agent.Id] = agent
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventPivot,
		Op:        "pivot_" + pivot.Mod,
		Message:   pivot.Abstract(),
	})
	return &clientpb.Empty{}, nil
}

func (rpc *Server) RemAgentLog(ctx context.Context, req *clientpb.REMAgent) (*clientpb.RemLog, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	pipe, ok := core.Listeners.Find(req.PipelineId)
	if !ok {
		return nil, types.ErrNotFoundListener
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

	status := lns.WaitCtrl(i)
	if err := waitForCtrlStatus("rem agent log", req.Id, status); err != nil {
		return nil, err
	}
	if status.GetJob() == nil || status.GetJob().GetRemLog() == nil {
		return nil, errors.New("rem agent log response missing log")
	}
	return status.GetJob().GetRemLog(), nil
}

func (rpc *Server) RemAgentStop(ctx context.Context, req *clientpb.REMAgent) (*clientpb.Empty, error) {
	pipe, ok := core.Listeners.Find(req.PipelineId)
	if !ok {
		return nil, types.ErrNotFoundListener
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

func (rpc *Server) HealthCheckRem(ctx context.Context, req *clientpb.Pipeline) (*clientpb.Empty, error) {
	_, err := db.SavePipeline(models.FromPipelinePb(req))
	if err != nil {
		return nil, err
	}

	ctxs, err := db.NewContextQuery().WhereType(consts.ContextPivoting).WherePipeline(req.Name).Find()
	if err != nil {
		return nil, err
	}

	agents := req.GetRem().Agents

	// Build a set of agent IDs already tracked in DB for reverse lookup.
	knownAgents := make(map[string]struct{}, len(ctxs))
	for _, c := range ctxs {
		piv := c.Context.(*output.PivotingContext)
		knownAgents[piv.RemAgentID] = struct{}{}
		if _, ok := agents[piv.RemAgentID]; !ok && piv.Enable {
			piv.Enable = false
			c.Value = piv.Marshal()
			_, err = db.SaveContext(c.ToProtobuf())
			if err != nil {
				return nil, err
			}
		} else if ok && !piv.Enable {
			piv.Enable = true
			c.Value = piv.Marshal()
			_, err = db.SaveContext(c.ToProtobuf())
			if err != nil {
				return nil, err
			}
		}
	}

	// Create PivotingContext for agents present in memory but missing from DB
	// (e.g. agents accepted via acceptLoop that were never explicitly dialed/forked).
	for id, agent := range agents {
		if _, exists := knownAgents[id]; exists {
			continue
		}
		pivot := output.NewPivotingWithRem(agent, req)
		_, err = db.SaveContext(&clientpb.Context{
			Pipeline: req,
			Type:     consts.ContextPivoting,
			Value:    output.MarshalContext(pivot),
		})
		if err != nil {
			return nil, err
		}
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventPivot,
			Op:        "pivot_" + pivot.Mod,
			Message:   pivot.Abstract(),
			Important: true,
		})
	}

	return &clientpb.Empty{}, nil
}
