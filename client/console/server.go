package console

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"io"
)

type Listener struct {
	*clientpb.Listener
}

type Client struct {
	*clientpb.Client
}

func InitServerStatus(conn *grpc.ClientConn) (*ServerStatus, error) {
	var err error
	s := &ServerStatus{
		Rpc:      clientrpc.NewMaliceRPCClient(conn),
		Sessions: make(map[string]*clientpb.Session),
		Alive:    true,
	}

	s.Info, err = s.Rpc.GetBasic(context.Background(), &clientpb.Empty{})
	if err != nil {
		return nil, err
	}

	clients, err := s.Rpc.GetClients(context.Background(), &clientpb.Empty{})
	if err != nil {
		return nil, err
	}
	for _, client := range clients.GetClients() {
		s.Clients = append(s.Clients, &Client{client})
	}

	listeners, err := s.Rpc.GetListeners(context.Background(), &clientpb.Empty{})
	if err != nil {
		return nil, err
	}
	for _, listener := range listeners.GetListeners() {
		s.Listeners = append(s.Listeners, &Listener{listener})
	}

	err = s.UpdateSession()
	if err != nil {
		return nil, err
	}

	go s.EventHandler()

	return s, nil
}

type ServerStatus struct {
	Rpc       clientrpc.MaliceRPCClient
	Info      *clientpb.Basic
	Clients   []*Client
	Listeners []*Listener
	Sessions  map[string]*clientpb.Session
	Alive     bool
}

func (s *ServerStatus) UpdateSession() error {
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"client_name", "test"),
	)
	sessions, err := s.Rpc.GetSessions(ctx, &clientpb.Empty{})
	if err != nil {
		return err
	}

	if len(sessions.GetSessions()) == 0 {
		return nil
	}

	for _, session := range sessions.GetSessions() {
		s.Sessions[session.SessionId] = session
	}
	return nil
}

func (s *ServerStatus) EventHandler() {
	eventStream, err := s.Rpc.Events(context.Background(), &clientpb.Empty{})
	if err != nil {
		logs.Log.Warnf("Error getting event stream: %v", err)
		return
	}
	for {
		event, err := eventStream.Recv()
		if err == io.EOF || event == nil {
			return
		}

		// Trigger event based on type
		switch event.EventType {

		case consts.EventJoin:
			Log.Infof("%s has joined the game", event.Client.Name)
		case consts.EventLeft:
			Log.Infof("%s left the game", event.Client.Name)

		}

		//con.triggerReactions(event)
	}
}
