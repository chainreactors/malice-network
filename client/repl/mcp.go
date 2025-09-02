package repl

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"net/http"
	"strings"
)

// MCPServer 包装了MCP服务器实例
type MCPServer struct {
	server *server.MCPServer
	srv    *http.Server
}

// NewMCPServer 创建一个新的MCP服务器实例
func (c *Console) NewMCPServer(cmds map[string]*cobra.Command) {
	s := server.NewMCPServer(
		"Malice Network C2 Client",
		"1.0.0",
	)

	s.AddPrompt(mcp.NewPrompt("greeting",
		mcp.WithPromptDescription("A friendly greeting prompt"),
	), func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {

		return mcp.NewGetPromptResult(
			"A friendly greeting",
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(
					mcp.RoleAssistant,
					mcp.NewTextContent(fmt.Sprintf("Hello, This is IoM! How can I help you today?")),
				),
				mcp.NewPromptMessage(
					mcp.RoleUser,
					mcp.NewTextContent(fmt.Sprintf("IoM is a feature-rich and highly flexible C2 framework that provides a server for data processing and interactive services, a listener for forward and reverse connections, and a client for user-friendly operations. Its modular design and plug-in compatibility make it easy for users to customize and expand tool functions during red team testing and post-penetration phases to adapt to different attack scenarios and target environments. Official wiki: https://chainreactors.github.io/wiki/IoM."))),
			},
		), nil
	})

	s.AddPrompt(mcp.NewPrompt("c2_command_execution",
		mcp.WithPromptDescription("Command and Control assistance"),
	), func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {

		return mcp.NewGetPromptResult(
			"Command and Control assistance",
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(
					mcp.RoleUser,
					mcp.NewTextContent(fmt.Sprintf("All tool command need arguments in JSON format, such as: {\"cmdline\": \"command\"}")),
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
	})

	// 注册所有命令

	c.MCP = &MCPServer{
		server: s,
		srv:    &http.Server{
			//ReadTimeout:  60 * time.Second,
			//WriteTimeout: 60 * time.Second,
		},
	}
	c.registerCommands(cmds)
}

// registerCommands 注册所有命令
func (c *Console) registerCommands(cmds map[string]*cobra.Command) {
	for _, cobraCmd := range cmds {
		// 注册命令
		c.registerCobraCommands(cobraCmd, "")
	}
}

// registerCobraCommands 递归注册cobra命令
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
	// 注册当前命令

	if cmd.Annotations["isStatic"] != "true" && cmd.Annotations["resource"] != "true" {
		// 为每个命令创建工具
		toolDescription := generateCommandDoc(cmd)
		if cmd.GroupID == consts.ImplantGroup || cmd.GroupID == consts.ExecuteGroup ||
			cmd.GroupID == consts.SysGroup || cmd.GroupID == consts.FileGroup {
			toolDescription = toolDescription + " (Implant)"
		}
		tool := mcp.NewTool(toolName, mcp.WithDescription(toolDescription),
			mcp.WithString("cmdline",
				mcp.Required(),
				mcp.Description("Command line to execute")))
		c.MCP.server.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			response := ""
			var err error

			// 检查参数是否存在
			if request.Params.Name == "" || request.Params.Arguments == nil {
				// 如果没有参数，返回命令文档
				return mcp.NewToolResultText(toolDescription), nil
			}

			// 获取命令参数
			cmdLine, ok := request.Params.Arguments["cmdline"].(string)
			if !ok {
				// 如果参数类型不正确，返回命令文档
				return mcp.NewToolResultText(toolDescription), nil
			}

			response, err = RunCommand(c, cmdLine)
			if err != nil {
				logs.Log.Errorf("Error executing command: %v", err)
			}

			if err != nil {
				return nil, err
			}

			if response != "" {
				return mcp.NewToolResultText(response), nil
			}

			return mcp.NewToolResultText(toolDescription), nil
		})
	} else if cmd.Annotations["resource"] == "true" {
		// 为静态命令创建资源
		resource := mcp.Resource{
			URI:         fmt.Sprintf("iom://%s", cmdPath),
			Name:        cmdPath,
			Description: cmd.Short,
			MIMEType:    "text/plain",
		}

		// 注册资源处理器
		c.MCP.server.AddResource(resource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			response := ""
			var err error

			cmdLine := ""
			if cmd.Use == consts.CommandSession {
				cmdLine = cmdPath + " --all " + "--static"
			} else if parentPath == consts.CommandArtifact {
				cmdLine = cmdPath + " " + "--static"
			} else {
				cmdLine = cmdPath
			}
			response, err = RunCommand(c, cmdLine)
			if err != nil {
				logs.Log.Errorf("Error executing command: %v", err)
			}

			if err != nil {
				return nil, err
			}

			if response != "" {
				return []mcp.ResourceContents{
					mcp.TextResourceContents{
						URI:      request.Params.URI,
						MIMEType: "text/plain",
						Text:     response,
					},
				}, nil
			}

			// 如果没有响应，返回命令文档
			doc := generateCommandDoc(cmd)
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      request.Params.URI,
					MIMEType: "text/plain",
					Text:     doc,
				},
			}, nil
		})
	}

	// 递归注册子命令
	for _, subCmd := range cmd.Commands() {
		c.registerCobraCommands(subCmd, cmdPath)
	}
}

// generateCommandDoc 生成详细的命令文档
func generateCommandDoc(cmd *cobra.Command) string {
	var doc strings.Builder

	GenMarkdownCustom(cmd, &doc, func(s string) string {
		return s
	})
	return doc.String()
}

// Start 启动MCP HTTP和gRPC服务器
func (m *MCPServer) Start(host string, port int) error {
	// 创建HTTP处理器
	sse := server.NewSSEServer(m.server, server.WithBaseURL("http://"+host+":"+fmt.Sprintf("%d", port)+"/mcp"),
		server.WithHTTPServer(m.srv))

	// 启动HTTP服务器
	go func() {
		m.srv.Addr = fmt.Sprintf("%s:%d", host, port)
		m.srv.Handler = sse
		err := m.srv.ListenAndServe()
		if err != nil {
			logs.Log.Errorf("Failed to start MCP server: %v\n", err)
			return
		}
		//sse.Start(":" + fmt.Sprintf("%d", port))
		logs.Log.Infof("Starting MCP HTTP server on %s:%d\n", host, port)
	}()
	return nil
}

// Stop 停止MCP服务器
func (m *MCPServer) Stop() error {
	if m.srv != nil {
		if err := m.srv.Shutdown(context.Background()); err != nil {
			return err
		}
	}
	return nil
}

// AddTool 添加新的工具到MCP服务器
func (m *MCPServer) AddTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	m.server.AddTool(tool, handler)
}
