package core

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"net/http"
)

// MCPServer 包装了MCP服务器实例
type MCPServer struct {
	server *server.MCPServer
	srv    *http.Server
}

// NewMCPServer 创建一个新的MCP服务器实例
func NewMCPServer() *MCPServer {
	s := server.NewMCPServer(
		"Malice Network C2 Server",
		"1.0.0",
	)

	// 添加会话管理工具
	sessionTool := mcp.NewTool("list_sessions",
		mcp.WithDescription("List all active sessions"))
	s.AddTool(sessionTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sessions := Sessions.All()
		var result string
		for _, sess := range sessions {
			result += fmt.Sprintf("Session ID: %s, Target: %s\n",
				sess.Name, sess.Target)
		}
		if result == "" {
			result = "No active sessions found"
		}
		return mcp.NewToolResultText(result), nil
	})

	// 添加监听器管理工具
	listenerTool := mcp.NewTool("list_listeners",
		mcp.WithDescription("List all active listeners"))
	s.AddTool(listenerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var result string
		Listeners.Range(func(key, value interface{}) bool {
			listener := value.(*clientpb.Listener)
			result += fmt.Sprintf("Listener ID: %s, Ip: %s\n",
				listener.Id, listener.Ip)
			return true
		})
		if result == "" {
			result = "No active listeners found"
		}
		return mcp.NewToolResultText(result), nil
	})

	return &MCPServer{
		server: s,
	}
}

// Start 启动MCP HTTP服务器
func (m *MCPServer) Start(host string, port int) error {
	// 创建HTTP处理器
	sse := server.NewSSEServer(m.server, server.WithBaseURL("http://"+host+":"+fmt.Sprintf("%d", port)+"/mcp"))

	sse.Start(":" + fmt.Sprintf("%d", port))
	logs.Log.Infof("Starting MCP HTTP server on %s:%d", host, port)
	return nil
}

// Stop 停止MCP服务器
func (m *MCPServer) Stop() error {
	if m.srv != nil {
		return m.srv.Shutdown(context.Background())
	}
	return nil
}

// AddTool 添加新的工具到MCP服务器
func (m *MCPServer) AddTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	m.server.AddTool(tool, handler)
}
