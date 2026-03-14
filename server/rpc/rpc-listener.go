package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/protobuf/proto"
	"io"
	"time"
)

func (rpc *Server) GetListeners(ctx context.Context, req *clientpb.Empty) (*clientpb.Listeners, error) {
	return core.Listeners.ToProtobuf(), nil
}

func (rpc *Server) RegisterListener(ctx context.Context, req *clientpb.RegisterListener) (*clientpb.Empty, error) {
	//ip := getRemoteIp(ctx)
	core.Listeners.Add(core.NewListener(req.Name, req.Host))
	core.EventBroker.Notify(core.Event{
		EventType: consts.EventListener,
		Op:        consts.CtrlListenerStart,
		Message:   fmt.Sprintf("Listener %s started at %s", req.Name, req.Host),
	})
	logs.Log.Importantf("[server] %s register listener: %s", req.Host, req.Name)
	return &clientpb.Empty{}, nil
}

func (rpc *Server) SpiteStream(stream listenerrpc.ListenerRPC_SpiteStreamServer) error {
	pipelineID, err := getPipelineID(stream.Context())
	if err != nil {
		logs.Log.Error(err.Error())
		return err
	}
	pipelinesCh.Store(pipelineID, stream)

	for {
		msg, err := stream.Recv()
		if err != nil {
			logs.Log.Error("pipeline stream exit!")
			return err
		}

		sess, err := core.Sessions.Get(msg.SessionId)
		if err != nil {
			logs.Log.Warnf("session %s not found", msg.SessionId)
			continue
		}
		sess.SetLastCheckin(time.Now().Unix())
		if sess.MarkAlive() {
			if err := sess.Save(); err != nil {
				logs.Log.Errorf("save session %s reborn state failed: %s", sess.ID, err.Error())
			}
			sess.Publish(consts.CtrlSessionReborn, fmt.Sprintf("session %s from %s reborn at %s", sess.Abstract(), sess.Target, sess.PipelineID), true, true)
		}

		if size := proto.Size(msg.Spite); size <= 1000 {
			logs.Log.Debugf("[server.%s] receive spite %s from %s, %v", sess.ID, msg.Spite.Name, msg.ListenerId, msg.Spite)
		} else {
			logs.Log.Debugf("[server.%s] receive spite %s from %s, %d bytes", sess.ID, msg.Spite.Name, msg.ListenerId, size)
		}

		ch, ok := sess.GetResp(msg.TaskId)
		if !ok {
			logs.Log.Warnf("response channel missing for session %s task %d", msg.SessionId, msg.TaskId)
			continue
		}
		if err := deliverSpiteResponse(ch, msg.Spite); err != nil {
			logs.Log.Warnf("deliver spite response failed for session %s task %d: %v", msg.SessionId, msg.TaskId, err)
		}
	}
}

//func (s *Server) ListenerCtrl(req *clientpb.CtrlStatus, stream listenerrpc.ListenerRPC_ListenerCtrlServer) error {
//	var resp clientpb.CtrlPipeline
//	for {
//		if req.CtrlType == consts.CtrlPipelineStart {
//
//		} else if req.CtrlType == consts.CtrlPipelineStop {
//			err := core.Listeners.Stop(req.ListenerID)
//			if err != nil {
//				logs.Log.Error(err.Error())
//			}
//			resp = clientpb.CtrlPipeline{
//				ListenerID: req.ListenerID,
//				CtrlType:     consts.CtrlPipelineStop,
//			}
//		}
//		if err := stream.Send(&resp); err != nil {
//			return err
//		}
//	}
//}

func (rpc *Server) JobStream(stream listenerrpc.ListenerRPC_JobStreamServer) error {
	listenerID, err := getListenerID(stream.Context())
	if err != nil {
		return err
	}
	lns, err := core.Listeners.Get(listenerID)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	recvMsgCh := make(chan *clientpb.JobStatus)
	sendErrCh := core.GoGuarded("listener-job-stream-send:"+listenerID, func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case msg, ok := <-lns.Ctrl:
				if !ok {
					return nil
				}
				lns.CtrlJob.Store(msg.Id, nil)
				if err := stream.Send(msg); err != nil {
					lns.CtrlJob.Delete(msg.Id)
					return fmt.Errorf("send job ctrl failed: %w", err)
				}
			}
		}
	}, core.LogGuardedError("listener-job-stream-send:"+listenerID))
	recvErrCh := core.GoGuarded("listener-job-stream-recv:"+listenerID, func() error {
		defer close(recvMsgCh)
		for {
			msg, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
			select {
			case <-ctx.Done():
				return nil
			case recvMsgCh <- msg:
			}
		}
	}, core.LogGuardedError("listener-job-stream-recv:"+listenerID))

	var pendingRecvCh <-chan *clientpb.JobStatus = recvMsgCh
	var pendingSendErrCh <-chan error = sendErrCh
	var pendingRecvErrCh <-chan error = recvErrCh

	for pendingRecvCh != nil || pendingSendErrCh != nil || pendingRecvErrCh != nil {
		select {
		case msg, ok := <-pendingRecvCh:
			if !ok {
				cancel()
				pendingRecvCh = nil
				continue
			}
			handleJobStatus(lns, msg)
		case err, ok := <-pendingSendErrCh:
			pendingSendErrCh = nil
			cancel()
			if ok && err != nil {
				return err
			}
		case err, ok := <-pendingRecvErrCh:
			pendingRecvErrCh = nil
			cancel()
			if ok && err != nil {
				return err
			}
		}
	}

	return nil
}

func handleJobStatus(lns *core.Listener, msg *clientpb.JobStatus) {
	if _, ok := lns.CtrlJob.Load(msg.CtrlId); ok {
		lns.CtrlJob.Store(msg.CtrlId, msg)
		core.GoGuarded("listener-job-status-cleanup", func() error {
			time.Sleep(1 * time.Second)
			lns.CtrlJob.Delete(msg.CtrlId)
			return nil
		}, core.LogGuardedError("listener-job-status-cleanup"))
	}
	if msg.Ctrl == consts.CtrlPipelineSync {
		return
	}
	if msg.Status == consts.CtrlStatusSuccess {
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventJob,
			Op:        msg.Ctrl,
			IsNotify:  true,
			Job:       msg.Job,
			Important: true,
		})
		return
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventJob,
		Op:        msg.Ctrl,
		Err:       fmt.Sprintf("%s faild,status %d,  %s", msg.Job.Name, msg.Status, msg.Error),
		IsNotify:  true,
		Important: true,
	})
}

func deliverSpiteResponse(ch chan *implantpb.Spite, spite *implantpb.Spite) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = core.RecoverError("listener-spite-response", recovered)
		}
	}()

	select {
	case ch <- spite:
		return nil
	default:
		return fmt.Errorf("response channel full")
	}
}

func (rpc *Server) ListJobs(ctx context.Context, req *clientpb.Empty) (*clientpb.Pipelines, error) {
	var pipelines []*clientpb.Pipeline
	for _, job := range core.Jobs.All() {
		pipelines = append(pipelines, job.Pipeline)
	}
	return &clientpb.Pipelines{Pipelines: pipelines}, nil
}
