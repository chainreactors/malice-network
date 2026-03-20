package core

import (
	"context"
	"fmt"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/services/localrpc"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/kballard/go-shellquote"
	"google.golang.org/grpc"
	"net"
	"runtime/debug"
	"strings"
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

// StreamCommand executes a command and continuously streams back task event output.
// It is a general-purpose streaming RPC: any command that produces persistent EventTaskDone
// events (tapping, poison, etc.) will have its rendered output streamed to the caller.
//
// Design:
//  1. Register an EventHook BEFORE executing the command (no race window).
//  2. Execute the command via cobra; read Session.LastTask for the task ID (no polling).
//  3. EventHook filters events by task ID, renders via InternalFunctions, writes to channel.
//  4. Main loop reads channel and streams to gRPC client.
//  5. On context cancel: remove EventHook, return.
func (s *LocalRPCServer) StreamCommand(req *localrpc.ExecuteCommandRequest, stream localrpc.CommandService_StreamCommandServer) error {
	reqID := atomic.AddUint64(&localRPCRequestSeq, 1)
	client.Log.Infof("LocalRPC[%d]: StreamCommand start (session=%s, command=%q)\n", reqID, req.SessionId, req.Command)

	ch := make(chan string, 128)
	ctx := stream.Context()

	// taskID is written after command execution, read by the EventHook goroutine.
	var taskID atomic.Uint32

	// 1. Register EventHook BEFORE executing the command.
	//    This ensures zero race window — events are captured from the moment the task is created.
	//    The hook matches all task-done events; filtering by taskID happens inside.
	hookCondition := client.EventCondition{
		Type: consts.EventTask,
		Op:   consts.CtrlTaskCallback,
	}
	hookFn := client.OnEventFunc(func(event *clientpb.Event) (bool, error) {
		task := event.GetTask()
		if task == nil {
			return false, nil
		}

		// Filter: only forward events for our task on our session.
		tid := taskID.Load()
		if tid == 0 || task.TaskId != tid || task.SessionId != req.SessionId {
			return false, nil
		}

		tctx := wrapToTaskContext(event)
		fn, ok := intermediate.InternalFunctions[task.Type]
		if !ok || fn.DoneCallback == nil {
			return false, nil
		}
		formatted, err := fn.DoneCallback(tctx)
		if err != nil || formatted == "" {
			return false, nil
		}

		select {
		case ch <- formatted:
		default:
			// Drop if consumer is slow — never block the event dispatch goroutine.
		}
		return false, nil
	})
	s.console.AddEventHook(hookCondition, hookFn)
	defer s.console.removeEventHook(hookCondition, hookFn)

	// 2. Execute the command; LastTask is returned from within the lock (no race).
	syncOutput, lastTask, err := executeStreamCommand(s.console, req.Command, req.SessionId)
	if err != nil {
		client.Log.Errorf("LocalRPC[%d]: StreamCommand exec failed: %v\n", reqID, err)
		return stream.Send(&localrpc.ExecuteCommandResponse{
			Output:  syncOutput,
			Error:   err.Error(),
			Success: false,
		})
	}

	if lastTask == nil {
		client.Log.Debugf("LocalRPC[%d]: StreamCommand no task created, returning sync output\n", reqID)
		return stream.Send(&localrpc.ExecuteCommandResponse{
			Output:  syncOutput,
			Success: true,
		})
	}
	taskID.Store(lastTask.TaskId)
	client.Log.Infof("LocalRPC[%d]: StreamCommand streaming task %d (session=%s)\n",
		reqID, lastTask.TaskId, req.SessionId)

	// 3. Send initial ACK with sync output.
	if err := stream.Send(&localrpc.ExecuteCommandResponse{
		Output:  syncOutput + "\n",
		Success: true,
	}); err != nil {
		return err
	}

	// 4. Stream events until the client cancels.
	for {
		select {
		case <-ctx.Done():
			client.Log.Infof("LocalRPC[%d]: StreamCommand context cancelled\n", reqID)
			return nil
		case msg := <-ch:
			if err := stream.Send(&localrpc.ExecuteCommandResponse{
				Output:  msg + "\n",
				Success: true,
			}); err != nil {
				return err
			}
		}
	}
}

// executeStreamCommand runs a cobra command for StreamCommand.
// It acquires commandExecMu only for the duration of command execution (no polling).
// Returns the sync console output and the task created by the command (nil if none).
func executeStreamCommand(con *Console, command, sessionID string) (string, *clientpb.Task, error) {
	if command == "" {
		return "", nil, fmt.Errorf("command is required")
	}

	commandExecMu.Lock()
	defer commandExecMu.Unlock()

	restore := con.WithNonInteractiveExecution(true)
	defer restore()

	if err := switchSessionWithCallee(con, sessionID, consts.CalleeRPC); err != nil {
		return "", nil, err
	}

	// Clear LastTask so we can detect whether the command created a new one.
	sess := con.GetInteractive()
	if sess != nil {
		sess.LastTask = nil
	}

	args, err := shellquote.Split(command)
	if err != nil {
		return "", nil, err
	}
	args = stripWaitFlag(args)

	start := time.Now()
	if err := con.App.Execute(con.Context(), con.App.ActiveMenu(), args, false); err != nil {
		return "", nil, err
	}

	syncOutput := strings.TrimSpace(client.RemoveANSI(client.Stdout.Range(start, time.Now())))

	// Capture LastTask while still holding the lock.
	var task *clientpb.Task
	if sess != nil {
		task = sess.LastTask
	}
	return syncOutput, task, nil
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
