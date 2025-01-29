package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/server/internal/core"
	"math"
	"strings"
)

var (
	argueMap = map[string]string{}
)

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
	go greq.HandlerResponse(ch, types.MsgBinaryResponse)
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
			var result string
			var err error
			switch bofResp.CallbackType {
			case output.CALLBACK_OUTPUT, output.CALLBACK_OUTPUT_OEM, output.CALLBACK_OUTPUT_UTF8:
				results.WriteString(string(bofResp.Data))
				continue
			case output.CALLBACK_ERROR:
				results.WriteString(fmt.Sprintf("Error occurred: %s\n", string(bofResp.Data)))
				continue
			case output.CALLBACK_SCREENSHOT:
				if bofResp.Length-4 <= 0 {
					results.WriteString("Null screenshot data\n")
					continue
				}
				result, err = core.HandleScreenshot(bofResp.Data, greq.Task)
				if err != nil {
					result = fmt.Sprintf("Screenshot error: %v", err)
				}
			case output.CALLBACK_FILE:
				result, err = core.HandleFileOperations("open", bofResp.Data, greq.Task)
				if err != nil {
					result = fmt.Sprintf("File open error: %v", err)
				}
			case output.CALLBACK_FILE_WRITE:
				result, err = core.HandleFileOperations("write", bofResp.Data, greq.Task)
				if err != nil {
					result = fmt.Sprintf("File write error: %v", err)
				}
			case output.CALLBACK_FILE_CLOSE:
				result, err = core.HandleFileOperations("close", bofResp.Data, greq.Task)
				if err != nil {
					result = fmt.Sprintf("File close error: %v", err)
				}
			default:
				result = fmt.Sprintf("Unimplemented callback type : %d", bofResp.CallbackType)
			}
			results.WriteString(result + "\n")
		}

		core.EventBroker.Publish(core.Event{
			EventType: consts.EventBof,
			Op:        consts.CtrlBof,
			Message:   results.String(),
			IsNotify:  true,
		})
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
