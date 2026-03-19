package core

import (
	"context"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/services/localrpc"
	"google.golang.org/grpc"
	"net"
	"runtime/debug"
	"sync/atomic"
	"time"
)

var localRPCRequestSeq uint64

// LocalRPCServer wraps the gRPC server for local command execution
type LocalRPCServer struct {
	localrpc.UnimplementedCommandServiceServer
	console *Console
}

// NewLocalRPCServer creates a new LocalRPCServer instance
func NewLocalRPCServer(console *Console) *LocalRPCServer {
	return &LocalRPCServer{
		console: console,
	}
}

// ExecuteCommand implements the CommandService.ExecuteCommand RPC method
func (s *LocalRPCServer) ExecuteCommand(ctx context.Context, req *localrpc.ExecuteCommandRequest) (*localrpc.ExecuteCommandResponse, error) {
	reqID := atomic.AddUint64(&localRPCRequestSeq, 1)
	start := time.Now()
	client.Log.Debugf("LocalRPC[%d]: ExecuteCommand start (session=%s, command=%q)\n", reqID, req.SessionId, req.Command)

	output, err := executeRPCCommand(s.console, req.Command, req.SessionId)
	if err != nil {
		client.Log.Errorf("LocalRPC[%d]: ExecuteCommand failed after %s: %v\n", reqID, time.Since(start), err)
		return &localrpc.ExecuteCommandResponse{
			Output:  output,
			Error:   err.Error(),
			Success: false,
		}, nil
	}

	client.Log.Debugf("LocalRPC[%d]: ExecuteCommand done in %s (output_len=%d)\n", reqID, time.Since(start), len(output))
	return &localrpc.ExecuteCommandResponse{
		Output:  output,
		Error:   "",
		Success: true,
	}, nil
}

// ExecuteLua implements the CommandService.ExecuteLua RPC method
func (s *LocalRPCServer) ExecuteLua(ctx context.Context, req *localrpc.ExecuteLuaRequest) (*localrpc.ExecuteLuaResponse, error) {
	reqID := atomic.AddUint64(&localRPCRequestSeq, 1)
	start := time.Now()
	client.Log.Debugf("LocalRPC[%d]: ExecuteLua start (session=%s, script_len=%d)\n", reqID, req.SessionId, len(req.Script))

	output, err := executeLua(s.console, req.Script, req.SessionId, consts.CalleeRPC)
	if err != nil {
		client.Log.Errorf("LocalRPC[%d]: ExecuteLua failed after %s: %v\n", reqID, time.Since(start), err)
		return &localrpc.ExecuteLuaResponse{
			Output:  output,
			Error:   err.Error(),
			Success: false,
		}, nil
	}

	client.Log.Debugf("LocalRPC[%d]: ExecuteLua done in %s (output_len=%d)\n", reqID, time.Since(start), len(output))
	return &localrpc.ExecuteLuaResponse{
		Output:  output,
		Error:   "",
		Success: true,
	}, nil
}

// GetHistory implements the CommandService.GetHistory RPC method
func (s *LocalRPCServer) GetHistory(ctx context.Context, req *localrpc.GetHistoryRequest) (*localrpc.GetHistoryResponse, error) {
	client.Log.Debugf("LocalRPC: GetHistory called with task_id: %d, session_id: %s\n", req.TaskId, req.SessionId)

	output, err := getHistory(s.console, req.TaskId, req.SessionId)
	if err != nil {
		client.Log.Errorf("LocalRPC: Error getting history: %v\n", err)
		return &localrpc.GetHistoryResponse{
			Output:  "",
			Error:   err.Error(),
			Success: false,
		}, nil
	}

	client.Log.Debugf("LocalRPC: History retrieved successfully\n")
	return &localrpc.GetHistoryResponse{
		Output:  client.RemoveANSI(output),
		Error:   "",
		Success: true,
	}, nil
}

