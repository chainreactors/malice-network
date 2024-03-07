package rpc

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/client/rootpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/configs"

	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/internal/certs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/proto"
)

func (rpc *Server) GetListeners(ctx context.Context, req *clientpb.Empty) (*clientpb.Listeners, error) {
	return core.Listeners.ToProtobuf(), nil
}

func (rpc *Server) RegisterListener(ctx context.Context, req *lispb.RegisterListener) (*implantpb.Empty, error) {
	core.Listeners.Add(&core.Listener{
		Name:   req.Name,
		Host:   req.Addr,
		Active: true,
	})
	p, ok := peer.FromContext(ctx)
	if !ok {
		return &implantpb.Empty{}, nil
	}
	logs.Log.Importantf("%s register listener %s", p.Addr, req.Name)
	return &implantpb.Empty{}, nil
}

func (rpc *Server) SpiteStream(stream listenerrpc.ListenerRPC_SpiteStreamServer) error {
	listenerID, err := getListenerID(stream.Context())
	if err != nil {
		logs.Log.Error(err.Error())
		return err
	}
	listenersCh[listenerID] = stream
	dbSession := db.Session()
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		sess, ok := core.Sessions.Get(msg.SessionId)

		err = models.UpdateLast(dbSession, sess.ID)
		if err != nil {
			logs.Log.Error(err.Error())
		}
		if !ok {
			return ErrNotFoundSession
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
	_, _, err := certs.ClientGenerateCertificate(cfg.GRPCHost, req.Name, int(cfg.GRPCPort), certs.ListenerCA)
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

func (s *Server) RemoveListener(ctx context.Context, req *rootpb.Operator) (*rootpb.Response, error) {
	err := mtls.RemoveConfig(req.Name, certs.ListenerCA)
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
	files, err := mtls.GetListeners()
	if err != nil {
		return nil, err
	}
	listeners := &clientpb.Listeners{}
	for _, file := range files {
		listeners.Listeners = append(listeners.Listeners, &clientpb.Listener{
			Id: file,
		})
	}

	return listeners, nil
}
