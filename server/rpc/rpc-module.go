package rpc

import (
	"context"
	"errors"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func handlerModule(sess *core.Session) func(spite *implantpb.Spite) {
	return func(spite *implantpb.Spite) {
		if modules := spite.GetModules(); modules != nil {
			sess.Modules = modules.Modules
		}
		sess.SaveAndNotify("")
	}
}

func (rpc *Server) ListModule(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	err := types.AssertRequestName(req, consts.ModuleListModule)
	if err != nil {
		return nil, err
	}
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, types.MsgListModule, handlerModule(greq.Session))
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) LoadModule(ctx context.Context, req *implantpb.LoadModule) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, types.MsgListModule, func(spite *implantpb.Spite) {
		greq.Session.Modules = append(greq.Session.Modules, spite.GetModules().Modules...)
		greq.Session.SaveAndNotify("")
	})
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) RefreshModule(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	err := types.AssertRequestName(req, consts.ModuleRefreshModule)
	if err != nil {
		return nil, err
	}
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, types.MsgListModule, handlerModule(greq.Session))
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) Clear(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	err := types.AssertRequestName(req, consts.ModuleClear)
	if err != nil {
		return nil, err
	}
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, types.MsgEmpty)
	return greq.Task.ToProtobuf(), nil
}

// ExecuteModule passthrough for fully dynamic module execution.
// For streaming modules (e.g. tapping/llm.observe), it uses a continuous
// loop that keeps the task alive instead of finishing after one response.
func (rpc *Server) ExecuteModule(ctx context.Context, req *implantpb.ExecuteModuleRequest) (*clientpb.Task, error) {
	if req == nil || req.Spite == nil {
		return nil, errors.New("spite required")
	}

	expect := types.MsgName(req.Expect)

	// Streaming module: keep reading from the channel until context is cancelled.
	if req.Spite.Name == "tapping" || req.Spite.Name == "poison" {
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
					continue
				}
				if err := greq.HandlerSpite(resp); err != nil {
					return err
				}
			}
		}, greq.Task.Close)

		return greq.Task.ToProtobuf(), nil
	}

	return Handler(ctx, rpc, req.Spite, expect)
}
