package rpc

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"strconv"
)

func (rpc *Server) Events(_ *clientpb.Empty, stream clientrpc.MaliceRPC_EventsServer) error {
	events := core.EventBroker.Subscribe()
	clientID := core.GetCurrentID()
	defer func() {
		logs.Log.Infof("client: %d disconnected", clientID)
		core.Clients.Remove(int(clientID))
		core.EventBroker.Unsubscribe(events)
	}()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case event := <-events:
			err := stream.Send(event.ToProtobuf())
			if err != nil {
				logs.Log.Warnf(err.Error())
				return err
			}
		}
	}
}

func (rpc *Server) Broadcast(ctx context.Context, req *clientpb.Event) (*clientpb.Empty, error) {
	core.EventBroker.Publish(core.Event{
		EventType: req.Type,
		Op:        req.Op,
		Client:    req.Client,
		Err:       req.Err,
		Message:   string(req.Message),
		Important: true,
	})
	return &clientpb.Empty{}, nil
}

func (rpc *Server) Notify(ctx context.Context, req *clientpb.Event) (*clientpb.Empty, error) {
	core.EventBroker.Notify(core.Event{
		EventType: req.Type,
		Op:        req.Op,
		Message:   string(req.Message),
		Client:    req.Client,
		IsNotify:  true,
		Err:       req.Err,
	})
	return &clientpb.Empty{}, nil
}

func (rpc *Server) SessionEvent(ctx context.Context, req *clientpb.Event) (*clientpb.Empty, error) {
	core.EventBroker.Publish(core.Event{
		Session:   req.Session,
		Task:      req.Task,
		Client:    req.Client,
		EventType: req.Type,
		Op:        req.Op,
		Err:       req.Err,
		Message:   string(req.Message),
	})
	if req.Op == consts.CtrlSessionTask {
		taskId := strconv.FormatUint(uint64(req.Task.TaskId), 10)
		id := req.Session.SessionId + "-" + taskId
		err := db.UpdateTaskDescription(id, string(req.Message))
		if err != nil {
			return nil, err
		}
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) GetEvent(ctx context.Context, req *clientpb.Int) (*clientpb.Events, error) {
	events := core.EventBroker.GetAll()

	eventspb := &clientpb.Events{
		Events: []*clientpb.Event{},
	}
	for _, e := range events {
		eventspb.Events = append(eventspb.Events, e.ToProtobuf())
	}
	return eventspb, nil
}
