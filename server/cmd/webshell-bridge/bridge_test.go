package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	malefic "github.com/chainreactors/malice-network/server/internal/parser/malefic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

type blockingDialTransport struct {
	started chan struct{}
}

func (t *blockingDialTransport) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	select {
	case <-t.started:
	default:
		close(t.started)
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

// badHandshakeTransport returns a connection that sends an invalid malefic frame.
type badHandshakeTransport struct{}

func (t *badHandshakeTransport) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	clientConn, serverConn := net.Pipe()
	go func() {
		defer serverConn.Close()
		// Send a malefic frame with corrupt payload (invalid protobuf)
		frame := []byte{
			malefic.DefaultStartDelimiter,
			0x01, 0x00, 0x00, 0x00, // sessionID = 1
			0x03, 0x00, 0x00, 0x00, // length = 3
			0xBA, 0xAD, 0x00, // corrupt payload
			malefic.DefaultEndDelimiter,
		}
		serverConn.Write(frame)
	}()
	return clientConn, nil
}

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
	f.closeOnce.Do(func() {
		close(f.closed)
	})
	return nil
}
func (f *fakeSpiteStream) Context() context.Context { return context.Background() }
func (f *fakeSpiteStream) SendMsg(m interface{}) error {
	return nil
}
func (f *fakeSpiteStream) RecvMsg(m interface{}) error {
	<-f.closed
	return io.EOF
}
func (f *fakeSpiteStream) Send(*clientpb.SpiteResponse) error { return nil }
func (f *fakeSpiteStream) Recv() (*clientpb.SpiteRequest, error) {
	<-f.closed
	return nil, io.EOF
}

