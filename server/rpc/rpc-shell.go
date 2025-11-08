package rpc

import (
	"context"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"strings"
)

// Shell 统一的PTY shell处理方法
func (rpc *Server) PtyRequest(ctx context.Context, req *implantpb.PtyRequest) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// 获取操作类型
	action := req.GetType()

	switch action {
	case consts.ModulePtyStart:
		// 启动shell会话，使用流式处理支持实时输出
		greq.Count = -1
		_, out, err := rpc.StreamGenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}

		go func() {
			for {
				resp := <-out
				err := types.AssertSpite(resp, types.MsgPtyResponse)
				if err != nil {
					greq.Task.Panic(buildErrorEvent(greq.Task, err))
					return
				}
				err = greq.HandlerSpite(resp)
				if err != nil {
					return
				}
				greq.Task.Finish(resp, "")

				// 检查是否会话结束
				moduleResp := resp.GetResponse()
				if moduleResp != nil && moduleResp.GetError() != "" &&
					(strings.Contains(moduleResp.GetError(), "session") &&
						strings.Contains(moduleResp.GetError(), "closed")) {
					greq.Task.Finish(resp, "Shell session ended")
					break
				}
			}
		}()

	default:
		// 默认处理
		ch, err := rpc.GenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}
		go greq.HandlerResponse(ch, types.MsgPtyResponse)
	}

	return greq.Task.ToProtobuf(), nil
}
