package rpc

import (
	"context"
	"encoding/json"
	"github.com/chainreactors/malice-network/server/internal/db"
	"math"
	"strings"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/server/internal/core"
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
		var ctx output.Context
		var err error
		switch typ {
		case "gogo":
			ctx, err = output.ParseGOGO(content)
		}
		var value []byte
		value, err = json.Marshal(ctx)
		if err != nil {
			logs.Log.Error(err)
			return
		}

		_, err = db.SaveContext(&clientpb.Context{
			Task:    task.ToProtobuf(),
			Session: task.Session.ToProtobufLite(),
			Type:    ctx.Type(),
			Value:   value,
			Nonce:   nonce,
		})
		if err != nil {
			logs.Log.Error(err)
			return
		}
	}
}

func (rpc *Server) Execute(ctx context.Context, req *implantpb.ExecRequest) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go greq.HandlerResponse(ch, types.MsgExec)
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
		var results strings.Builder
		for _, bofResp := range bofResps.(output.BOFResponses) {
			switch bofResp.CallbackType {
			case output.CALLBACK_OUTPUT, output.CALLBACK_OUTPUT_OEM, output.CALLBACK_OUTPUT_UTF8:
				continue
			case output.CALLBACK_ERROR:
				continue
			case output.CALLBACK_SCREENSHOT:
				if bofResp.Length <= 4 {
					results.WriteString("Null screenshot data\n")
					continue
				}
				err = core.HandleScreenshot(bofResp.Data, greq.Task)
			case output.CALLBACK_FILE:
				err = core.HandleFileOperations("open", bofResp.Data, greq.Task)
			case output.CALLBACK_FILE_WRITE:
				err = core.HandleFileOperations("write", bofResp.Data, greq.Task)
			case output.CALLBACK_FILE_CLOSE:
				err = core.HandleFileOperations("close", bofResp.Data, greq.Task)
			default:
				logs.Log.Errorf("Unimplemented callback type : %d", bofResp.CallbackType)
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

	go greq.HandlerResponse(ch, types.MsgBinaryResponse)
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
