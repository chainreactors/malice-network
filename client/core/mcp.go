package core

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/kballard/go-shellquote"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

// MCPServer 包装了MCP服务器实例
type MCPServer struct {
	server       *server.MCPServer
	streamServer *server.StreamableHTTPServer
	sseServer    *server.SSEServer
	httpServer   *http.Server
	console      *Console
}

const mcpEndpointPath = "/mcp"

// NewMCP 创建一个新的MCP服务器实例
func NewMCP(console *Console) *MCPServer {
	s := server.NewMCPServer(
		"Malice Network C2 Client",
		"1.0.0",
	)

	mcp := &MCPServer{
		server:  s,
		console: console,
	}

	// 注册提示词和工具
	mcp.registerPrompts()
	mcp.registerCustomTools()

	return mcp
}

// registerPrompts 注册 MCP 提示词
func (m *MCPServer) registerPrompts() {
	// 问候提示词
	m.server.AddPrompt(
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
	m.server.AddPrompt(
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
	if cmd.Annotations["static"] != "true" && cmd.Annotations["resource"] != "true" {
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
		// 获取命令参数
		cmdLine, err := request.RequireString("cmdline")
		if err != nil {
			return mcp.NewToolResultText(toolDescription), nil
		}

		// 执行命令
		restore := c.WithNonInteractiveExecution(true)
		defer restore()

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
		restore := c.WithNonInteractiveExecution(true)
		defer restore()

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
		return cmdPath + " --all"
	} else if parentPath == consts.CommandArtifact {
		return cmdPath
	}
	return cmdPath
}

// generateCommandDoc 生成详细的命令文档
func generateCommandDoc(cmd *cobra.Command) string {
	var doc strings.Builder
	repl.GenMarkdownCustom(cmd, &doc, func(s string) string {
		return s
	})
	return doc.String()
}

// isHelpRequest reports whether cmdline asks for help, e.g.
// "bof --help", "elevate EfsPotato -h", "help", "help bof".
func isHelpRequest(cmdline string) bool {
	args, err := shellquote.Split(cmdline)
	if err != nil || len(args) == 0 {
		return false
	}
	if args[0] == "help" {
		return true
	}
	for _, a := range args {
		if a == "--help" || a == "-h" {
			return true
		}
	}
	return false
}

// renderCobraHelp writes a command's help into a buffer and returns the
// ANSI-stripped text. Cobra normally writes --help to the command output
// writer (os.Stdout); this captures it so the MCP caller receives the text.
func renderCobraHelp(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	var buf bytes.Buffer
	orig := cmd.OutOrStdout()
	cmd.SetOut(&buf)
	err := cmd.Help()
	cmd.SetOut(orig)
	if err != nil {
		return ""
	}
	if out := strings.TrimSpace(client.RemoveANSI(buf.String())); out != "" {
		return out
	}
	if doc := strings.TrimSpace(generateCommandDoc(cmd)); doc != "" {
		return doc
	}
	return ""
}

func menuRoot(con *Console, menu string) *cobra.Command {
	if con == nil || con.App == nil {
		return nil
	}
	m := con.App.Menu(menu)
	if m == nil {
		return nil
	}
	return m.Command
}

func activeMenuRoot(con *Console, roots ...*cobra.Command) *cobra.Command {
	if con == nil || con.App == nil {
		return nil
	}
	am := con.App.ActiveMenu()
	if am == nil {
		return nil
	}
	for _, r := range roots {
		if r != nil && r == am.Command {
			return r
		}
	}
	return nil
}

// renderCommandHelp resolves the command targeted by a help request and
// returns its rendered help text. It tries both menus so implant commands
// (e.g. "bof") resolve even from the client menu. An empty path ("--help" /
// "help") returns the active menu's full command tree.
func (m *MCPServer) renderCommandHelp(cmdline string) string {
	args, err := shellquote.Split(cmdline)
	if err != nil {
		return ""
	}
	var path []string
	for _, a := range args {
		if a == "--help" || a == "-h" || a == "help" {
			continue
		}
		// MAL/plugin commands use a ":" namespace (e.g. "uac-bypass:elevatedcom")
		// which maps to a cobra parent->child path. Split on ":" so Find resolves.
		for _, p := range strings.Split(a, ":") {
			if p != "" {
				path = append(path, p)
			}
		}
	}

	implantRoot := menuRoot(m.console, consts.ImplantMenu)
	clientRoot := menuRoot(m.console, consts.ClientMenu)

	// Serialize against command execution (shared cobra tree / output writer).
	commandExecMu.Lock()
	defer commandExecMu.Unlock()

	if len(path) == 0 {
		root := activeMenuRoot(m.console, implantRoot, clientRoot)
		if root == nil {
			root = implantRoot
		}
		return renderCobraHelp(root)
	}

	for _, root := range []*cobra.Command{implantRoot, clientRoot} {
		if root == nil {
			continue
		}
		target, _, e := root.Find(path)
		if e != nil || target == nil || target == root {
			continue
		}
		if out := renderCobraHelp(target); out != "" {
			return out
		}
	}
	return ""
}

// Start 启动 MCP HTTP 服务器
func (m *MCPServer) Start(host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)

	// Streamable HTTP is the current MCP transport. Keep the legacy SSE
	// endpoints mounted for older MCP clients that still use HTTP+SSE.
	m.streamServer = server.NewStreamableHTTPServer(
		m.server,
		server.WithEndpointPath(mcpEndpointPath),
	)
	m.sseServer = server.NewSSEServer(
		m.server,
		server.WithBaseURL(fmt.Sprintf("http://%s:%d%s", host, port, mcpEndpointPath)),
	)

	mux := http.NewServeMux()
	mux.Handle(mcpEndpointPath, m.streamServer)
	mux.Handle(m.sseServer.CompleteSsePath(), m.sseServer.SSEHandler())
	mux.Handle(m.sseServer.CompleteMessagePath(), m.sseServer.MessageHandler())

	m.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	go func() {
		if err := m.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			logs.Log.Errorf("Failed to start MCP server: %v\n", err)
		}
	}()

	return nil
}

