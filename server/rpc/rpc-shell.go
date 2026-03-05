package rpc

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/core"
)

type streamingPtySession struct {
	inCh      chan *implantpb.Spite
	task      *core.Task
	sessionID string
}

var ptyStreamingSessions sync.Map

func ptySessionKey(implantSessionID, ptySessionID string) string {
	return implantSessionID + ":" + ptySessionID
}

func (rpc *Server) PtyRequest(ctx context.Context, req *implantpb.PtyRequest) (*clientpb.Task, error) {
	action := req.GetType()

	switch action {
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

	key := ptySessionKey(greq.Session.ID, req.SessionId)
	ptyStreamingSessions.Store(key, &streamingPtySession{
		inCh:      in,
		task:      greq.Task,
		sessionID: req.SessionId,
	})

	core.SafeGoWithTask(greq.Task, func() {
		for {
			resp := <-out
			if resp == nil {
				break
			}

			ptyResp := resp.GetPtyResponse()

			err := greq.HandlerSpite(resp)
			if err != nil {
				return
			}
			greq.Task.Finish(resp, "")

			if ptyResp != nil && !ptyResp.SessionActive {
				greq.Task.Finish(resp, "Shell session ended")
				break
			}

			moduleResp := resp.GetResponse()
			if moduleResp != nil && moduleResp.GetError() != "" &&
				(strings.Contains(moduleResp.GetError(), "session") &&
					strings.Contains(moduleResp.GetError(), "closed")) {
				greq.Task.Finish(resp, "Shell session ended")
				break
			}
		}
	}, func() {
		greq.Task.Close()
		ptyStreamingSessions.Delete(key)
		logs.Log.Debugf("[pty-streaming] cleaned up session %s", key)
	})

	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) handlePtyStop(ctx context.Context, req *implantpb.PtyRequest) (*clientpb.Task, error) {
	session, err := getSession(ctx)
	if err != nil {
		return nil, err
	}

	key := ptySessionKey(session.ID, req.SessionId)
	if val, ok := ptyStreamingSessions.Load(key); ok {
		sess := val.(*streamingPtySession)
		spite := &implantpb.Spite{
			Name:   types.MsgPty.String(),
			Body:   &implantpb.Spite_PtyRequest{PtyRequest: req},
			TaskId: sess.task.Id,
		}
		sess.inCh <- spite
		ptyStreamingSessions.Delete(key)
		return sess.task.ToProtobuf(), nil
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

	key := ptySessionKey(session.ID, req.SessionId)
	if val, ok := ptyStreamingSessions.Load(key); ok {
		sess := val.(*streamingPtySession)
		spite := &implantpb.Spite{
			Name:   types.MsgPty.String(),
			Body:   &implantpb.Spite_PtyRequest{PtyRequest: req},
			TaskId: sess.task.Id,
		}
		sess.inCh <- spite
		return sess.task.ToProtobuf(), nil
	}

	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	greq.HandlerResponse(ch, types.MsgPtyResponse)
	return greq.Task.ToProtobuf(), nil
}
