package rpc

import (
	"context"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func applyModulesResponse(sess *core.Session, spite *implantpb.Spite, appendOnly bool) {
	if sess == nil || spite == nil {
		return
	}
	modules := spite.GetModules()
	if modules == nil {
		return
	}
	if appendOnly {
		sess.Modules = append(sess.Modules, modules.Modules...)
	} else {
		sess.Modules = modules.Modules
	}
	sess.SaveAndNotify("")
}

func (rpc *Server) ListModule(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	return rpc.AssertAndHandleWithSession(ctx, req, consts.ModuleListModule, types.MsgListModule, func(greq *GenericRequest, spite *implantpb.Spite) {
		applyModulesResponse(greq.Session, spite, false)
	})
}

func (rpc *Server) LoadModule(ctx context.Context, req *implantpb.LoadModule) (*clientpb.Task, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	return rpc.GenericInternalWithSession(ctx, req, types.MsgListModule, func(greq *GenericRequest, spite *implantpb.Spite) {
		applyModulesResponse(greq.Session, spite, true)
	})
}

func (rpc *Server) RefreshModule(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	return rpc.AssertAndHandleWithSession(ctx, req, consts.ModuleRefreshModule, types.MsgListModule, func(greq *GenericRequest, spite *implantpb.Spite) {
		applyModulesResponse(greq.Session, spite, false)
	})
}

func (rpc *Server) Clear(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandle(ctx, req, consts.ModuleClear, types.MsgEmpty)
}

// ExecuteModule passthrough for fully dynamic module execution.
// For streaming modules (e.g. tapping/llm.observe), it uses a continuous
// loop that keeps the task alive instead of finishing after one response.
func (rpc *Server) ExecuteModule(ctx context.Context, req *implantpb.ExecuteModuleRequest) (*clientpb.Task, error) {
	if req == nil || req.Spite == nil {
		return nil, types.ErrMissingRequestField
	}

	expect := types.MsgName(req.Expect)

	// Streaming module: keep reading from the channel until context is cancelled.
	if req.Spite.Name == "tapping" || req.Spite.Name == "poison" || req.Spite.Name == "agent" {
		greq, err := newGenericRequest(ctx, req.Spite)
		if err != nil {
			return nil, err
		}
		greq.Count = -1 // streaming mode, no auto-finish
		out, err := rpc.GenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}

		runTaskHandler(greq.Task, func() error {
			for {
				resp, ok := recvSpite(greq.Task.Ctx, out)
				if !ok {
					return ErrTaskContextCancelled
				}
				if resp == nil {
					return nil
				}
				err := types.AssertSpite(resp, expect)
				if err != nil {
					logs.Log.Warnf("ExecuteModule: unexpected message type, assert failed: %v", err)
					continue
				}
				if err := greq.HandlerSpite(resp); err != nil {
					return err
				}
			}
		}, greq.Task.Close)

		return greq.Task.ToProtobuf(), nil
	}

	return rpc.GenericInternal(ctx, req.Spite, expect)
}
