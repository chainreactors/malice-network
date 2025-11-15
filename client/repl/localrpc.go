package repl

import (
	"context"
	"fmt"
	"net"
	"runtime/debug"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/services/localrpc"
	"google.golang.org/grpc"
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
	client.Log.Debugf("LocalRPC: ExecuteCommand called with command: %s, session_id: %s", req.Command, req.SessionId)

	// Validate request
	if req.Command == "" {
		return &localrpc.ExecuteCommandResponse{
			Output:  "",
			Error:   "command is required",
			Success: false,
		}, nil
	}

	// Execute the command using Console's method
	output, err := s.console.ExecuteCommandWithSession(req.Command, req.SessionId)
	if err != nil {
		client.Log.Errorf("LocalRPC: Error executing command: %v", err)
		return &localrpc.ExecuteCommandResponse{
			Output:  output,
			Error:   err.Error(),
			Success: false,
		}, nil
	}

	client.Log.Debugf("LocalRPC: Command executed successfully, output length: %d", len(output))

	return &localrpc.ExecuteCommandResponse{
		Output:  output,
		Error:   "",
		Success: true,
	}, nil
}

// ExecuteLua implements the CommandService.ExecuteLua RPC method
func (s *LocalRPCServer) ExecuteLua(ctx context.Context, req *localrpc.ExecuteLuaRequest) (*localrpc.ExecuteLuaResponse, error) {
	client.Log.Debugf("LocalRPC: ExecuteLua called with script length: %d, session_id: %s", len(req.Script), req.SessionId)

	// Validate request
	if req.Script == "" {
		return &localrpc.ExecuteLuaResponse{
			Output:  "",
			Error:   "script is required",
			Success: false,
		}, nil
	}

	// Execute the Lua script using Console's method
	output, err := s.console.ExecuteLuaWithSession(req.Script, req.SessionId)
	if err != nil {
		client.Log.Errorf("LocalRPC: Error executing Lua script: %v", err)
		return &localrpc.ExecuteLuaResponse{
			Output:  output,
			Error:   err.Error(),
			Success: false,
		}, nil
	}

	client.Log.Debugf("LocalRPC: Lua script executed successfully, output length: %d", len(output))

	return &localrpc.ExecuteLuaResponse{
		Output:  output,
		Error:   "",
		Success: true,
	}, nil
}

// LocalRPC wraps the gRPC server instance
type LocalRPC struct {
	server   *grpc.Server
	listener net.Listener
	address  string
}

// StartLocalRPC starts the local gRPC server
func (c *Console) StartLocalRPC(address string) error {
	if c.Server == nil {
		return fmt.Errorf("server not initialized")
	}

	client.Log.Importantf("[client] starting local gRPC server on %s", address)

	ln, err := net.Listen("tcp", address)
	if err != nil {
		client.Log.Errorf("failed to listen on %s: %v", address, err)
		return err
	}

	options := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(10 * 1024 * 1024), // 10MB
		grpc.MaxSendMsgSize(10 * 1024 * 1024), // 10MB
	}

	grpcServer := grpc.NewServer(options...)
	localrpc.RegisterCommandServiceServer(grpcServer, NewLocalRPCServer(c))

	// Initialize LocalRPC wrapper
	c.LocalRPC = &LocalRPC{
		server:   grpcServer,
		listener: ln,
		address:  address,
	}

	// Start server in background
	go func() {
		panicked := true
		defer func() {
			if panicked {
				client.Log.Errorf("LocalRPC: stacktrace from panic: %s", string(debug.Stack()))
			}
		}()
		if err := grpcServer.Serve(ln); err != nil {
			client.Log.Warnf("LocalRPC: gRPC server exited with error: %v", err)
		} else {
			panicked = false
		}
	}()

	client.Log.Importantf("[client] local gRPC server started successfully on %s", address)
	return nil
}

// StopLocalRPC stops the local gRPC server
func (c *Console) StopLocalRPC() error {
	if c.LocalRPC == nil {
		return nil
	}

	client.Log.Infof("Stopping local gRPC server on %s", c.LocalRPC.address)

	if c.LocalRPC.server != nil {
		c.LocalRPC.server.GracefulStop()
	}

	if c.LocalRPC.listener != nil {
		if err := c.LocalRPC.listener.Close(); err != nil {
			return err
		}
	}

	c.LocalRPC = nil
	client.Log.Infof("Local gRPC server stopped")
	return nil
}

// InitLocalRPC initializes the local RPC server if address is provided
func (c *Console) InitLocalRPC(address string) error {
	if address == "" {
		return nil
	}

	return c.StartLocalRPC(address)
}
