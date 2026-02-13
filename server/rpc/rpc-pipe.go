package rpc

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/handler"
	"github.com/chainreactors/malice-network/server/internal/parser"
	"github.com/gookit/config/v2"
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

	go greq.HandlerResponse(ch, types.MsgEmpty)
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

	go greq.HandlerResponse(ch, types.MsgResponse)
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) PipeUpload(ctx context.Context, pipe *implantpb.PipeRequest) (*clientpb.Task, error) {
	req := pipe.Pipe
	count := parser.Count(req.Data, config.Int(consts.ConfigMaxPacketLength))
	if count == 1 {
		greq, err := newGenericRequest(ctx, pipe)
		if err != nil {
			return nil, err
		}
		ch, err := rpc.GenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}
		go greq.HandlerResponse(ch, types.MsgAck)
		return greq.Task.ToProtobuf(), nil
	} else {
		greq, err := newGenericRequest(ctx, &implantpb.PipeRequest{
			Type: consts.ModulePipeUpload,
			Pipe: &implantpb.Pipe{
				Name:   req.Name,
				Target: req.Target,
			},
		}, count)
		in, out, err := rpc.StreamGenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}
		var blockId = 0
		go func() {
			stat := <-out
			err := handler.HandleMaleficError(stat)
			if err != nil {
				greq.Task.Panic(buildErrorEvent(greq.Task, err))
				return
			}
			for block := range parser.Chunked(req.Data, config.Int(consts.ConfigMaxPacketLength)) {
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
				in <- spite
				resp := <-out
				err = handler.AssertSpite(resp, types.MsgAck)
				if err != nil {
					return
				}
				greq.Session.AddMessage(resp, blockId)

				err = greq.Session.TaskLog(greq.Task, resp)
				if err != nil {
					logs.Log.Errorf("Failed to write task log: %v", err)
					return
				}
				if resp.GetAck().Success {
					greq.Task.Done(resp, "")
					if msg.End {
						greq.Task.Finish(resp, "")
					}
				}
			}
			close(in)
		}()
		return greq.Task.ToProtobuf(), nil
	}
}
