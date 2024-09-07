package rpc

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func (rpc *Server) Events(_ *clientpb.Empty, stream clientrpc.MaliceRPC_EventsServer) error {
	clientName := getClientName(stream.Context())
	events := core.EventBroker.Subscribe()
	client := core.NewClient(clientName)
	core.Clients.Add(client)
	defer func() {
		logs.Log.Infof("%d client disconnected", client.ID)
		core.EventBroker.Unsubscribe(events)
		core.Clients.Remove(int(client.ID))
	}()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case event := <-events:
			pbEvent := &clientpb.Event{
				Type:   event.EventType,
				Op:     event.Op,
				Source: event.SourceName,
				Data:   event.Data,
			}

			if event.Job != nil {
				pbEvent.Job = event.Job.ToProtobuf()
			}
			if event.Client != nil {
				pbEvent.Client = event.Client.ToProtobuf()
			}
			if event.Session != nil {
				pbEvent.Session = event.Session.ToProtobuf()
			}
			if event.Task != nil {
				pbEvent.Task = event.Task.ToProtobuf()
			}
			if event.Err != "" {
				pbEvent.Err = event.Err
			}
			if event.Message != "" {
				pbEvent.Message = event.Message
			}
			err := stream.Send(pbEvent)
			if err != nil {
				logs.Log.Warnf(err.Error())
				return err
			}
		}
	}
}

func (rpc *Server) Broadcast(ctx context.Context, req *clientpb.Event) (*clientpb.Empty, error) {
	clientName := getClientName(ctx)
	core.EventBroker.Publish(core.Event{
		EventType:  req.Type,
		Data:       req.Data,
		SourceName: clientName,
		Err:        req.Err,
	})
	return &clientpb.Empty{}, nil
}
