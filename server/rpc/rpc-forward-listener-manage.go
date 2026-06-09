package rpc

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (rpc *Server) ConnectForwardListener(ctx context.Context, req *clientpb.ForwardListenerConnect) (*clientpb.ForwardListenerStatus, error) {
	if err := requireAdminRole(ctx); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "forward listener connect request is nil")
	}
	listenerID := strings.TrimSpace(req.GetListenerId())
	if listenerID == "" {
		return nil, status.Error(codes.InvalidArgument, "listener_id is required")
	}
	host := strings.TrimSpace(req.GetConnectHost())
	if host == "" {
		return nil, status.Error(codes.InvalidArgument, "connect_host is required")
	}
	if req.GetConnectPort() > uint32(^uint16(0)) {
		return nil, status.Errorf(codes.InvalidArgument, "connect_port must be between 0 and %d", ^uint16(0))
	}
	if _, err := loadForwardListenerOperator(listenerID); err != nil {
		return nil, err
	}
	port := uint16(req.GetConnectPort())
	if port == 0 {
		port = 5005
	}
	timeout := time.Duration(req.GetTimeoutSeconds()) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	cfg := &configs.ListenerConfig{
		Enable:    true,
		Name:      listenerID,
		IP:        host,
		Transport: configs.ListenerTransportForward,
		Forward: &configs.ForwardListenerConfig{
			ConnectHost: host,
			ConnectPort: port,
		},
	}
	runtime, err := startForwardListenerClient(ctx, cfg, timeout)
	if err != nil {
		return nil, forwardListenerRPCError(err)
	}
	return runtime.toProto(), nil
}

func (rpc *Server) DisconnectForwardListener(ctx context.Context, req *clientpb.Listener) (*clientpb.ForwardListenerStatus, error) {
	if err := requireAdminRole(ctx); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listener request is nil")
	}
	reply, err := stopForwardListenerClient(req.GetId())
	if err != nil {
		return nil, err
	}
	return reply, nil
}

func (rpc *Server) GetForwardListenerStatus(ctx context.Context, req *clientpb.Listener) (*clientpb.ForwardListenerStatus, error) {
	if err := requireAdminRole(ctx); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listener request is nil")
	}
	listenerID := strings.TrimSpace(req.GetId())
	if listenerID == "" {
		return nil, status.Error(codes.InvalidArgument, "listener_id is required")
	}
	if runtime, ok := getForwardListenerRuntime(listenerID); ok {
		return runtime.toProto(), nil
	}
	if _, err := loadForwardListenerOperator(listenerID); err != nil {
		return nil, err
	}
	return inactiveForwardListenerStatus(listenerID), nil
}

func (rpc *Server) ListForwardListeners(ctx context.Context, req *clientpb.Empty) (*clientpb.ForwardListenerStatuses, error) {
	if err := requireAdminRole(ctx); err != nil {
		return nil, err
	}
	reply := &clientpb.ForwardListenerStatuses{}
	forwardListenerRuntimes.Range(func(_, value interface{}) bool {
		reply.Listeners = append(reply.Listeners, value.(*forwardListenerRuntime).toProto())
		return true
	})
	return reply, nil
}

func forwardListenerRPCError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, errForwardListenerAlreadyConnected) {
		return status.Error(codes.AlreadyExists, err.Error())
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	return status.Error(codes.Unavailable, err.Error())
}
