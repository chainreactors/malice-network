package listener

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
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
	core.GoGuarded("forward-listener:"+cfg.Name, func() error {
		if err := grpcServer.Serve(ln); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			return err
		}
		return nil
	}, core.LogGuardedError("forward-listener:"+cfg.Name))
	Listener = lns
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
	local := s.registry.get(listenerID, pipelineID)
	local.attach(stream)
	return local.serve(stream)
}

type forwardPipelineRPC struct {
	listenerID string
	registry   *forwardStreamRegistry
}

func (c *forwardPipelineRPC) OpenForwardStream(ctx context.Context, pipeline core.Pipeline) (core.ForwardStream, error) {
	listenerID := c.listenerID
	if pb := pipeline.ToProtobuf(); pb != nil && pb.ListenerId != "" {
		listenerID = pb.ListenerId
	}
	return c.registry.get(listenerID, pipeline.ID()), nil
}

func (c *forwardPipelineRPC) Register(ctx context.Context, in *clientpb.RegisterSession, _ ...grpc.CallOption) (*clientpb.Empty, error) {
	if in == nil {
		return nil, fmt.Errorf("register session is nil")
	}
	stream := c.registry.get(in.ListenerId, in.PipelineId)
	return &clientpb.Empty{}, stream.sendEvent(&clientpb.SpiteRequest{
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
	sessionID, _ := metadataValue(ctx, "session_id")
	listenerID, _ := metadataValue(ctx, "listener_id")
	pipelineID, _ := metadataValue(ctx, "pipeline_id")
	if listenerID == "" {
		listenerID = c.listenerID
	}
	stream := c.registry.get(listenerID, pipelineID)
	return &clientpb.Empty{}, stream.sendEvent(&clientpb.SpiteRequest{
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

type forwardStreamRegistry struct {
	mu      sync.Mutex
	streams map[string]*forwardLocalStream
}

func newForwardStreamRegistry() *forwardStreamRegistry {
	return &forwardStreamRegistry{streams: make(map[string]*forwardLocalStream)}
}

func (r *forwardStreamRegistry) get(listenerID, pipelineID string) *forwardLocalStream {
	key := core.PipelineRuntimeKey(listenerID, pipelineID)
	r.mu.Lock()
	defer r.mu.Unlock()
	if stream := r.streams[key]; stream != nil {
		return stream
	}
	stream := &forwardLocalStream{
		listenerID: listenerID,
		pipelineID: pipelineID,
		requests:   make(chan *clientpb.SpiteRequest, 255),
		events:     make(chan *clientpb.SpiteRequest, 255),
	}
	r.streams[key] = stream
	return stream
}

type forwardLocalStream struct {
	listenerID string
	pipelineID string
	requests   chan *clientpb.SpiteRequest
	events     chan *clientpb.SpiteRequest
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

func (s *forwardLocalStream) Recv() (*clientpb.SpiteRequest, error) {
	req, ok := <-s.requests
	if !ok {
		return nil, io.EOF
	}
	return req, nil
}

func (s *forwardLocalStream) attach(_ forwardrpc.ForwardListener_TaskStreamServer) {}

func (s *forwardLocalStream) serve(stream forwardrpc.ForwardListener_TaskStreamServer) error {
	errCh := make(chan error, 2)
	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				errCh <- err
				return
			}
			s.requests <- req
		}
	}()
	go func() {
		for event := range s.events {
			if err := stream.Send(event); err != nil {
				errCh <- err
				return
			}
		}
	}()
	err := <-errCh
	if err == io.EOF {
		return nil
	}
	return err
}

func (s *forwardLocalStream) sendEvent(event *clientpb.SpiteRequest) error {
	if event == nil {
		return nil
	}
	select {
	case s.events <- event:
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("forward stream %s:%s event queue full", s.listenerID, s.pipelineID)
	}
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
