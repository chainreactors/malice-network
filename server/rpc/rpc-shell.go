package rpc

import (
	"context"
	"sync"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
)

var implantPTYManagers sync.Map

func getImplantPTYManager(implantID string) *ImplantPTYManager {
	if v, ok := implantPTYManagers.Load(implantID); ok {
		return v.(*ImplantPTYManager)
	}
	mgr := NewImplantPTYManager()
	actual, _ := implantPTYManagers.LoadOrStore(implantID, mgr)
	return actual.(*ImplantPTYManager)
}

func (rpc *Server) PtyRequest(ctx context.Context, req *implantpb.PtyRequest) (*clientpb.Task, error) {
	switch req.GetType() {
	case consts.ModulePtyStart:
		greq, err := newGenericRequest(ctx, req)
		if err != nil {
			return nil, err
		}
		return rpc.handlePtyStart(ctx, greq, req)
	case consts.ModulePtyStop:
		return rpc.handlePtyStop(ctx, req)
	default:
		return rpc.handlePtyCommand(ctx, req)
	}
}

func (rpc *Server) handlePtyStart(ctx context.Context, greq *GenericRequest, req *implantpb.PtyRequest) (*clientpb.Task, error) {
	if req.Params == nil {
		req.Params = make(map[string]string)
	}
	req.Params["streaming"] = "true"

	greq.Count = -1
	in, out, err := rpc.StreamGenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	mgr := getImplantPTYManager(greq.Session.ID)
	mgr.Register(req.SessionId, in, greq)

	runTaskHandler(greq.Task, func() error {
		mgr.PumpOutput(req.SessionId, greq, out)
		return nil
	}, in.Close, func() {
		greq.Task.Close()
		mgr.Remove(req.SessionId)
		logs.Log.Debugf("[pty] cleaned up session %s:%s", greq.Session.ID, req.SessionId)
	})

	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) handlePtyStop(ctx context.Context, req *implantpb.PtyRequest) (*clientpb.Task, error) {
	session, err := getSession(ctx)
	if err != nil {
		return nil, err
	}

	mgr := getImplantPTYManager(session.ID)
	if taskPb, ok := mgr.GetTaskProto(req.SessionId); ok && mgr.SendCommand(req.SessionId, req) {
		mgr.Remove(req.SessionId)
		return taskPb, nil
	}

	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	greq.HandlerResponse(ch, types.MsgPtyResponse)
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) handlePtyCommand(ctx context.Context, req *implantpb.PtyRequest) (*clientpb.Task, error) {
	session, err := getSession(ctx)
	if err != nil {
		return nil, err
	}

	mgr := getImplantPTYManager(session.ID)
	if taskPb, ok := mgr.GetTaskProto(req.SessionId); ok && mgr.SendCommand(req.SessionId, req) {
		return taskPb, nil
	}

	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	greq.HandlerResponse(ch, types.MsgPtyResponse)
	return greq.Task.ToProtobuf(), nil
}
