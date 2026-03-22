package rpc

import (
	"context"
	"fmt"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	types "github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/server/internal/parser"
)

func (rpc *Server) PipeClose(ctx context.Context, req *implantpb.PipeRequest) (*clientpb.Task, error) {
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

func (rpc *Server) PipeRead(ctx context.Context, req *implantpb.PipeRequest) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	greq.HandlerResponse(ch, types.MsgBinaryResponse, ContextCallback(greq.Task, ctx))
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) PipeServer(ctx context.Context, req *implantpb.PipeRequest) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, types.MsgResponse)
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) PipeUpload(ctx context.Context, pipe *implantpb.PipeRequest) (*clientpb.Task, error) {
	if pipe == nil || pipe.Pipe == nil {
		return nil, types.ErrMissingRequestField
	}
	req := pipe.Pipe
	count := parser.Count(req.Data, getPacketLength(ctx))
	if count == 1 {
		greq, err := newGenericRequest(ctx, pipe)
		if err != nil {
			return nil, err
		}
		ch, err := rpc.GenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}
		greq.HandlerResponse(ch, types.MsgAck)
		return greq.Task.ToProtobuf(), nil
	} else {
		greq, err := newGenericRequest(ctx, &implantpb.PipeRequest{
			Type: consts.ModulePipeUpload,
			Pipe: &implantpb.Pipe{
				Name:   req.Name,
				Target: req.Target,
			},
		}, count)
		if err != nil {
			return nil, err
		}
		in, out, err := rpc.StreamGenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}
		var blockId = 0
		runTaskHandler(greq.Task, func() error {
			stat, ok := recvSpite(greq.Task.Ctx, out)
			if !ok {
				return ErrTaskContextCancelled
			}
			err := types.HandleMaleficError(stat)
			if err != nil {
				return buildTaskError(err)
			}
			for block := range parser.Chunked(req.Data, greq.Session.GetPacketLength()) {
				msg := &implantpb.Block{
					BlockId: uint32(blockId),
					Content: block,
				}
				blockId++
				if blockId == count {
					msg.End = true
				}
				spite, _ := types.BuildSpite(&implantpb.Spite{
					Timeout: uint64(consts.MinTimeout.Seconds()),
					TaskId:  greq.Task.Id,
				}, msg)
				spite.Name = types.MsgUpload.String()
				if err := in.Send(spite); err != nil {
					return err
				}
				resp, ok := recvSpite(greq.Task.Ctx, out)
				if !ok {
					return ErrTaskContextCancelled
				}
				err = types.AssertSpite(resp, types.MsgAck)
				if err != nil {
					return buildTaskError(err)
				}
				greq.Session.AddMessage(resp, blockId)

				err = greq.Session.TaskLog(greq.Task, resp)
				if err != nil {
					return fmt.Errorf("write task log: %w", err)
				}
				if resp.GetAck().Success {
					greq.Task.Done(resp, "")
					if msg.End {
						greq.Task.Finish(resp, "")
					}
				}
			}
			return nil
		}, greq.Task.Close, in.Close)
		return greq.Task.ToProtobuf(), nil
	}
}
