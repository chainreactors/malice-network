package repl

import (
	"context"
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"net/http"
	"strings"

	"github.com/chainreactors/logs"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

// MCPServer 包装了MCP服务器实例
type MCPServer struct {
	server    *server.MCPServer
	sseServer *server.SSEServer
}

// NewMCPServer 创建一个新的MCP服务器实例
func (c *Console) NewMCPServer() {
	s := server.NewMCPServer(
		"Malice Network C2 Client",
		"1.0.0",
	)

	// 注册提示词
	c.registerPrompts(s)

	// 初始化 MCP 服务器
	c.MCP = &MCPServer{
		server: s,
	}
	c.registerCobraCommands(c.App.Menu("client").Command, "")
	c.registerCobraCommands(c.App.Menu("implant").Command, "")
}

// registerPrompts 注册 MCP 提示词
func (c *Console) registerPrompts(s *server.MCPServer) {
	// 问候提示词
	s.AddPrompt(
		mcp.NewPrompt("greeting", mcp.WithPromptDescription("A friendly greeting prompt")),
		func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			return mcp.NewGetPromptResult(
				"A friendly greeting",
				[]mcp.PromptMessage{
					mcp.NewPromptMessage(
						mcp.RoleAssistant,
						mcp.NewTextContent("Hello, This is IoM! How can I help you today?"),
					),
					mcp.NewPromptMessage(
						mcp.RoleUser,
						mcp.NewTextContent("IoM is a feature-rich and highly flexible C2 framework that provides a server for data processing and interactive services, a listener for forward and reverse connections, and a client for user-friendly operations. Its modular design and plug-in compatibility make it easy for users to customize and expand tool functions during red team testing and post-penetration phases to adapt to different attack scenarios and target environments. Official wiki: https://chainreactors.github.io/wiki/IoM."),
					),
				},
			), nil
		},
	)

	// C2 命令执行提示词
	s.AddPrompt(
		mcp.NewPrompt("c2_command_execution", mcp.WithPromptDescription("Command and Control assistance")),
		func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			return mcp.NewGetPromptResult(
				"Command and Control assistance",
				[]mcp.PromptMessage{
					mcp.NewPromptMessage(
						mcp.RoleUser,
						mcp.NewTextContent(`All tool command need arguments in JSON format, such as: {"cmdline": "command"}`),
					),
					mcp.NewPromptMessage(
						mcp.RoleUser,
						mcp.NewTextContent(`If the tool description contains the (implant) mark, you need to judge it like this!
1. Whether use tool is used in the previous operation
2. If not, you need to first obtain the session through the session resource of resource, bring --use sessionID in the necessary parameters, and enter implant mode
3. If you need to switch sessions, bring --use sessionID in the necessary parameters
4. All tools with the (implant) mark in the necessary parameters must include --wait, unless the tool is use.`),
					),
				},
			), nil
		},
	)
}

// registerCobraCommands 递归注册 cobra 命令为 MCP 工具或资源
func (c *Console) registerCobraCommands(cmd *cobra.Command, parentPath string) {
	// 跳过隐藏命令
	if cmd.Hidden {
		return
	}

	// 构建完整的命令路径
	cmdPath := cmd.Use
	if parentPath != "" {
		cmdPath = parentPath + " " + cmdPath
	}
	toolName := strings.Replace(cmd.CommandPath(), "client implant ", "", 1)

	// 根据注解类型注册命令
	if cmd.Annotations["isStatic"] != "true" && cmd.Annotations["resource"] != "true" {
		c.registerTool(cmd, toolName, cmdPath)
	} else if cmd.Annotations["resource"] == "true" {
		c.registerResource(cmd, cmdPath, parentPath)
	}

	// 递归注册子命令
	for _, subCmd := range cmd.Commands() {
		c.registerCobraCommands(subCmd, cmdPath)
	}
}

// registerTool 注册命令为 MCP 工具
func (c *Console) registerTool(cmd *cobra.Command, toolName, cmdPath string) {
	toolDescription := generateCommandDoc(cmd)

	// 为 Implant 相关命令添加标记
	if cmd.GroupID == consts.ImplantGroup || cmd.GroupID == consts.ExecuteGroup ||
		cmd.GroupID == consts.SysGroup || cmd.GroupID == consts.FileGroup {
		toolDescription = toolDescription + " (Implant)"
	}

	tool := mcp.NewTool(
		toolName,
		mcp.WithDescription(toolDescription),
		mcp.WithString("cmdline", mcp.Required(), mcp.Description("Command line to execute")),
	)

	c.MCP.server.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// 检查参数是否存在
		if request.Params.Name == "" || request.Params.Arguments == nil {
			return mcp.NewToolResultText(toolDescription), nil
		}

		// 获取命令参数
		cmdLine, ok := request.Params.Arguments["cmdline"].(string)
		if !ok {
			return mcp.NewToolResultText(toolDescription), nil
		}

		// 执行命令
		response, err := RunCommand(c, cmdLine)
		if err != nil {
			logs.Log.Errorf("Error executing command: %v", err)
			return nil, err
		}

		if response != "" {
			return mcp.NewToolResultText(response), nil
		}

		return mcp.NewToolResultText(toolDescription), nil
	})
}

// registerResource 注册命令为 MCP 资源
func (c *Console) registerResource(cmd *cobra.Command, cmdPath, parentPath string) {
	resource := mcp.Resource{
		URI:         fmt.Sprintf("iom://%s", cmdPath),
		Name:        cmdPath,
		Description: cmd.Short,
		MIMEType:    "text/plain",
	}

	c.MCP.server.AddResource(resource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// 构建命令行
		cmdLine := buildResourceCommandLine(cmd, cmdPath, parentPath)

		// 执行命令
		response, err := RunCommand(c, cmdLine)
		if err != nil {
			logs.Log.Errorf("Error executing command: %v", err)
			return nil, err
		}

		// 返回响应或文档
		text := response
		if text == "" {
			text = generateCommandDoc(cmd)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "text/plain",
				Text:     text,
			},
		}, nil
	})
}

// buildResourceCommandLine 构建资源命令行
func buildResourceCommandLine(cmd *cobra.Command, cmdPath, parentPath string) string {
	if cmd.Use == consts.CommandSession {
		return cmdPath + " --all --static"
	} else if parentPath == consts.CommandArtifact {
		return cmdPath + " --static"
	}
	return cmdPath
}

// generateCommandDoc 生成详细的命令文档
func generateCommandDoc(cmd *cobra.Command) string {
	var doc strings.Builder
	GenMarkdownCustom(cmd, &doc, func(s string) string {
		return s
	})
	return doc.String()
}

// Start 启动 MCP HTTP 服务器
func (m *MCPServer) Start(host string, port int) error {
	// 创建 SSE 服务器，让它自己管理 HTTP 服务器
	m.sseServer = server.NewSSEServer(
		m.server,
		server.WithBaseURL(fmt.Sprintf("http://%s:%d/mcp", host, port)),
	)

	// 在后台启动服务器
	go func() {
		addr := fmt.Sprintf("%s:%d", host, port)
		if err := m.sseServer.Start(addr); err != nil && err != http.ErrServerClosed {
			logs.Log.Errorf("Failed to start MCP server: %v\n", err)
		}
	}()

	return nil
}

// Stop 停止 MCP 服务器
func (m *MCPServer) Stop() error {
	if m.sseServer != nil {
		return m.sseServer.Shutdown(context.Background())
	}
	return nil
}

// AddTool 添加新的工具到 MCP 服务器
func (m *MCPServer) AddTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	m.server.AddTool(tool, handler)
}
