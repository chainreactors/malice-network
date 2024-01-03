package rpc

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/server/core"
)

func (rpc *Server) Events(_ *clientpb.Empty, stream clientrpc.MaliceRPC_EventsServer) error {
	clientName, err := rpc.getClientName(stream.Context())
	if err != nil {
		return err
	}
	client := core.NewClient(clientName)
	core.Clients.Add(client)
	events := core.EventBroker.Subscribe()

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
				EventType: event.EventType,
				Data:      event.Data,
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
			if event.Err != nil {
				pbEvent.Err = event.Err.Error()
			}

			err := stream.Send(pbEvent)
			if err != nil {
				logs.Log.Warnf(err.Error())
				return err
			}
		}
	}
}
