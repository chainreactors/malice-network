package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

type fakeBridgeRPC struct {
	listenerrpc.ListenerRPCClient

	mu          sync.Mutex
	stopCalls   []*clientpb.CtrlPipeline
	spiteStream listenerrpc.ListenerRPC_SpiteStreamClient
}

func (f *fakeBridgeRPC) SpiteStream(ctx context.Context, opts ...grpc.CallOption) (listenerrpc.ListenerRPC_SpiteStreamClient, error) {
	if f.spiteStream == nil {
		return nil, errors.New("missing spite stream")
	}
	return f.spiteStream, nil
}

func (f *fakeBridgeRPC) StopPipeline(ctx context.Context, in *clientpb.CtrlPipeline, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopCalls = append(f.stopCalls, proto.Clone(in).(*clientpb.CtrlPipeline))
	return &clientpb.Empty{}, nil
}

func (f *fakeBridgeRPC) StopCalls() []*clientpb.CtrlPipeline {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*clientpb.CtrlPipeline, len(f.stopCalls))
	copy(out, f.stopCalls)
	return out
}

type fakeSpiteStream struct {
	grpc.ClientStream

	closeOnce sync.Once
	closed    chan struct{}
}

func newFakeSpiteStream() *fakeSpiteStream {
	return &fakeSpiteStream{
		closed: make(chan struct{}),
	}
}

func (f *fakeSpiteStream) Header() (metadata.MD, error) { return nil, nil }
func (f *fakeSpiteStream) Trailer() metadata.MD         { return nil }
func (f *fakeSpiteStream) CloseSend() error {
	f.closeOnce.Do(func() { close(f.closed) })
	return nil
}
func (f *fakeSpiteStream) Context() context.Context      { return context.Background() }
func (f *fakeSpiteStream) SendMsg(m interface{}) error    { return nil }
func (f *fakeSpiteStream) RecvMsg(m interface{}) error    { <-f.closed; return io.EOF }
func (f *fakeSpiteStream) Send(*clientpb.SpiteResponse) error { return nil }
func (f *fakeSpiteStream) Recv() (*clientpb.SpiteRequest, error) {
	<-f.closed
	return nil, io.EOF
}

// collectingSpiteStream records all sent SpiteResponses for test assertions.
type collectingSpiteStream struct {
	grpc.ClientStream

	mu       sync.Mutex
	sent     []*clientpb.SpiteResponse
	closed   chan struct{}
	recvOnce sync.Once
}

func newCollectingSpiteStream() *collectingSpiteStream {
	return &collectingSpiteStream{closed: make(chan struct{})}
}

func (s *collectingSpiteStream) Header() (metadata.MD, error) { return nil, nil }
func (s *collectingSpiteStream) Trailer() metadata.MD         { return nil }
func (s *collectingSpiteStream) CloseSend() error {
	s.recvOnce.Do(func() { close(s.closed) })
	return nil
}
func (s *collectingSpiteStream) Context() context.Context   { return context.Background() }
func (s *collectingSpiteStream) SendMsg(m interface{}) error { return nil }
func (s *collectingSpiteStream) RecvMsg(m interface{}) error { <-s.closed; return io.EOF }
func (s *collectingSpiteStream) Send(resp *clientpb.SpiteResponse) error {
	s.mu.Lock()
	s.sent = append(s.sent, proto.Clone(resp).(*clientpb.SpiteResponse))
	s.mu.Unlock()
	return nil
}
func (s *collectingSpiteStream) Recv() (*clientpb.SpiteRequest, error) {
	<-s.closed
	return nil, io.EOF
}
func (s *collectingSpiteStream) Sent() []*clientpb.SpiteResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*clientpb.SpiteResponse, len(s.sent))
	copy(out, s.sent)
	return out
}

