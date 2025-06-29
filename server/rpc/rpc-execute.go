package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chainreactors/malice-network/helper/utils/handler"
	"math"
	"strings"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
)

//var (
//	argueMap = map[string]string{}
//)

func handleBinary(binary *implantpb.ExecuteBinary) *implantpb.ExecuteBinary {
	if binary.ProcessName == "" {
		binary.ProcessName = `C:\\Windows\\System32\\notepad.exe`
	}
	if binary.Timeout == 0 {
		binary.Timeout = math.MaxUint32
	}
	if len(binary.Args) == 0 {
		binary.Args = []string{""}
	}
	binary.Timeout = binary.Timeout * 1000
	return binary
}

func ContextCallback(task *core.Task, ctx context.Context) func(*implantpb.Spite) {
	typ, nonce := getContextNonce(ctx)
	if typ == "" || nonce == "" {
		return func(spite *implantpb.Spite) {
			return
		}
	}
	return func(spite *implantpb.Spite) {
		content := spite.GetBinaryResponse().GetData()
		if content == nil {
			logs.Log.Error("Empty content")
			return
		}
		var ctxs output.Contexts
		switch typ {
		case output.GOGOPortType:
			c, err := output.ParseGOGO(content)
			if err != nil {
				logs.Log.Error(err)
				return
			}
			ctxs = append(ctxs, c)
		case "zombie":
			cs, err := output.ParseZombie(content)
			if err != nil {
				logs.Log.Error(err)
				return
			}
			for _, c := range cs {
				ctxs = append(ctxs, c)
			}
		case "mimikatz":
			cs, err := output.ParseMimikatz(content)
			//fmt.Println(string(content))
			//fmt.Printf("cs: %v", cs)
			if err != nil {
				logs.Log.Error(err)
				return
			}
			for _, c := range cs {
				ctxs = append(ctxs, c)
			}
		}

		for _, c := range ctxs {
			value, err := json.Marshal(c)
			if err != nil {
				logs.Log.Error(err)
				return
			}

			model, err := db.SaveContext(&clientpb.Context{
				Task:    task.ToProtobuf(),
				Session: task.Session.ToProtobufLite(),
				Type:    c.Type(),
				Value:   value,
				Nonce:   nonce,
			})
			if err != nil {
				logs.Log.Error(err)
				return
			}

			core.EventBroker.Publish(core.Event{
				EventType: consts.EventContext,
				Op:        c.Type(),
				Task:      task.ToProtobuf(),
				Message:   fmt.Sprintf("new %s context: %s", c.Type(), model.ID),
			})
		}
	}
}

func (rpc *Server) Execute(ctx context.Context, req *implantpb.ExecRequest) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	if !req.Realtime {
		ch, err := rpc.GenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}

		go greq.HandlerResponse(ch, types.MsgExec)
	} else {
		greq.Count = -1
		_, out, err := rpc.StreamGenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}

		go func() {
			for {
				resp := <-out
				exec := resp.GetExecResponse()
				err := handler.AssertSpite(resp, types.MsgExec)
				if err != nil {
					greq.Task.Panic(buildErrorEvent(greq.Task, err))
					return
				}
				err = greq.HandlerSpite(resp)
				if err != nil {
					return
				}
				if exec.End {
					greq.Task.Finish(resp, "")
					break
				}
			}
		}()
	}

	return greq.Task.ToProtobuf(), nil

}

func (rpc *Server) ExecuteAssembly(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	go greq.HandlerResponse(ch, types.MsgBinaryResponse)
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) ExecuteShellcode(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	req = handleBinary(req)
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	go greq.HandlerResponse(ch, types.MsgBinaryResponse, ContextCallback(greq.Task, ctx))
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) ExecuteBof(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	go greq.HandlerResponse(ch, types.MsgBinaryResponse, func(spite *implantpb.Spite) {
		tctx := greq.TaskContext(spite)
		bofResps, err := output.ParseBOFResponse(tctx)
		if err != nil {
			logs.Log.Error(err)
			return
		}

		// handler context bof callback
		var results strings.Builder
		for _, bofResp := range bofResps.(output.BOFResponses) {
			switch bofResp.CallbackType {
			case output.CallbackScreenshot:
				if bofResp.Length <= 4 {
					results.WriteString("Null screenshot data\n")
					continue
				}
				err = core.HandleScreenshot(bofResp.Data, greq.Task)
			case output.CallbackFile:
				err = core.HandleFileOperations("open", bofResp.Data, greq.Task)
			case output.CallbackFileWrite:
				err = core.HandleFileOperations("write", bofResp.Data, greq.Task)
			case output.CallbackFileClose:
				err = core.HandleFileOperations("close", bofResp.Data, greq.Task)
			default:
				continue
			}
		}
	})

	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) ExecuteEXE(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	req = handleBinary(req)
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go greq.HandlerResponse(ch, types.MsgBinaryResponse, ContextCallback(greq.Task, ctx))

	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) ExecuteDll(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	req = handleBinary(req)
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	go greq.HandlerResponse(ch, types.MsgBinaryResponse)
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) ExecutePowerpick(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	go greq.HandlerResponse(ch, types.MsgBinaryResponse)
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) ExecuteArmory(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	req = handleBinary(req)
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	go greq.HandlerResponse(ch, types.MsgBinaryResponse)
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) ExecuteLocal(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	req = handleBinary(req)
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	go greq.HandlerResponse(ch, types.MsgBinaryResponse)
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) InlineLocal(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	req = handleBinary(req)
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	go greq.HandlerResponse(ch, types.MsgBinaryResponse)
	return greq.Task.ToProtobuf(), nil
}
