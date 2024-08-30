package rpc

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/client/rootpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/internal/certs"
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

func (rpc *Server) RegisterListener(ctx context.Context, req *lispb.RegisterListener) (*implantpb.Empty, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return &implantpb.Empty{}, nil
	}
	core.Listeners.Add(&core.Listener{
		Name:   req.Name,
		Host:   p.Addr.String(),
		Active: true,
	})
	logs.Log.Importantf("%s register listener %s", p.Addr, req.Name)
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

func (s *Server) AddListener(ctx context.Context, req *rootpb.Operator) (*rootpb.Response, error) {
	cfg := configs.GetServerConfig()
	clientConf, err := certs.GenerateListenerCert(cfg.GRPCHost, req.Args[0], int(cfg.GRPCPort))
	if err != nil {
		return &rootpb.Response{
			Status: 1,
			Error:  err.Error(),
		}, err
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

func (s *Server) RemoveListener(ctx context.Context, req *rootpb.Operator) (*rootpb.Response, error) {
	err := certs.RemoveCertificate(certs.ListenerCA, certs.RSAKey, req.Args[0])
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

func (s *Server) ListListeners(ctx context.Context, req *rootpb.Operator) (*clientpb.Listeners, error) {
	dbListeners, err := db.ListListeners()
	if err != nil {
		return nil, err
	}
	listeners := &clientpb.Listeners{}
	for _, listener := range dbListeners {
		listeners.Listeners = append(listeners.Listeners, &clientpb.Listener{
			Id: listener.Name,
		})
	}

	return listeners, nil
}

//func (s *Server) ListenerCtrl(req *lispb.CtrlStatus, stream listenerrpc.ListenerRPC_ListenerCtrlServer) error {
//	var resp lispb.CtrlPipeline
//	for {
//		if req.CtrlType == consts.CtrlPipelineStart {
//
//		} else if req.CtrlType == consts.CtrlPipelineStop {
//			err := core.Listeners.Stop(req.ListenerName)
//			if err != nil {
//				logs.Log.Error(err.Error())
//			}
//			resp = lispb.CtrlPipeline{
//				ListenerName: req.ListenerName,
//				CtrlType:     consts.CtrlPipelineStop,
//			}
//		}
//		if err := stream.Send(&resp); err != nil {
//			return err
//		}
//	}
//}
