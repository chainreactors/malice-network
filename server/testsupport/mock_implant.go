//go:build mockimplant

package testsupport

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/encoders/hash"

	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

type MockImplantHandler func(context.Context, *clientpb.SpiteRequest, func(*implantpb.Spite) error) error

var mockRawIDCounter atomic.Uint32

func nextMockRawID() uint32 {
	for {
		rawID := mockRawIDCounter.Add(1)
		if rawID != 0 {
			return rawID
		}
	}
}

type MockImplant struct {
	Harness  *ControlPlaneHarness
	Pipeline *clientpb.Pipeline
	Config   *mtls.ClientConfig

	AutoCheckinInterval time.Duration

	SessionID string
	RawID     uint32
	Target    string
	Register  *implantpb.Register

	Conn   *grpc.ClientConn
	Client listenerrpc.ListenerRPCClient
	Stream listenerrpc.ListenerRPC_SpiteStreamClient

	ctx    context.Context
	cancel context.CancelFunc

	sendMu sync.Mutex
	wg     sync.WaitGroup

	periodicCheckinsPaused atomic.Bool

	mu       sync.Mutex
	requests []*clientpb.SpiteRequest
	errors   []error
	handlers map[string]MockImplantHandler
}

func NewMockImplant(t testing.TB, h *ControlPlaneHarness, pipeline *clientpb.Pipeline) *MockImplant {
	t.Helper()

	if h == nil {
		t.Fatal("control plane harness is nil")
	}
	if pipeline == nil {
		t.Fatal("pipeline is nil")
	}

	pipeline = proto.Clone(pipeline).(*clientpb.Pipeline)
	h.SeedPipeline(t, pipeline, true)

	name := fmt.Sprintf("mock-listener-%d", time.Now().UnixNano())
	rawID := nextMockRawID()
	mock := &MockImplant{
		Harness:             h,
		Pipeline:            pipeline,
		Config:              h.NewListenerClientConfig(t, name),
		AutoCheckinInterval: time.Second,
		SessionID:           hash.Md5Hash(encoders.Uint32ToBytes(rawID)),
		RawID:               rawID,
		Target:              "127.0.0.1",
		Register: &implantpb.Register{
			Name: "mock-implant",
			Timer: &implantpb.Timer{
				Expression: "* * * * *",
			},
			Sysinfo: &implantpb.SysInfo{
				Os: &implantpb.Os{
					Name:     "windows",
					Arch:     "amd64",
					Hostname: "mock-host",
				},
				Process: &implantpb.Process{
					Name: "mock.exe",
				},
			},
		},
		handlers: make(map[string]MockImplantHandler),
	}

	t.Cleanup(func() {
		_ = mock.Close()
	})

	return mock
}

func (m *MockImplant) On(name string, handler MockImplantHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[name] = handler
}

func (m *MockImplant) Start() error {
	if m == nil {
		return fmt.Errorf("mock implant is nil")
	}
	if m.Client != nil {
		return nil
	}

	connectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := m.Harness.ConnectWithConfig(connectCtx, m.Config)
	if err != nil {
		return err
	}

	m.Conn = conn
	m.Client = listenerrpc.NewListenerRPCClient(conn)
	m.ctx, m.cancel = context.WithCancel(context.Background())

	streamCtx := metadata.NewOutgoingContext(m.ctx, metadata.Pairs("pipeline_id", m.Pipeline.Name))
	stream, err := m.Client.SpiteStream(streamCtx)
	if err != nil {
		_ = conn.Close()
		m.Conn = nil
		m.Client = nil
		m.cancel()
		return err
	}
	m.Stream = stream

	m.wg.Add(1)
	go m.recvLoop()

	registerCtx, registerCancel := context.WithTimeout(m.rpcContext(m.SessionID), 10*time.Second)
	defer registerCancel()

	_, err = m.Client.Register(registerCtx, &clientpb.RegisterSession{
		Type:         m.Pipeline.Type,
		SessionId:    m.SessionID,
		RawId:        m.RawID,
		PipelineId:   m.Pipeline.Name,
		ListenerId:   m.Pipeline.ListenerId,
		Target:       m.Target,
		RegisterData: proto.Clone(m.Register).(*implantpb.Register),
	})
	if err != nil {
		_ = m.Close()
		return err
	}

	if err := m.Checkin(); err != nil {
		_ = m.Close()
		return fmt.Errorf("mock implant initial checkin: %w", err)
	}

	if m.AutoCheckinInterval > 0 {
		m.wg.Add(1)
		go m.autoCheckinLoop()
	}

	return nil
}

