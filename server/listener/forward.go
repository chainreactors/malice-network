package listener

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	iomtypes "github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/forwardrpc"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const forwardStreamCloseTimeout = 5 * time.Second

var (
	ErrForwardStreamClosing      = errors.New("forward stream is closing")
	ErrForwardStreamCloseTimeout = errors.New("forward stream close timed out")
	ErrForwardStreamPoisoned     = errors.New("forward stream has not drained")
)

type ForwardListener struct {
	lns      *listener
	server   *grpc.Server
	listener net.Listener
}

func NewForwardListener(cfg *configs.ListenerConfig) (*ForwardListener, error) {
	if cfg == nil {
		return nil, fmt.Errorf("listener config is nil")
	}
	forwardCfg := cfg.ForwardConfigOrDefault()
	registry := newForwardStreamRegistry()
	rpcClient := &forwardPipelineRPC{
		listenerID: cfg.Name,
		registry:   registry,
	}
	lns := &listener{
		Name:        cfg.Name,
		IP:          cfg.IP,
		pipelines:   core.NewPipelines(),
		cfg:         cfg,
		websites:    make(map[string]*Website),
		pipelineRPC: rpcClient,
	}

	serverOptions, err := forwardrpc.ServerOptions(cfg.Auth)
	if err != nil {
		return nil, err
	}

	ln, err := net.Listen("tcp", forwardCfg.ListenAddress())
	if err != nil {
		return nil, err
	}
	grpcServer := grpc.NewServer(serverOptions...)
	forwardrpc.RegisterForwardListenerServer(grpcServer, &forwardListenerService{
		lns:      lns,
		registry: registry,
	})
	runtime := &ForwardListener{
		lns:      lns,
		server:   grpcServer,
		listener: ln,
	}
	lns.shutdown = runtime.Close
	core.GoGuarded("forward-listener:"+cfg.Name, func() error {
		if err := grpcServer.Serve(ln); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			return err
		}
		return nil
	}, core.LogGuardedError("forward-listener:"+cfg.Name))
	setCurrentListener(lns)
	logs.Log.Importantf("listener.forward - start name=%s address=%s", cfg.Name, ln.Addr().String())
	return runtime, nil
}

func (f *ForwardListener) Close() error {
	if f == nil {
		return nil
	}
	if f.server != nil {
		f.server.Stop()
	}
	if f.listener != nil {
		_ = f.listener.Close()
	}
	if f.lns != nil {
		return f.lns.Close()
	}
	return nil
}

type forwardListenerService struct {
	forwardrpc.UnimplementedForwardListenerServer
	lns      *listener
	registry *forwardStreamRegistry
}

func (s *forwardListenerService) ControlStream(stream forwardrpc.ForwardListener_ControlStreamServer) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		statusMsg := s.lns.handleJobCtrl(msg)
		if statusMsg == nil {
			continue
		}
		if err := stream.Send(statusMsg); err != nil {
			return err
		}
	}
}

func (s *forwardListenerService) TaskStream(stream forwardrpc.ForwardListener_TaskStreamServer) error {
	pipelineID, err := metadataValue(stream.Context(), "pipeline_id")
	if err != nil {
		return err
	}
	listenerID, err := metadataValue(stream.Context(), "listener_id")
	if err != nil {
		return err
	}
	local, err := s.registry.open(listenerID, pipelineID)
	if err != nil {
		return status.Error(codes.Unavailable, err.Error())
	}
	local.attach(stream)
	return local.serve(stream)
}

type forwardPipelineRPC struct {
	listenerID string
	registry   *forwardStreamRegistry
}

func (c *forwardPipelineRPC) OpenForwardStream(ctx context.Context, pipeline core.Pipeline) (core.ForwardStream, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	listenerID := c.listenerID
	if pb := pipeline.ToProtobuf(); pb != nil && pb.ListenerId != "" {
		listenerID = pb.ListenerId
	}
	return c.registry.open(listenerID, pipeline.ID())
}

func (c *forwardPipelineRPC) Register(ctx context.Context, in *clientpb.RegisterSession, _ ...grpc.CallOption) (*clientpb.Empty, error) {
	if in == nil {
		return nil, fmt.Errorf("register session is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	stream, err := c.registry.open(in.ListenerId, in.PipelineId)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, stream.sendEventContext(ctx, &clientpb.SpiteRequest{
		ListenerId: in.ListenerId,
		Session: &clientpb.Session{
			SessionId:  in.SessionId,
			RawId:      in.RawId,
			PipelineId: in.PipelineId,
			ListenerId: in.ListenerId,
			Target:     in.Target,
			Type:       in.Type,
		},
		Spite: &implantpb.Spite{
			Name: iomtypes.MsgRegister.String(),
			Body: &implantpb.Spite_Register{Register: in.RegisterData},
		},
	})
}

