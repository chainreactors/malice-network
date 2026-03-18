package core

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	"github.com/mark3labs/mcp-go/server"
)

// extractTextContent extracts the text from a CallToolResult.
func extractTextContent(result *mcp.CallToolResult) (string, error) {
	var b strings.Builder
	for _, content := range result.Content {
		tc, ok := content.(mcp.TextContent)
		if !ok {
			return "", fmt.Errorf("unsupported content type: %T", content)
		}
		b.WriteString(tc.Text)
	}
	if result.IsError {
		return "", fmt.Errorf("tool error: %s", b.String())
	}
	return b.String(), nil
}

func TestMCPToolRegistration(t *testing.T) {
	ctx := context.Background()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool: mcp.NewTool("test_tool",
			mcp.WithDescription("A test tool"),
			mcp.WithString("name", mcp.Required(), mcp.Description("The name")),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("ok"), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	result, err := srv.Client().ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatal("ListTools:", err)
	}

	if len(result.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result.Tools))
	}

	tool := result.Tools[0]
	if tool.Name != "test_tool" {
		t.Errorf("tool name = %q, want %q", tool.Name, "test_tool")
	}
	if tool.Description != "A test tool" {
		t.Errorf("tool description = %q, want %q", tool.Description, "A test tool")
	}
}

