package core

import (
	"context"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/services/localrpc"
	"google.golang.org/grpc"
	"net"
	"runtime/debug"
)

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
	client.Log.Debugf("LocalRPC: ExecuteCommand called with command: %s, session_id: %s\n", req.Command, req.SessionId)

	output, err := executeCommand(s.console, req.Command, req.SessionId, consts.CalleeRPC)
	if err != nil {
		client.Log.Errorf("LocalRPC: Error executing command: %v\n", err)
		return &localrpc.ExecuteCommandResponse{
			Output:  output,
			Error:   err.Error(),
			Success: false,
		}, nil
	}

	client.Log.Debugf("LocalRPC: Command executed successfully, output length: %d\n", len(output))
	return &localrpc.ExecuteCommandResponse{
		Output:  output,
		Error:   "",
		Success: true,
	}, nil
}

// ExecuteLua implements the CommandService.ExecuteLua RPC method
func (s *LocalRPCServer) ExecuteLua(ctx context.Context, req *localrpc.ExecuteLuaRequest) (*localrpc.ExecuteLuaResponse, error) {
	client.Log.Debugf("LocalRPC: ExecuteLua called with script length: %d, session_id: %s\n", len(req.Script), req.SessionId)

	output, err := executeLua(s.console, req.Script, req.SessionId, consts.CalleeRPC)
	if err != nil {
		client.Log.Errorf("LocalRPC: Error executing Lua script: %v\n", err)
		return &localrpc.ExecuteLuaResponse{
			Output:  output,
			Error:   err.Error(),
			Success: false,
		}, nil
	}

	client.Log.Debugf("LocalRPC: Lua script executed successfully, output length: %d\n", len(output))
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