func TestForwardToSessionUnary(t *testing.T) {
	srv, _ := startMockWebshell(t)

	ch := NewChannel(srv.URL, "")
	defer ch.Close()

	if _, err := ch.Handshake(); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	ch.StartRecvLoop()

	session := &Session{
		ID:         "test-session",
		PipelineID: "test-pipeline",
		ListenerID: "test-listener",
		channel:    ch,
	}

	stream := newCollectingSpiteStream()
	runtimeCtx, runtimeCancel := context.WithCancel(context.Background())
	defer runtimeCancel()

	runtime := &pipelineRuntime{
		name:        "test-pipeline",
		ctx:         runtimeCtx,
		cancel:      runtimeCancel,
		spiteStream: stream,
		done:        make(chan struct{}),
	}
	runtime.sessions.Store(session.ID, session)

	bridge := &Bridge{cfg: &Config{ListenerName: "test-listener"}}

	req := &clientpb.SpiteRequest{
		Session: &clientpb.Session{SessionId: session.ID},
		Task:    &clientpb.Task{TaskId: 1, Total: 1},
		Spite:   &implantpb.Spite{Name: "exec"},
	}

	bridge.forwardToSession(runtime, session.ID, 1, req)

	sent := stream.Sent()
	if len(sent) != 1 {
		t.Fatalf("expected 1 response, got %d", len(sent))
	}
	if sent[0].GetSpite().GetName() != "resp:exec" {
		t.Errorf("expected 'resp:exec', got %q", sent[0].GetSpite().GetName())
	}
}

func TestForwardToSessionStreaming(t *testing.T) {
	srv, mock := startMockWebshell(t)

	var callCount int
	var mu sync.Mutex
	mock.setHandler(func(stage string, body []byte) ([]byte, int) {
		if stage == "status" {
			return []byte("LOADED"), 200
		}
		if stage == "init" {
			regData, _ := proto.Marshal(mock.register)
			sid := make([]byte, 4)
			sid[0] = byte(mock.sessionID)
			return append(sid, regData...), 200
		}
		if stage == "spite" {
			// Parse input
			inSpites := &implantpb.Spites{}
			if len(body) > 0 {
				proto.Unmarshal(body, inSpites)
			}

			mu.Lock()
			callCount++
			n := callCount
			mu.Unlock()

			// First few calls: return streaming responses for task 500
			if n <= 3 {
				resp := &implantpb.Spites{
					Spites: []*implantpb.Spite{{
						Name:   fmt.Sprintf("stream-resp-%d", n-1),
						TaskId: 500,
					}},
				}
				data, _ := proto.Marshal(resp)
				return data, 200
			}
			empty, _ := proto.Marshal(&implantpb.Spites{})
			return empty, 200
		}
		return nil, 404
	})

	ch := NewChannel(srv.URL, "")
	defer ch.Close()

	if _, err := ch.Handshake(); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	ch.StartRecvLoop()

	session := &Session{
		ID:         "test-session",
		PipelineID: "test-pipeline",
		ListenerID: "test-listener",
		channel:    ch,
	}

	stream := newCollectingSpiteStream()
	runtimeCtx, runtimeCancel := context.WithCancel(context.Background())
	defer runtimeCancel()

	runtime := &pipelineRuntime{
		name:        "test-pipeline",
		ctx:         runtimeCtx,
		cancel:      runtimeCancel,
		spiteStream: stream,
		done:        make(chan struct{}),
	}
	runtime.sessions.Store(session.ID, session)

	bridge := &Bridge{cfg: &Config{ListenerName: "test-listener"}}

	req := &clientpb.SpiteRequest{
		Session: &clientpb.Session{SessionId: session.ID},
		Task:    &clientpb.Task{TaskId: 500, Total: -1},
		Spite:   &implantpb.Spite{Name: "start-pty"},
	}

	bridge.forwardToSession(runtime, session.ID, 500, req)

	// Wait for streaming responses
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		sent := stream.Sent()
		if len(sent) >= 2 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	sent := stream.Sent()
	if len(sent) < 2 {
		t.Fatalf("expected at least 2 streaming responses, got %d", len(sent))
	}
}

var _ listenerrpc.ListenerRPC_SpiteStreamClient = (*fakeSpiteStream)(nil)
var _ listenerrpc.ListenerRPC_SpiteStreamClient = (*collectingSpiteStream)(nil)
var _ listenerrpc.ListenerRPCClient = (*fakeBridgeRPC)(nil)
