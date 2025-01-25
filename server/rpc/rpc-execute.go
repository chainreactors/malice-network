package rpc

import (
	"bytes"
	"context"
	"encoding/binary"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/handlers"
	"io"
	"math"
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
		reader := bytes.NewReader(spite.GetBinaryResponse().GetData())
		var bofResps handlers.BOFResponses
		for {
			bofResp := &handlers.BOFResponse{}

			err := binary.Read(reader, binary.LittleEndian, &bofResp.OutputType)
			if err == io.EOF {
				break
			} else if err != nil {
				return
			}

			err = binary.Read(reader, binary.LittleEndian, &bofResp.CallbackType)
			if err == io.EOF {
				break
			} else if err != nil {
				return
			}

			err = binary.Read(reader, binary.LittleEndian, &bofResp.Length)
			if err == io.EOF {
				break
			} else if err != nil {
				return
			}

			strData := make([]byte, bofResp.Length)
			_, err = io.ReadFull(reader, strData)
			if err == io.EOF {
				break
			} else if err != nil {
				return
			}

			bofResp.Data = strData

			bofResps = append(bofResps, bofResp)
		}

		msg := bofResps.Handler(greq.Session.ToProtobuf(), greq.Task.ToProtobuf())
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventBof,
			Op:        consts.CtrlBof,
			Message:   msg,
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