// Stop 停止 MCP 服务器
func (m *MCPServer) Stop() error {
	var shutdownErr error
	if m.httpServer != nil {
		shutdownErr = m.httpServer.Shutdown(context.Background())
	}
	if m.streamServer != nil {
		if err := m.streamServer.Shutdown(context.Background()); shutdownErr == nil {
			shutdownErr = err
		}
	}
	if m.sseServer != nil {
		if err := m.sseServer.Shutdown(context.Background()); shutdownErr == nil {
			shutdownErr = err
		}
	}
	return shutdownErr
}

// AddTool 添加新的工具到 MCP 服务器
func (m *MCPServer) AddTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	m.server.AddTool(tool, handler)
}

// registerCustomTools 注册自定义 MCP 工具
func (m *MCPServer) registerCustomTools() {
	m.registerExecuteCommandTool()
	m.registerLuaScriptTool()
	m.registerGetHistoryTool()
	m.registerSearchTool()
}

// registerExecuteCommandTool 注册通用命令执行工具
func (m *MCPServer) registerExecuteCommandTool() {
	tool := mcp.NewTool(
		"execute_command",
		mcp.WithDescription(`Execute any client command as if you were typing in the console.

Examples:
- "session --all" - List all sessions
- "session --help" - Show multiline command help
- "search screenshot" - Search commands and plugins
- "skill list" - List available AI skills
- "ask explain this result" - Ask the configured local AI
- "analyze permission denied" - Analyze an error with the configured local AI
- "use <session_id>" - Switch to a session
- "whoami" - Execute whoami in current session (requires active session)
- "ls" - List files in current directory (requires active session)
- "download /path/to/file" - Download a file (requires active session)

The command will be executed in the current context (client or implant mode).
Commands are automatically routed to client menu or implant menu based on whether there's an active session.`),
		mcp.WithString("command", mcp.Required(), mcp.Description("The command to execute, exactly as you would type it in the console")),
		mcp.WithString("session_id", mcp.Description("Optional session ID to set as active context before execution")),
	)

	m.server.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		command, err := request.RequireString("command")
		if err != nil || command == "" {
			return mcp.NewToolResultError("command is required"), nil
		}

		sessionID, _ := request.GetArguments()["session_id"].(string)

		// Cobra writes --help to the command output writer (os.Stdout), which
		// executeCommand does not capture. Short-circuit help requests and
		// render them into a buffer so the help text reaches the caller.
		if isHelpRequest(command) {
			if help := m.renderCommandHelp(command); help != "" {
				return mcp.NewToolResultText(help), nil
			}
		}

		response, err := executeCommand(m.console, command, sessionID, consts.CalleeMCP)
		if err != nil {
			logs.Log.Errorf("Error executing command: %v", err)
			return mcp.NewToolResultError(fmt.Sprintf("Error: %v", err)), nil
		}

		if response == "" {
			response = "Command executed successfully (no output)"
		}

		return mcp.NewToolResultText(response), nil
	})
}