func (c *forwardPipelineRPC) Checkin(ctx context.Context, in *implantpb.Ping, _ ...grpc.CallOption) (*clientpb.Empty, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	sessionID, _ := metadataValue(ctx, "session_id")
	listenerID, _ := metadataValue(ctx, "listener_id")
	pipelineID, _ := metadataValue(ctx, "pipeline_id")
	if listenerID == "" {
		listenerID = c.listenerID
	}
	stream, err := c.registry.open(listenerID, pipelineID)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, stream.sendEventContext(ctx, &clientpb.SpiteRequest{
		ListenerId: listenerID,
		Session: &clientpb.Session{
			SessionId:  sessionID,
			PipelineId: pipelineID,
			ListenerId: listenerID,
		},
		Spite: &implantpb.Spite{
			Name: iomtypes.MsgPing.String(),
			Body: &implantpb.Spite_Ping{Ping: in},
		},
	})
}

func (c *forwardPipelineRPC) GetArtifact(context.Context, *clientpb.Artifact, ...grpc.CallOption) (*clientpb.Artifact, error) {
	return nil, status.Error(codes.Unimplemented, "artifact fetch is not supported by forward listener transport")
}

func (c *forwardPipelineRPC) retireStream(listenerID, pipelineID string) (func(), error) {
	return c.registry.retire(listenerID, pipelineID)
}

type forwardStreamRegistry struct {
	mu       sync.Mutex
	streams  map[string]*forwardLocalStream
	retiring map[string]int
	poisoned map[string]*forwardLocalStream
}

func newForwardStreamRegistry() *forwardStreamRegistry {
	return &forwardStreamRegistry{
		streams:  make(map[string]*forwardLocalStream),
		retiring: make(map[string]int),
		poisoned: make(map[string]*forwardLocalStream),
	}
}

func (r *forwardStreamRegistry) open(listenerID, pipelineID string) (*forwardLocalStream, error) {
	key := core.PipelineRuntimeKey(listenerID, pipelineID)
	r.mu.Lock()
	defer r.mu.Unlock()
	if stream := r.streams[key]; stream != nil {
		return stream, nil
	}
	if r.retiring[key] > 0 {
		return nil, fmt.Errorf("%w: %s", ErrForwardStreamClosing, key)
	}
	if r.poisoned[key] != nil {
		return nil, fmt.Errorf("%w: %s", ErrForwardStreamPoisoned, key)
	}
	stream := newForwardLocalStream(r, listenerID, pipelineID)
	r.streams[key] = stream
	return stream, nil
}

func newForwardLocalStream(registry *forwardStreamRegistry, listenerID, pipelineID string) *forwardLocalStream {
	return &forwardLocalStream{
		registry:   registry,
		listenerID: listenerID,
		pipelineID: pipelineID,
		requests:   make(chan *clientpb.SpiteRequest, 255),
		events:     make(chan *clientpb.SpiteRequest, 255),
		done:       make(chan struct{}),
	}
}

func (r *forwardStreamRegistry) retire(listenerID, pipelineID string) (func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), forwardStreamCloseTimeout)
	defer cancel()
	return r.retireContext(ctx, listenerID, pipelineID)
}

func (r *forwardStreamRegistry) retireContext(ctx context.Context, listenerID, pipelineID string) (func(), error) {
	key := core.PipelineRuntimeKey(listenerID, pipelineID)
	r.mu.Lock()
	r.retiring[key]++
	stream := r.streams[key]
	delete(r.streams, key)
	r.mu.Unlock()

	var once sync.Once
	release := func() {
		once.Do(func() {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.retiring[key]--
			if r.retiring[key] == 0 {
				delete(r.retiring, key)
			}
		})
	}
	if stream != nil {
		err := stream.closeContext(ctx)
		r.poisonIfUndrained(key, stream, err)
		return release, err
	}
	return release, nil
}

func (r *forwardStreamRegistry) discard(stream *forwardLocalStream) error {
	if r == nil || stream == nil {
		return nil
	}
	key := core.PipelineRuntimeKey(stream.listenerID, stream.pipelineID)
	r.mu.Lock()
	if r.streams[key] != stream {
		r.mu.Unlock()
		return stream.close()
	}
	r.retiring[key]++
	delete(r.streams, key)
	r.mu.Unlock()

	err := stream.close()
	r.poisonIfUndrained(key, stream, err)
	r.mu.Lock()
	r.retiring[key]--
	if r.retiring[key] == 0 {
		delete(r.retiring, key)
	}
	r.mu.Unlock()
	return err
}

func (r *forwardStreamRegistry) poisonIfUndrained(key string, stream *forwardLocalStream, closeErr error) {
	if !errors.Is(closeErr, ErrForwardStreamCloseTimeout) || stream.drained == nil {
		return
	}
	r.mu.Lock()
	r.poisoned[key] = stream
	r.mu.Unlock()
	go func() {
		<-stream.drained
		r.mu.Lock()
		if r.poisoned[key] == stream {
			delete(r.poisoned, key)
		}
		r.mu.Unlock()
	}()
}

type forwardLocalStream struct {
	registry   *forwardStreamRegistry
	listenerID string
	pipelineID string
	requests   chan *clientpb.SpiteRequest
	events     chan *clientpb.SpiteRequest
	done       chan struct{}
	closeOnce  sync.Once
	closed     atomic.Bool
	eventMu    sync.Mutex
	slotsOnce  sync.Once
	eventSlots chan struct{}
	remoteSend sync.WaitGroup
	closeErr   error
	drained    chan struct{}
}

