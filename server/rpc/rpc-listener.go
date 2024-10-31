package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/client/rootpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

func (rpc *Server) GetListeners(ctx context.Context, req *clientpb.Empty) (*clientpb.Listeners, error) {
	return core.Listeners.ToProtobuf(), nil
}

func (rpc *Server) RegisterListener(ctx context.Context, req *clientpb.RegisterListener) (*implantpb.Empty, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get peer from context")
	}
	core.Listeners.Add(&core.Listener{
		Name:      req.Name,
		Host:      p.Addr.String(),
		Active:    true,
		Pipelines: make(map[string]*clientpb.Pipeline),
	})
	core.EventBroker.Notify(core.Event{
		EventType: consts.EventListener,
		Op:        consts.CtrlListenerStart,
		Message:   fmt.Sprintf("Listener %s started at %s", req.Name, p.Addr.String()),
	})
	logs.Log.Importantf("[server] %s register listener: %s", p.Addr, req.Name)
	return &implantpb.Empty{}, nil
}

func (rpc *Server) SpiteStream(stream listenerrpc.ListenerRPC_SpiteStreamServer) error {
	listenerID, err := getPipelineID(stream.Context())
	if err != nil {
		logs.Log.Error(err.Error())
		return err
	}
	pipelinesCh[listenerID] = stream
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}

		sess, ok := core.Sessions.Get(msg.SessionId)
		if !ok {
			logs.Log.Warnf("session %s not found", msg.SessionId)
			continue
		}

		if size := proto.Size(msg.Spite); size <= 1000 {
			logs.Log.Debugf("[server.%s] receive spite %s from %s, %v", sess.ID, msg.Spite.Name, msg.ListenerId, msg.Spite)
		} else {
			logs.Log.Debugf("[server.%s] receive spite %s from %s, %d bytes", sess.ID, msg.Spite.Name, msg.ListenerId, size)
		}

		if ch, ok := sess.GetResp(msg.TaskId); ok {
			ch <- msg.Spite
		}
	}
}

func (rpc *Server) AddListener(ctx context.Context, req *rootpb.Operator) (*rootpb.Response, error) {
	cfg := configs.GetServerConfig()
	clientConf, err := certutils.GenerateListenerCert(cfg.IP, req.Args[0], int(cfg.GRPCPort))
	if err != nil {
		return &rootpb.Response{
			Status: 1,
			Error:  err.Error(),
		}, err
	}
	err = db.CreateOperator(req.Args[0], mtls.Listener, getRemoteAddr(ctx))
	if err != nil {
		return nil, err
	}
	data, err := yaml.Marshal(clientConf)
	if err != nil {
		return &rootpb.Response{
			Status: 1,
			Error:  err.Error(),
		}, err
	}
	return &rootpb.Response{
		Status:   0,
		Response: string(data),
	}, nil
}

func (rpc *Server) RemoveListener(ctx context.Context, req *rootpb.Operator) (*rootpb.Response, error) {
	err := certutils.RemoveCertificate(certutils.ListenerCA, certutils.RSAKey, req.Args[0])
	if err != nil {
		return &rootpb.Response{
			Status: 1,
			Error:  err.Error(),
		}, err
	}
	return &rootpb.Response{
		Status:   0,
		Response: "",
	}, nil
}

func (rpc *Server) ListListeners(ctx context.Context, req *rootpb.Operator) (*clientpb.Listeners, error) {
	dbListeners, err := db.ListListeners()
	if err != nil {
		return nil, err
	}
	listeners := &clientpb.Listeners{}
	for _, listener := range dbListeners {
		listeners.Listeners = append(listeners.Listeners, &clientpb.Listener{
			Id:   listener.Name,
			Addr: listener.Remote,
		})
	}

	return listeners, nil
}

//func (s *Server) ListenerCtrl(req *clientpb.CtrlStatus, stream listenerrpc.ListenerRPC_ListenerCtrlServer) error {
//	var resp clientpb.CtrlPipeline
//	for {
//		if req.CtrlType == consts.CtrlPipelineStart {
//
//		} else if req.CtrlType == consts.CtrlPipelineStop {
//			err := core.Listeners.Stop(req.ListenerName)
//			if err != nil {
//				logs.Log.Error(err.Error())
//			}
//			resp = clientpb.CtrlPipeline{
//				ListenerName: req.ListenerName,
//				CtrlType:     consts.CtrlPipelineStop,
//			}
//		}
//		if err := stream.Send(&resp); err != nil {
//			return err
//		}
//	}
//}

func (rpc *Server) JobStream(stream listenerrpc.ListenerRPC_JobStreamServer) error {

	go func() {
		for {
			select {
			case msg := <-core.Jobs.Ctrl:
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
		if msg.Status == consts.CtrlStatusSuccess {
			if msg.Ctrl == consts.CtrlWebUpload {
				continue
			}
			core.EventBroker.Publish(core.Event{
				EventType: consts.EventJob,
				Op:        msg.Ctrl,
				IsNotify:  true,
				Job:       msg.Job,
			})
		} else {
			if msg.Ctrl == consts.CtrlWebUpload {
				core.EventBroker.Publish(core.Event{
					EventType: consts.EventWebsite,
					Op:        msg.Ctrl,
					Err:       fmt.Sprintf("status %d,  %s", msg.Status, msg.Error),
				})
				continue
			}
			core.EventBroker.Publish(core.Event{
				EventType: consts.EventJob,
				Op:        msg.Ctrl,
				Err:       fmt.Sprintf("%s faild,status %d,  %s", msg.Job.Name, msg.Status, msg.Error),
			})
		}
	}
}

func (rpc *Server) ListJobs(ctx context.Context, req *clientpb.Empty) (*clientpb.Pipelines, error) {
	var pipelines []*clientpb.Pipeline
	for _, job := range core.Jobs.All() {
		pipeline, ok := job.Message.(*clientpb.Pipeline)
		if !ok {
			continue
		}
		if pipeline.GetTcp() != nil {
			pipelines = append(pipelines, job.Message.(*clientpb.Pipeline))
		}
	}
	return &clientpb.Pipelines{Pipelines: pipelines}, nil
}