func (m *MockImplant) Checkin() error {
	if m == nil || m.Client == nil {
		return fmt.Errorf("mock implant is not started")
	}

	ctx, cancel := context.WithTimeout(m.rpcContext(m.SessionID), 5*time.Second)
	defer cancel()

	_, err := m.Client.Checkin(ctx, &implantpb.Ping{Nonce: 1})
	return err
}

func (m *MockImplant) PauseAutoCheckins() {
	if m == nil {
		return
	}
	m.periodicCheckinsPaused.Store(true)
}

func (m *MockImplant) ResumeAutoCheckins() {
	if m == nil {
		return
	}
	m.periodicCheckinsPaused.Store(false)
}

func (m *MockImplant) Requests() []*clientpb.SpiteRequest {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]*clientpb.SpiteRequest, 0, len(m.requests))
	for _, request := range m.requests {
		out = append(out, proto.Clone(request).(*clientpb.SpiteRequest))
	}
	return out
}

func (m *MockImplant) RequestsByName(name string) []*clientpb.SpiteRequest {
	requests := m.Requests()
	filtered := make([]*clientpb.SpiteRequest, 0, len(requests))
	for _, request := range requests {
		if request.GetSpite().GetName() == name {
			filtered = append(filtered, request)
		}
	}
	return filtered
}

func (m *MockImplant) LastRequest(name string) *clientpb.SpiteRequest {
	requests := m.RequestsByName(name)
	if len(requests) == 0 {
		return nil
	}
	return requests[len(requests)-1]
}

func (m *MockImplant) Errors() []error {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]error, len(m.errors))
	copy(out, m.errors)
	return out
}

func (m *MockImplant) Close() error {
	if m == nil {
		return nil
	}

	if m.cancel != nil {
		m.cancel()
	}
	if m.Stream != nil {
		_ = m.Stream.CloseSend()
	}
	if m.Conn != nil {
		_ = m.Conn.Close()
	}
	m.wg.Wait()
	return nil
}

func (m *MockImplant) recvLoop() {
	defer m.wg.Done()

	for {
		req, err := m.Stream.Recv()
		if err != nil {
			if err == io.EOF || (m.ctx != nil && m.ctx.Err() != nil) {
				return
			}
			m.appendError(fmt.Errorf("mock implant recv: %w", err))
			return
		}

		m.recordRequest(req)

		handler := m.handler(req.GetSpite().GetName())
		if handler == nil {
			m.appendError(fmt.Errorf("mock implant has no handler for spite %q", req.GetSpite().GetName()))
			continue
		}

		m.wg.Add(1)
		go func(request *clientpb.SpiteRequest, fn MockImplantHandler) {
			defer m.wg.Done()

			err := fn(m.ctx, proto.Clone(request).(*clientpb.SpiteRequest), func(spite *implantpb.Spite) error {
				return m.sendResponse(request, spite)
			})
			if err != nil {
				m.appendError(err)
			}
		}(proto.Clone(req).(*clientpb.SpiteRequest), handler)
	}
}

func (m *MockImplant) autoCheckinLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.AutoCheckinInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if m.periodicCheckinsPaused.Load() {
				continue
			}
			if err := m.Checkin(); err != nil {
				if m.ctx != nil && m.ctx.Err() != nil {
					return
				}
				m.appendError(fmt.Errorf("mock implant auto checkin: %w", err))
			}
		case <-m.ctx.Done():
			return
		}
	}
}

func (m *MockImplant) sendResponse(req *clientpb.SpiteRequest, spite *implantpb.Spite) error {
	if spite == nil {
		return fmt.Errorf("mock implant response is nil")
	}

	response := proto.Clone(spite).(*implantpb.Spite)
	if response.TaskId == 0 && req.GetTask() != nil {
		response.TaskId = req.GetTask().GetTaskId()
	}
	if response.Name == "" {
		response.Name = req.GetSpite().GetName()
	}

	m.sendMu.Lock()
	defer m.sendMu.Unlock()

	return m.Stream.Send(&clientpb.SpiteResponse{
		ListenerId: m.Pipeline.ListenerId,
		SessionId:  req.GetSession().GetSessionId(),
		TaskId:     response.TaskId,
		Spite:      response,
	})
}

func (m *MockImplant) recordRequest(req *clientpb.SpiteRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = append(m.requests, proto.Clone(req).(*clientpb.SpiteRequest))
}

func (m *MockImplant) handler(name string) MockImplantHandler {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.handlers[name]
}

func (m *MockImplant) appendError(err error) {
	if err == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors = append(m.errors, err)
}

func (m *MockImplant) rpcContext(sessionID string) context.Context {
	return metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"session_id", sessionID,
		"listener_id", m.Pipeline.ListenerId,
		"timestamp", strconv.FormatInt(time.Now().Unix(), 10),
	))
}