func TestHandlePipelineStartReturnsBeforeDLLConnectCompletes(t *testing.T) {
	transport := &blockingDialTransport{started: make(chan struct{})}
	bridge := &Bridge{
		cfg: &Config{
			ListenerName: "listener-a",
			DLLAddr:      "127.0.0.1:13338",
		},
		transport: transport,
		rpc: &fakeBridgeRPC{
			spiteStream: newFakeSpiteStream(),
		},
	}

	job := &clientpb.Job{
		Name: "ws-a",
		Pipeline: &clientpb.Pipeline{
			Name: "ws-a",
			Type: pipelineType,
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- bridge.handlePipelineStart(context.Background(), job)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("handlePipelineStart returned error: %v", err)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("handlePipelineStart blocked on DLL connect")
	}

	select {
	case <-transport.started:
	case <-time.After(time.Second):
		t.Fatal("background DLL connect was not started")
	}

	if err := bridge.stopActiveRuntime("ws-a"); err != nil {
		t.Fatalf("stopActiveRuntime failed: %v", err)
	}
}

func TestRunRuntimeSyncsPipelineStopAfterUnexpectedConnectFailure(t *testing.T) {
	rpc := &fakeBridgeRPC{spiteStream: newFakeSpiteStream()}
	bridge := &Bridge{
		cfg: &Config{
			ListenerName: "listener-b",
			DLLAddr:      "127.0.0.1:13338",
		},
		transport: &badHandshakeTransport{},
		rpc:       rpc,
	}

	job := &clientpb.Job{
		Name: "ws-b",
		Pipeline: &clientpb.Pipeline{
			Name: "ws-b",
			Type: pipelineType,
		},
	}

	if err := bridge.handlePipelineStart(context.Background(), job); err != nil {
		t.Fatalf("handlePipelineStart failed: %v", err)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		calls := rpc.StopCalls()
		if len(calls) == 1 {
			if calls[0].GetName() != "ws-b" {
				t.Fatalf("stop name = %q, want %q", calls[0].GetName(), "ws-b")
			}
			if calls[0].GetListenerId() != "listener-b" {
				t.Fatalf("stop listener_id = %q, want %q", calls[0].GetListenerId(), "listener-b")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("expected background stop sync after connect failure")
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
	return &collectingSpiteStream{
		closed: make(chan struct{}),
	}
}

func (s *collectingSpiteStream) Header() (metadata.MD, error) { return nil, nil }
func (s *collectingSpiteStream) Trailer() metadata.MD         { return nil }
func (s *collectingSpiteStream) CloseSend() error {
	s.recvOnce.Do(func() { close(s.closed) })
	return nil
}
func (s *collectingSpiteStream) Context() context.Context { return context.Background() }
func (s *collectingSpiteStream) SendMsg(m interface{}) error {
	return nil
}
func (s *collectingSpiteStream) RecvMsg(m interface{}) error {
	<-s.closed
	return io.EOF
}
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

func TestForwardToSessionStreaming(t *testing.T) {
	// Set up a mock DLL that sends 3 streaming responses for one task
	mock := newMockMaleficDLL(t)
	defer mock.close()

	const taskID uint32 = 500
	const numResponses = 3

	go func() {
		conn, err := mock.ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Send handshake
		regSpite := &implantpb.Spites{
			Spites: []*implantpb.Spite{
				{Body: &implantpb.Spite_Register{Register: mock.register}},
			},
		}
		testWriteMaleficFrame(conn, regSpite, mock.sessionID)

		// Read the initial streaming request
		testReadMaleficFrame(conn)

		// Send multiple responses for the same task
		for i := 0; i < numResponses; i++ {
			resp := &implantpb.Spites{
				Spites: []*implantpb.Spite{
					{
						Name:   fmt.Sprintf("stream-resp-%d", i),
						TaskId: taskID,
					},
				},
			}
			testWriteMaleficFrame(conn, resp, mock.sessionID)
		}
	}()

	// Build a real channel + session
	ch := dialMockDLL(t, mock.addr())
	defer ch.Close()

	reg, err := ch.Handshake()
	if err != nil {
		t.Fatalf("handshake: %v", err)
	}
	_ = reg

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

	bridge := &Bridge{
		cfg: &Config{ListenerName: "test-listener"},
	}

	// Forward with streaming task (Total = -1)
	req := &clientpb.SpiteRequest{
		Session: &clientpb.Session{SessionId: session.ID},
		Task:    &clientpb.Task{TaskId: taskID, Total: -1},
		Spite:   &implantpb.Spite{Name: "start-pty"},
	}

	bridge.forwardToSession(runtime, session.ID, taskID, req)

	// Wait for all streaming responses to be forwarded
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		sent := stream.Sent()
		if len(sent) >= numResponses {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	sent := stream.Sent()
	if len(sent) != numResponses {
		t.Fatalf("expected %d responses, got %d", numResponses, len(sent))
	}
	for i, resp := range sent {
		expected := fmt.Sprintf("stream-resp-%d", i)
		if resp.GetSpite().GetName() != expected {
			t.Errorf("response %d: expected %q, got %q", i, expected, resp.GetSpite().GetName())
		}
		if resp.GetTaskId() != taskID {
			t.Errorf("response %d: expected taskID %d, got %d", i, taskID, resp.GetTaskId())
		}
	}

}

func TestForwardToSessionUnary(t *testing.T) {
	mock := newMockMaleficDLL(t)
	defer mock.close()

	go mock.serve(t, 1) // Handshake + 1 Spite roundtrip

	ch := dialMockDLL(t, mock.addr())
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

	bridge := &Bridge{
		cfg: &Config{ListenerName: "test-listener"},
	}

	// Forward with unary task (Total = 1, not streaming)
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

func TestHandleSyncSession(t *testing.T) {
	mock := newMockMaleficDLL(t)
	defer mock.close()

	go mock.serve(t, 0) // Handshake only

	ch := dialMockDLL(t, mock.addr())
	defer ch.Close()

	if _, err := ch.Handshake(); err != nil {
		t.Fatalf("handshake: %v", err)
	}

	session := &Session{
		ID:         "test-session",
		PipelineID: "test-pipeline",
		ListenerID: "test-listener",
		channel:    ch,
	}

	runtime := &pipelineRuntime{
		name: "test-pipeline",
		secureConfig: &implanttypes.SecureConfig{
			Enable:           true,
			ServerPrivateKey: "AGE-SECRET-KEY-PIPELINE",
			ImplantPublicKey: "age1pipeline",
		},
		done: make(chan struct{}),
	}
	runtime.sessions.Store(session.ID, session)
	runtime.sessionsByRawID.Store(ch.sessionID, session)

	bridge := &Bridge{
		cfg: &Config{ListenerName: "test-listener"},
	}
	bridge.activeMu.Lock()
	bridge.active = runtime
	bridge.activeMu.Unlock()

	// Simulate CtrlListenerSyncSession from server
	bridge.handleSyncSession(&clientpb.Session{
		RawId: ch.sessionID,
		KeyPair: &clientpb.KeyPair{
			PublicKey:  "age1session-specific",
			PrivateKey: "AGE-SECRET-KEY-SESSION",
		},
	})

	// Verify the channel's parser got updated with merged keys
	if ch.parser == nil {
		t.Fatal("parser should not be nil")
	}
	// The parser should now have keyPair set (we can verify via WithSecure's effect)
	// Since MaleficParser.keyPair is unexported, we verify indirectly:
	// the fact that handleSyncSession didn't panic and the channel is still alive
	if !session.Alive() {
		t.Error("session should still be alive after key sync")
	}
}

func TestHandleSyncSessionUnknownRawID(t *testing.T) {
	runtime := &pipelineRuntime{
		name: "test-pipeline",
		done: make(chan struct{}),
	}

	bridge := &Bridge{
		cfg: &Config{ListenerName: "test-listener"},
	}
	bridge.activeMu.Lock()
	bridge.active = runtime
	bridge.activeMu.Unlock()

	// Should not panic with unknown raw ID
	bridge.handleSyncSession(&clientpb.Session{
		RawId: 99999,
		KeyPair: &clientpb.KeyPair{
			PublicKey:  "age1unknown",
			PrivateKey: "AGE-SECRET-KEY-UNKNOWN",
		},
	})
}

var _ listenerrpc.ListenerRPC_SpiteStreamClient = (*fakeSpiteStream)(nil)
var _ listenerrpc.ListenerRPC_SpiteStreamClient = (*collectingSpiteStream)(nil)
var _ listenerrpc.ListenerRPCClient = (*fakeBridgeRPC)(nil)