func (s *forwardLocalStream) Send(resp *clientpb.SpiteResponse) error {
	if resp == nil {
		return nil
	}
	return s.sendEvent(&clientpb.SpiteRequest{
		ListenerId: s.listenerID,
		Session: &clientpb.Session{
			SessionId:  resp.SessionId,
			PipelineId: s.pipelineID,
			ListenerId: s.listenerID,
		},
		Task:  &clientpb.Task{TaskId: resp.TaskId, SessionId: resp.SessionId},
		Spite: resp.Spite,
	})
}

func (s *forwardLocalStream) CloseSend() error {
	if s.registry != nil {
		return s.registry.discard(s)
	}
	return s.close()
}

func (s *forwardLocalStream) Recv() (*clientpb.SpiteRequest, error) {
	select {
	case req, ok := <-s.requests:
		if !ok {
			return nil, io.EOF
		}
		return req, nil
	case <-s.done:
		return nil, io.EOF
	}
}

func (s *forwardLocalStream) attach(_ forwardrpc.ForwardListener_TaskStreamServer) {}

func (s *forwardLocalStream) serve(stream forwardrpc.ForwardListener_TaskStreamServer) error {
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	errCh := make(chan error, 2)
	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				errCh <- err
				return
			}
			select {
			case s.requests <- req:
			case <-ctx.Done():
				return
			case <-s.done:
				return
			}
		}
	}()
	go func() {
		s.eventSlotPool()
		for {
			select {
			case event, ok := <-s.events:
				if !ok {
					return
				}
				s.releaseEventSlot()
				if err := s.sendRemoteEvent(stream, event); err != nil {
					if errors.Is(err, io.ErrClosedPipe) {
						return
					}
					errCh <- err
					return
				}
			case <-ctx.Done():
				return
			case <-s.done:
				return
			}
		}
	}()
	var err error
	select {
	case err = <-errCh:
	case <-s.done:
		return nil
	}
	if err == io.EOF {
		return nil
	}
	return err
}

func (s *forwardLocalStream) close() error {
	ctx, cancel := context.WithTimeout(context.Background(), forwardStreamCloseTimeout)
	defer cancel()
	return s.closeContext(ctx)
}

func (s *forwardLocalStream) closeContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.closeOnce.Do(func() {
		s.eventMu.Lock()
		s.closed.Store(true)
		if s.done != nil {
			close(s.done)
		}
		s.eventMu.Unlock()

		s.drained = make(chan struct{})
		go func() {
			s.remoteSend.Wait()
			close(s.drained)
		}()
		select {
		case <-s.drained:
		case <-ctx.Done():
			s.closeErr = fmt.Errorf("%w: %s:%s: %v", ErrForwardStreamCloseTimeout, s.listenerID, s.pipelineID, ctx.Err())
		}
	})
	return s.closeErr
}

func (s *forwardLocalStream) sendEvent(event *clientpb.SpiteRequest) error {
	return s.sendEventContext(context.Background(), event)
}

func (s *forwardLocalStream) sendEventContext(ctx context.Context, event *clientpb.SpiteRequest) error {
	if event == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	slots := s.eventSlotPool()
	select {
	case <-slots:
		s.eventMu.Lock()
		if s.closed.Load() || ctx.Err() != nil {
			s.eventMu.Unlock()
			s.releaseEventSlot()
			if err := ctx.Err(); err != nil {
				return err
			}
			return io.ErrClosedPipe
		}
		s.events <- event
		s.eventMu.Unlock()
		return nil
	case <-s.done:
		return io.ErrClosedPipe
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return fmt.Errorf("forward stream %s:%s event queue full", s.listenerID, s.pipelineID)
	}
}

func (s *forwardLocalStream) eventSlotPool() chan struct{} {
	s.slotsOnce.Do(func() {
		capacity := cap(s.events)
		s.eventSlots = make(chan struct{}, capacity)
		for i := len(s.events); i < capacity; i++ {
			s.eventSlots <- struct{}{}
		}
	})
	return s.eventSlots
}

func (s *forwardLocalStream) releaseEventSlot() {
	s.eventSlotPool() <- struct{}{}
}

func (s *forwardLocalStream) sendRemoteEvent(stream forwardrpc.ForwardListener_TaskStreamServer, event *clientpb.SpiteRequest) error {
	s.eventMu.Lock()
	if s.closed.Load() {
		s.eventMu.Unlock()
		return io.ErrClosedPipe
	}
	s.remoteSend.Add(1)
	s.eventMu.Unlock()

	defer s.remoteSend.Done()
	err := stream.Send(event)
	return err
}

func metadataValue(ctx context.Context, key string) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md, ok = metadata.FromOutgoingContext(ctx)
	}
	if !ok {
		return "", status.Errorf(codes.InvalidArgument, "missing metadata %s", key)
	}
	values := md.Get(key)
	if len(values) == 0 || values[0] == "" {
		return "", status.Errorf(codes.InvalidArgument, "missing metadata %s", key)
	}
	return values[0], nil
}
