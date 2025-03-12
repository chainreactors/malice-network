package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/protobuf/proto"
	"sync"
	"time"
)

func (rpc *Server) GetListeners(ctx context.Context, req *clientpb.Empty) (*clientpb.Listeners, error) {
	return core.Listeners.ToProtobuf(), nil
}

func (rpc *Server) RegisterListener(ctx context.Context, req *clientpb.RegisterListener) (*clientpb.Empty, error) {
	//ip := getRemoteIp(ctx)
	core.Listeners.Add(&core.Listener{
		Name:      req.Name,
		IP:        req.Host,
		Active:    true,
		Pipelines: make(map[string]*clientpb.Pipeline),
		Ctrl:      make(chan *clientpb.JobCtrl),
		CtrlJob:   &sync.Map{},
	})
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
	pipelinesCh[pipelineID] = stream

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

		if size := proto.Size(msg.Spite); size <= 1000 {
			logs.Log.Debugf("[server.%s] receive spite %s from %s, %v", sess.ID, msg.Spite.Name, msg.ListenerId, msg.Spite)
		} else {
			logs.Log.Debugf("[server.%s] receive spite %s from %s, %d bytes", sess.ID, msg.Spite.Name, msg.ListenerId, size)
		}

		if ch, ok := sess.GetResp(msg.TaskId); ok {
			go func() {
				ch <- msg.Spite
			}()
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
	go func() {
		for {
			select {
			case msg := <-lns.Ctrl:
				lns.CtrlJob.Store(msg.Id, nil)
				err := stream.Send(msg)
				if err != nil {
					logs.Log.Errorf("send job ctrl faild %v", err)
					return
				}
			}
		}
	}()

	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		_, ok := lns.CtrlJob.Load(msg.CtrlId)
		if ok {
			lns.CtrlJob.Store(msg.CtrlId, msg.Job)
			go func() {
				time.Sleep(1 * time.Second)
				lns.CtrlJob.Delete(msg.CtrlId)
			}()
		}
		if msg.Ctrl == consts.CtrlPipelineSync {
			continue
		}
		if msg.Status == consts.CtrlStatusSuccess {
			core.EventBroker.Publish(core.Event{
				EventType: consts.EventJob,
				Op:        msg.Ctrl,
				IsNotify:  true,
				Job:       msg.Job,
				Important: true,
			})
		} else {
			core.EventBroker.Publish(core.Event{
				EventType: consts.EventJob,
				Op:        msg.Ctrl,
				Err:       fmt.Sprintf("%s faild,status %d,  %s", msg.Job.Name, msg.Status, msg.Error),
				IsNotify:  true,
				Important: true,
			})
		}

	}
}

func (rpc *Server) ListJobs(ctx context.Context, req *clientpb.Empty) (*clientpb.Pipelines, error) {
	var pipelines []*clientpb.Pipeline
	for _, job := range core.Jobs.All() {
		pipelines = append(pipelines, job.Pipeline)
	}
	return &clientpb.Pipelines{Pipelines: pipelines}, nil
}
