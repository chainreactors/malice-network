package core

import (
	"context"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/services/localrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestLocalRPCExecuteCommand 测试通过 gRPC 执行命令
func TestLocalRPCExecuteCommand(t *testing.T) {
	// 连接到 Local RPC 服务器
	conn, err := grpc.Dial("127.0.0.1:15004",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	//grpc.WithTimeout(10*time.Second))
	if err != nil {
		t.Fatalf("Failed to connect to Local RPC server: %v", err)
	}
	defer conn.Close()

	client := localrpc.NewCommandServiceClient(conn)

	// 测试 1: 获取 session 列表
	t.Run("GetSessions", func(t *testing.T) {
		req := &localrpc.ExecuteCommandRequest{
			Command: "session --static",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resp, err := client.ExecuteCommand(ctx, req)
		if err != nil {
			t.Fatalf("ExecuteCommand failed: %v", err)
		}

		if !resp.Success {
			t.Fatalf("Command failed: %s", resp.Error)
		}

		t.Logf("Sessions output:\n%s", resp.Output)
	})

	// 测试 2: 在指定 session 中执行 whoami
	t.Run("ExecuteWhoami", func(t *testing.T) {
		// 首先获取一个 session ID
		// 这里假设你已经有一个 session，需要替换为实际的 session ID
		sessionID := "08d6c05a21512a79a1dfeb9d2a8f262f" // 从上一个测试的输出中获取

		req := &localrpc.ExecuteCommandRequest{
			Command:   "whoami --wait",
			SessionId: sessionID,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := client.ExecuteCommand(ctx, req)
		if err != nil {
			t.Fatalf("ExecuteCommand failed: %v", err)
		}

		if !resp.Success {
			t.Fatalf("Command failed: %s", resp.Error)
		}

		t.Logf("Whoami output:\n%s", resp.Output)
	})
}

// TestLocalRPCExecuteLua 测试通过 gRPC 执行 Lua 脚本
func TestLocalRPCExecuteLua(t *testing.T) {
	// 连接到 Local RPC 服务器
	conn, err := grpc.Dial("127.0.0.1:15004",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("Failed to connect to Local RPC server: %v", err)
	}
	defer conn.Close()

	client := localrpc.NewCommandServiceClient(conn)

	// 测试简单的 Lua 脚本
	t.Run("SimpleLuaScript", func(t *testing.T) {
		req := &localrpc.ExecuteLuaRequest{
			Script: `return "Hello from Lua!"`,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resp, err := client.ExecuteLua(ctx, req)
		if err != nil {
			t.Fatalf("ExecuteLua failed: %v", err)
		}

		if !resp.Success {
			t.Fatalf("Lua script failed: %s", resp.Error)
		}

		t.Logf("Lua output:\n%s", resp.Output)
	})
}

// TestLocalRPCGetHistory 测试通过 gRPC 获取历史记录
func TestLocalRPCGetHistory(t *testing.T) {
	conn, err := grpc.Dial("127.0.0.1:15004",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("Failed to connect to Local RPC server: %v", err)
	}
	defer conn.Close()

	client := localrpc.NewCommandServiceClient(conn)

	t.Run("GetTaskHistory", func(t *testing.T) {
		req := &localrpc.GetHistoryRequest{
			TaskId:    786,
			SessionId: "08d6c05a21512a79a1dfeb9d2a8f262f",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resp, err := client.GetHistory(ctx, req)
		if err != nil {
			t.Fatalf("GetHistory failed: %v", err)
		}

		if !resp.Success {
			t.Fatalf("GetHistory failed: %s", resp.Error)
		}

		t.Logf("History output:\n%s", resp.Output)
	})
}