func TestMCPToolCallWithRequireString(t *testing.T) {
	ctx := context.Background()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool: mcp.NewTool("echo",
			mcp.WithDescription("Echo command"),
			mcp.WithString("command", mcp.Required(), mcp.Description("Command to echo")),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			command, err := request.RequireString("command")
			if err != nil {
				return mcp.NewToolResultError("command is required"), nil
			}
			return mcp.NewToolResultText("echo: " + command), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	t.Run("valid_args", func(t *testing.T) {
		var req mcp.CallToolRequest
		req.Params.Name = "echo"
		req.Params.Arguments = map[string]any{"command": "hello"}

		result, err := srv.Client().CallTool(ctx, req)
		if err != nil {
			t.Fatal("CallTool:", err)
		}

		got, err := extractTextContent(result)
		if err != nil {
			t.Fatal(err)
		}
		if got != "echo: hello" {
			t.Errorf("got %q, want %q", got, "echo: hello")
		}
	})

	t.Run("missing_required_arg", func(t *testing.T) {
		var req mcp.CallToolRequest
		req.Params.Name = "echo"
		req.Params.Arguments = map[string]any{}

		result, err := srv.Client().CallTool(ctx, req)
		if err != nil {
			t.Fatal("CallTool:", err)
		}

		if !result.IsError {
			t.Error("expected error result for missing required arg")
		}
	})
}

func TestMCPToolCallWithRequireFloat(t *testing.T) {
	ctx := context.Background()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool: mcp.NewTool("get_task",
			mcp.WithDescription("Get task by ID"),
			mcp.WithNumber("task_id", mcp.Required(), mcp.Description("Task ID")),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			taskID, err := request.RequireFloat("task_id")
			if err != nil {
				return mcp.NewToolResultError("task_id is required"), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("task_%d", uint32(taskID))), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	var req mcp.CallToolRequest
	req.Params.Name = "get_task"
	req.Params.Arguments = map[string]any{"task_id": float64(42)}

	result, err := srv.Client().CallTool(ctx, req)
	if err != nil {
		t.Fatal("CallTool:", err)
	}

	got, err := extractTextContent(result)
	if err != nil {
		t.Fatal(err)
	}
	if got != "task_42" {
		t.Errorf("got %q, want %q", got, "task_42")
	}
}

func TestMCPToolCallWithOptionalArgs(t *testing.T) {
	ctx := context.Background()

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool: mcp.NewTool("cmd",
			mcp.WithDescription("Command with optional session"),
			mcp.WithString("command", mcp.Required(), mcp.Description("Command")),
			mcp.WithString("session_id", mcp.Description("Optional session ID")),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			command, err := request.RequireString("command")
			if err != nil {
				return mcp.NewToolResultError("command is required"), nil
			}
			sessionID, _ := request.GetArguments()["session_id"].(string)
			if sessionID != "" {
				return mcp.NewToolResultText(fmt.Sprintf("[%s] %s", sessionID, command)), nil
			}
			return mcp.NewToolResultText(command), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	t.Run("without_optional", func(t *testing.T) {
		var req mcp.CallToolRequest
		req.Params.Name = "cmd"
		req.Params.Arguments = map[string]any{"command": "whoami"}

		result, err := srv.Client().CallTool(ctx, req)
		if err != nil {
			t.Fatal("CallTool:", err)
		}

		got, err := extractTextContent(result)
		if err != nil {
			t.Fatal(err)
		}
		if got != "whoami" {
			t.Errorf("got %q, want %q", got, "whoami")
		}
	})

	t.Run("with_optional", func(t *testing.T) {
		var req mcp.CallToolRequest
		req.Params.Name = "cmd"
		req.Params.Arguments = map[string]any{
			"command":    "whoami",
			"session_id": "sess-123",
		}

		result, err := srv.Client().CallTool(ctx, req)
		if err != nil {
			t.Fatal("CallTool:", err)
		}

		got, err := extractTextContent(result)
		if err != nil {
			t.Fatal(err)
		}
		want := "[sess-123] whoami"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestMCPPromptRegistration(t *testing.T) {
	ctx := context.Background()

	srv := mcptest.NewUnstartedServer(t)
	srv.AddPrompt(
		mcp.NewPrompt("greeting", mcp.WithPromptDescription("A greeting prompt")),
		func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			return mcp.NewGetPromptResult(
				"Greeting",
				[]mcp.PromptMessage{
					mcp.NewPromptMessage(mcp.RoleAssistant, mcp.NewTextContent("Hello!")),
				},
			), nil
		},
	)
	if err := srv.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	// List prompts
	listResult, err := srv.Client().ListPrompts(ctx, mcp.ListPromptsRequest{})
	if err != nil {
		t.Fatal("ListPrompts:", err)
	}
	if len(listResult.Prompts) != 1 {
		t.Fatalf("expected 1 prompt, got %d", len(listResult.Prompts))
	}
	if listResult.Prompts[0].Name != "greeting" {
		t.Errorf("prompt name = %q, want %q", listResult.Prompts[0].Name, "greeting")
	}

	// Get prompt
	var getReq mcp.GetPromptRequest
	getReq.Params.Name = "greeting"
	getResult, err := srv.Client().GetPrompt(ctx, getReq)
	if err != nil {
		t.Fatal("GetPrompt:", err)
	}
	if len(getResult.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(getResult.Messages))
	}
	tc, ok := getResult.Messages[0].Content.(mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if tc.Text != "Hello!" {
		t.Errorf("prompt text = %q, want %q", tc.Text, "Hello!")
	}
}

func TestMCPResourceRegistration(t *testing.T) {
	ctx := context.Background()

	srv := mcptest.NewUnstartedServer(t)
	srv.AddResource(
		mcp.Resource{
			URI:         "test://sessions",
			Name:        "sessions",
			Description: "List all sessions",
			MIMEType:    "text/plain",
		},
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      request.Params.URI,
					MIMEType: "text/plain",
					Text:     "session-1\nsession-2",
				},
			}, nil
		},
	)
	if err := srv.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	// List resources
	listResult, err := srv.Client().ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil {
		t.Fatal("ListResources:", err)
	}
	if len(listResult.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(listResult.Resources))
	}
	if listResult.Resources[0].URI != "test://sessions" {
		t.Errorf("resource URI = %q, want %q", listResult.Resources[0].URI, "test://sessions")
	}

	// Read resource
	var readReq mcp.ReadResourceRequest
	readReq.Params.URI = "test://sessions"
	readResult, err := srv.Client().ReadResource(ctx, readReq)
	if err != nil {
		t.Fatal("ReadResource:", err)
	}
	if len(readResult.Contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(readResult.Contents))
	}
	trc, ok := readResult.Contents[0].(mcp.TextResourceContents)
	if !ok {
		t.Fatalf("expected TextResourceContents, got %T", readResult.Contents[0])
	}
	if !strings.Contains(trc.Text, "session-1") {
		t.Errorf("resource text = %q, want to contain %q", trc.Text, "session-1")
	}
}

func TestMCPSSEServerStartStop(t *testing.T) {
	s := server.NewMCPServer("test-server", "1.0.0")
	s.AddTool(
		mcp.NewTool("ping", mcp.WithDescription("Ping")),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("pong"), nil
		},
	)

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	sseServer := server.NewSSEServer(s,
		server.WithBaseURL(fmt.Sprintf("http://%s/mcp", addr)),
	)

	// Start server
	errCh := make(chan error, 1)
	go func() {
		if err := sseServer.Start(addr); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	// Wait for server to be ready
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Verify server is listening
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatalf("server not listening: %v", err)
	}
	conn.Close()

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sseServer.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	// Verify server stopped
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("server error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop in time")
	}
}