// GetSchemas implements the CommandService.GetSchemas RPC method
func (s *LocalRPCServer) GetSchemas(ctx context.Context, req *localrpc.GetSchemasRequest) (*localrpc.GetSchemasResponse, error) {
	client.Log.Debugf("LocalRPC: GetSchemas called with group: %s\n", req.Group)

	schemas, err := getSchemas(s.console, req.Group)
	if err != nil {
		client.Log.Errorf("LocalRPC: Error getting schemas: %v\n", err)
		return &localrpc.GetSchemasResponse{
			SchemasJson: "",
			Error:       err.Error(),
			Success:     false,
		}, nil
	}

	client.Log.Debugf("LocalRPC: Schemas retrieved successfully\n")
	return &localrpc.GetSchemasResponse{
		SchemasJson: schemas,
		Error:       "",
		Success:     true,
	}, nil
}

// GetGroups implements the CommandService.GetGroups RPC method
func (s *LocalRPCServer) GetGroups(ctx context.Context, req *localrpc.GetGroupsRequest) (*localrpc.GetGroupsResponse, error) {
	client.Log.Debugf("LocalRPC: GetGroups called\n")

	groups, err := getGroups(s.console)
	if err != nil {
		client.Log.Errorf("LocalRPC: Error getting groups: %v\n", err)
		return &localrpc.GetGroupsResponse{
			Groups:  nil,
			Error:   err.Error(),
			Success: false,
		}, nil
	}

	client.Log.Debugf("LocalRPC: Groups retrieved successfully, count: %d\n", len(groups))
	return &localrpc.GetGroupsResponse{
		Groups:  groups,
		Error:   "",
		Success: true,
	}, nil
}

// SearchCommands implements the CommandService.SearchCommands RPC method
func (s *LocalRPCServer) SearchCommands(ctx context.Context, req *localrpc.SearchCommandsRequest) (*localrpc.SearchCommandsResponse, error) {
	client.Log.Debugf("LocalRPC: SearchCommands called with query: %s, group: %s, session: %s\n", req.Query, req.Group, req.SessionId)

	commands, err := searchCommands(s.console, req.Query, req.Group, req.SessionId)
	if err != nil {
		client.Log.Errorf("LocalRPC: Error searching commands: %v\n", err)
		return &localrpc.SearchCommandsResponse{
			Commands: nil,
			Error:    err.Error(),
			Success:  false,
		}, nil
	}

	client.Log.Debugf("LocalRPC: SearchCommands found %d results\n", len(commands))
	return &localrpc.SearchCommandsResponse{
		Commands: commands,
		Error:    "",
		Success:  true,
	}, nil
}

// LocalRPC wraps the gRPC server instance
type LocalRPC struct {
	server   *grpc.Server
	listener net.Listener
	address  string
	console  *Console
}

// NewLocalRPC creates and starts a new LocalRPC server
func NewLocalRPC(console *Console, address string) (*LocalRPC, error) {
	if address == "" {
		return nil, nil
	}

	ln, err := net.Listen("tcp", address)
	if err != nil {
		client.Log.Errorf("failed to listen on %s: %v\n", address, err)
		return nil, err
	}

	options := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(10 * 1024 * 1024),
		grpc.MaxSendMsgSize(10 * 1024 * 1024),
	}

	grpcServer := grpc.NewServer(options...)
	localrpc.RegisterCommandServiceServer(grpcServer, NewLocalRPCServer(console))

	rpc := &LocalRPC{
		server:   grpcServer,
		listener: ln,
		address:  address,
		console:  console,
	}

	go func() {
		panicked := true
		defer func() {
			if panicked {
				client.Log.Errorf("LocalRPC: stacktrace from panic: %s\n", string(debug.Stack()))
			}
		}()
		if err := grpcServer.Serve(ln); err != nil {
			client.Log.Warnf("LocalRPC: gRPC server exited with error: %v\n", err)
		} else {
			panicked = false
		}
	}()

	return rpc, nil
}

// Stop stops the local gRPC server
func (l *LocalRPC) Stop() error {
	if l == nil {
		return nil
	}

	client.Log.Infof("Stopping local gRPC server on %s\n", l.address)

	if l.server != nil {
		l.server.GracefulStop()
	}

	if l.listener != nil {
		if err := l.listener.Close(); err != nil {
			return err
		}
	}

	client.Log.Infof("Local gRPC server stopped\n")
	return nil
}