// registerLuaScriptTool 注册 Lua 脚本执行工具
func (m *MCPServer) registerLuaScriptTool() {
	tool := mcp.NewTool(
		"execute_lua",
		mcp.WithDescription("Execute arbitrary Lua script in the client context. This tool allows you to run Lua code with access to all internal functions and the current session context."),
		mcp.WithString("script", mcp.Required(), mcp.Description("Lua script code to execute")),
		mcp.WithString("session_id", mcp.Description("Optional session ID to set as active context before execution")),
	)

	m.server.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		script, err := request.RequireString("script")
		if err != nil || script == "" {
			return mcp.NewToolResultError("script is required"), nil
		}

		sessionID, _ := request.GetArguments()["session_id"].(string)

		result, err := executeLua(m.console, script, sessionID, consts.CalleeMCP)
		if err != nil {
			logs.Log.Errorf("Error executing Lua script: %v", err)
			return mcp.NewToolResultError(fmt.Sprintf("Error: %v", err)), nil
		}

		return mcp.NewToolResultText(result), nil
	})
}

// registerGetHistoryTool 注册获取历史记录工具
func (m *MCPServer) registerGetHistoryTool() {
	tool := mcp.NewTool(
		"get_history",
		mcp.WithDescription("Get rendered history data for a specific task ID. Returns the output of a previously executed task."),
		mcp.WithNumber("task_id", mcp.Required(), mcp.Description("Task ID to retrieve history for")),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID context")),
	)

	m.server.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		taskID, err := request.RequireFloat("task_id")
		if err != nil {
			return mcp.NewToolResultError("task_id is required"), nil
		}

		sessionID, err := request.RequireString("session_id")
		if err != nil || sessionID == "" {
			return mcp.NewToolResultError("session_id is required"), nil
		}

		output, err := getHistory(m.console, uint32(taskID), sessionID)
		if err != nil {
			logs.Log.Errorf("Error getting history: %v", err)
			return mcp.NewToolResultError(fmt.Sprintf("Error: %v", err)), nil
		}

		return mcp.NewToolResultText(output), nil
	})
}

// registerSearchTool registers the unified search tool (semantic + keyword).
func (m *MCPServer) registerSearchTool() {
	tool := mcp.NewTool(
		"search_commands",
		mcp.WithDescription(`Search for commands and plugins using semantic search and keyword matching.
Supports natural language queries in Chinese and English.
Returns ranked results with descriptions, ATT&CK TTPs, and OPSEC ratings.

Examples:
- "credential dump" → credman, hashdump, logonpasswords
- "提权" → getsystem, elevate
- "lateral movement" → move, pivot, psexec
- "persist" → persistence, scheduled task`),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query (natural language or keywords)")),
		mcp.WithString("type", mcp.Description("Filter: command, plugin")),
		mcp.WithString("group", mcp.Description("Filter by command group (e.g., 'execute', 'sys', 'file', 'pivot')")),
		mcp.WithNumber("limit", mcp.Description("Max results (default 20)")),
	)

	m.server.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := request.RequireString("query")
		if err != nil || query == "" {
			return mcp.NewToolResultError("query is required"), nil
		}

		typeFilter, _ := request.GetArguments()["type"].(string)
		group, _ := request.GetArguments()["group"].(string)
		limit := 20
		if l, ok := request.GetArguments()["limit"].(float64); ok && l > 0 {
			limit = int(l)
		}

		results, err := HybridSearch(ctx, m.console.SearchIndex, m.console.VectorIndex, query, typeFilter, group, limit)
		if err != nil {
			logs.Log.Errorf("search error: %v", err)
			return mcp.NewToolResultError(fmt.Sprintf("Error: %v", err)), nil
		}

		if len(results) == 0 {
			return mcp.NewToolResultText("No commands found matching: " + query), nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d commands matching \"%s\":\n\n", len(results), query))
		for _, r := range results {
			sb.WriteString(fmt.Sprintf("- **%s** [%s]: %s\n", r.Name, r.Category, r.Description))
			if r.TTP != "" {
				sb.WriteString(fmt.Sprintf("  ATT&CK: %s", r.TTP))
				if r.Opsec != "" && r.Opsec != "0" {
					sb.WriteString(fmt.Sprintf(" | OPSEC: %s/10", r.Opsec))
				}
				sb.WriteString("\n")
			}
			if r.Usage != "" {
				sb.WriteString(fmt.Sprintf("  Usage: %s\n", r.Usage))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("Tip: Use execute_command(\"<command> --help\") for detailed usage.")

		return mcp.NewToolResultText(sb.String()), nil
	})
}
