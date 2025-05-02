package repl

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"net/http"
	"strings"
	"sync"
)

// MCPServer 包装了MCP服务器实例
type MCPServer struct {
	server *server.MCPServer
	srv    *http.Server
	cmds   map[string]*cobra.Command
}

var (
	clientStates = make(map[string]int) // key: clientID, value: call count
	mu           sync.Mutex
)

// 从请求中提取客户端唯一标识（示例，需根据实际协议调整）
func getClientID(ctx context.Context) string {
	// 如果是 HTTP 请求，可从 Header 或 RemoteAddr 获取
	// 示例：返回固定值（实际需替换为真实逻辑）
	return "default_client"
}

type ConsoleMCP struct {
	con *Console
}

var CMCP *ConsoleMCP

// NewMCPServer 创建一个新的MCP服务器实例
func NewMCPServer(con *Console, cmds map[string]*cobra.Command) *MCPServer {
	s := server.NewMCPServer(
		"Malice Network C2 Client",
		"1.0.0",
	)

	prompt := mcp.NewPrompt(
		"c2_command_execution",
		mcp.WithPromptDescription("首次返回帮助文档，第二次要求参数"),
	)

	s.AddPrompt(prompt, func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		clientID := getClientID(ctx)
		mu.Lock()
		defer mu.Unlock()

		if clientStates[clientID] == 0 {
			clientStates[clientID] = 1
			return mcp.NewGetPromptResult(
					"欢迎使用 Malice Network C2 客户端！\n\n", nil),
				nil
		}

		// 后续访问要求参数
		return mcp.NewGetPromptResult(
			"请输入完整命令（JSON 格式）：{\"cmdline\": \"命令内容\"}",
			nil,
		), nil
	})

	CMCP = &ConsoleMCP{
		con: con,
	}
	// 注册所有命令
	registerCommands(s, cmds)

	return &MCPServer{
		server: s,
		srv:    &http.Server{
			//ReadTimeout:  60 * time.Second,
			//WriteTimeout: 60 * time.Second,
		},
		cmds: cmds,
	}
}

// registerCommands 注册所有命令
func registerCommands(s *server.MCPServer, cmds map[string]*cobra.Command) {
	for _, cobraCmd := range cmds {
		// 注册命令
		registerCobraCommands(s, cobraCmd, "")
	}
}

// registerCobraCommands 递归注册cobra命令
func registerCobraCommands(s *server.MCPServer, cmd *cobra.Command, parentPath string) {
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
	if cmd.Short != "" {
		if cmd.Annotations["isStatic"] != "false" {
			// 为每个命令创建工具
			toolDescription := cmd.Short
			if cmd.GroupID == consts.ImplantGroup || cmd.GroupID == consts.ExecuteGroup ||
				cmd.GroupID == consts.SysGroup || cmd.GroupID == consts.FileGroup {
				toolDescription = cmd.Short + " (Implant)"
			}
			toolDescription = toolDescription + "\nExample: \n" + cmd.Example
			tool := mcp.NewTool(toolName, mcp.WithDescription(toolDescription))
			s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				response := ""
				var err error
				done := make(chan bool)

				// 检查参数是否存在
				if request.Params.Name == "" || request.Params.Arguments == nil {
					// 如果没有参数，返回命令文档
					doc := generateCommandDoc(cmd)
					return mcp.NewToolResultText(doc), nil
				}

				// 获取命令参数
				cmdLine, ok := request.Params.Arguments["cmdline"].(string)
				if !ok {
					// 如果参数类型不正确，返回命令文档
					doc := generateCommandDoc(cmd)
					return mcp.NewToolResultText(doc), nil
				}

				go func() {
					response, err = RunCommand(CMCP.con, cmdLine)
					if err != nil {
						logs.Log.Errorf("Error executing command: %v", err)
					}
					done <- true
				}()

				// 等待命令执行完成
				<-done

				if err != nil {
					return nil, err
				}

				if response != "" {
					return mcp.NewToolResultText(response), nil
				}

				// 如果没有响应，返回命令文档
				doc := generateCommandDoc(cmd)

				return mcp.NewToolResultText(doc), nil
			})
		} else {
			if cmd.Use == consts.CommandSession {
				resource := mcp.Resource{
					URI:         fmt.Sprintf("iom://%s", cmdPath),
					Name:        cmdPath,
					Description: cmd.Short,
					MIMEType:    "application/json",
				}

				s.AddResource(resource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
					err := CMCP.con.UpdateSessions(true)
					if err != nil {
						logs.Log.Errorf("Error updating sessions: %v", err)
						return []mcp.ResourceContents{
							mcp.TextResourceContents{
								URI:      request.Params.URI,
								MIMEType: "application/json",
								Text:     "[]",
							},
						}, err
					}

					// 构建 sessions JSON 数组
					var sessionsJSON []string
					for _, session := range CMCP.con.Sessions {
						marshal, err := protojson.Marshal(session.Session)
						if err != nil {
							return nil, err
						}
						sessionsJSON = append(sessionsJSON, string(marshal))
					}

					// 将 sessions 数组转换为 JSON 字符串
					jsonResponse := "[" + strings.Join(sessionsJSON, ",") + "]"

					return []mcp.ResourceContents{
						mcp.TextResourceContents{
							URI:      request.Params.URI,
							MIMEType: "application/json",
							Text:     jsonResponse,
						},
					}, nil
				})
			} else if cmd.Use == consts.CommandExplore {
				return
			} else {
				// 为静态命令创建资源
				resource := mcp.Resource{
					URI:         fmt.Sprintf("iom://%s", cmdPath),
					Name:        cmdPath,
					Description: cmd.Short,
					MIMEType:    "text/plain",
				}

				// 注册资源处理器
				s.AddResource(resource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
					response := ""
					var err error
					done := make(chan bool)

					go func() {
						cmdLine := ""
						if cmdPath != "" {
							cmdLine = cmdPath + " " + cmd.Use + " " + "--static"
						} else {
							cmdLine = cmd.Use + " " + "--static"
						}
						response, err = RunCommand(CMCP.con, cmdLine)
						if err != nil {
							logs.Log.Errorf("Error executing command: %v", err)
						}
						done <- true
					}()

					// 等待命令执行完成
					<-done

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
		}
	}

	// 递归注册子命令
	for _, subCmd := range cmd.Commands() {
		registerCobraCommands(s, subCmd, cmdPath)
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
