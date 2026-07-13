package rpc

import (
	"context"
	"strconv"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/grpc/metadata"
)

func (rpc *Server) Events(_ *clientpb.Empty, stream clientrpc.MaliceRPC_EventsServer) error {
	broker := core.EventBroker
	if broker == nil {
		return core.ErrEventBrokerUnavailable
	}
	requestMetadata, _ := metadata.FromIncomingContext(stream.Context())
	replayHistory := len(requestMetadata.Get(consts.EventStreamHistoryReplayHeader)) > 0
	var events chan core.Event
	var history []core.Event
	var err error
	if replayHistory {
		events, history, err = broker.SubscribeWithHistory()
	} else {
		events, err = broker.Subscribe()
	}
	if err != nil {
		return err
	}
	clientID := core.GetCurrentID()
	defer func() {
		logs.Log.Infof("client: %d disconnected", clientID)
		core.Clients.Remove(int(clientID))
		broker.Unsubscribe(events)
	}()
	responseHeader := metadata.Pairs(consts.EventStreamReadyHeader, "true")
	if replayHistory {
		responseHeader.Set(consts.EventStreamHistoryCountHeader, strconv.Itoa(len(history)))
	}
	if err := stream.SendHeader(responseHeader); err != nil {
		return err
	}
	for _, event := range history {
		if err := stream.Send(event.ToProtobuf()); err != nil {
			return err
		}
	}

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case event, ok := <-events:
			if !ok {
				return nil
			}
			pb := event.ToProtobuf()
			err := stream.Send(pb)
			if err != nil {
				logs.Log.Warnf("%s", err.Error())
				return err
			}
		}
	}
}

func (rpc *Server) EventsV2(req *clientpb.EventSubscription, stream clientrpc.MaliceRPC_EventsV2Server) error {
	broker := core.EventBroker
	if broker == nil {
		return core.ErrEventBrokerUnavailable
	}
	events, err := broker.SubscribeV2(core.EventSubscription{
		StreamID:          req.GetStreamId(),
		AfterSequence:     req.GetAfterSequence(),
		Replay:            req.GetReplay(),
		Topics:            req.GetTopics(),
		IncludeHeartbeats: req.GetIncludeHeartbeats(),
	})
	if err != nil {
		return err
	}
	if err := stream.SendHeader(metadata.Pairs(consts.EventStreamReadyHeader, "2")); err != nil {
		broker.UnsubscribeV2(events)
		return err
	}
	clientID := core.GetCurrentID()
	defer func() {
		logs.Log.Infof("client: %d disconnected", clientID)
		core.Clients.Remove(int(clientID))
		broker.UnsubscribeV2(events)
	}()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case event, ok := <-events:
			if !ok {
				return nil
			}
			if err := stream.Send(event.ToProtobuf()); err != nil {
				logs.Log.Warnf("EventsV2 send failed: %s", err.Error())
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
	if limit := int(req.GetLimit()); limit > 0 && len(events) > limit {
		events = events[len(events)-limit:]
	}

	eventspb := &clientpb.Events{
		Events: []*clientpb.Event{},
	}
	for _, e := range events {
		eventspb.Events = append(eventspb.Events, e.ToProtobuf())
	}
	return eventspb, nil
}
