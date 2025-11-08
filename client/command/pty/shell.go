package pty

import (
	"context"
	"fmt"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"time"

	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

// PTYClient 用于与服务器通信
type PTYClient struct {
	rpc  clientrpc.MaliceRPCClient
	sess *client.Session

	// 回调函数
	outputCallback     func(string, string)
	errorCallback      func(string, string)
	disconnectCallback func(string)
	promptCallback     func(string, string) // 新增prompt回调
}

// NewPTYClient 创建一个新的 PTY 客户端
func NewPTYClient(rpc clientrpc.MaliceRPCClient, sess *client.Session) *PTYClient {
	return &PTYClient{
		rpc:  rpc,
		sess: sess,
	}
}

// sendPtyRequest 统一的pty请求发送方法
func (c *PTYClient) sendPtyRequest(ctx context.Context, req *implantpb.PtyRequest) error {
	_, err := c.rpc.PtyRequest(ctx, req)
	return err
}

// getNewline 根据操作系统获取换行符
func (c *PTYClient) getNewline() string {
	if c.sess.Os.Name == "windows" {
		return "\r\n"
	}
	return "\n"
}

// StartShell 启动shell会话
func (c *PTYClient) StartShell(ctx context.Context, sessionID, shellType string, cols, rows int) error {
	req := &implantpb.PtyRequest{
		Type:      consts.ModulePtyStart,
		SessionId: sessionID,
		Shell:     shellType,
		Cols:      uint32(cols),
		Rows:      uint32(rows),
	}

	if err := c.sendPtyRequest(ctx, req); err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}
	return nil
}

// SendInput 发送输入到shell
func (c *PTYClient) SendInput(ctx context.Context, sessionID, input string) error {
	req := &implantpb.PtyRequest{
		Type:      consts.ModulePtyInput,
		SessionId: sessionID,
		InputData: []byte(input),
	}

	if err := c.sendPtyRequest(ctx, req); err != nil {
		return fmt.Errorf("failed to send input: %w", err)
	}
	return nil
}

// ResizeShell 调整shell大小
func (c *PTYClient) ResizeShell(ctx context.Context, sessionID string, cols, rows int) error {
	req := &implantpb.PtyRequest{
		Type:      "resize",
		SessionId: sessionID,
		Cols:      uint32(cols),
		Rows:      uint32(rows),
	}

	if err := c.sendPtyRequest(ctx, req); err != nil {
		return fmt.Errorf("failed to resize shell: %w", err)
	}
	return nil
}

// StopShell 停止shell会话
func (c *PTYClient) StopShell(ctx context.Context, sessionID string) error {
	req := &implantpb.PtyRequest{
		Type:      consts.ModulePtyStop,
		SessionId: sessionID,
	}

	if err := c.sendPtyRequest(ctx, req); err != nil {
		return fmt.Errorf("failed to stop shell: %w", err)
	}
	return nil
}

// SetOutputCallback 设置输出回调
func (c *PTYClient) SetOutputCallback(callback func(string, string)) {
	c.outputCallback = callback
}

// SetErrorCallback 设置错误回调
func (c *PTYClient) SetErrorCallback(callback func(string, string)) {
	c.errorCallback = callback
}

// SetDisconnectCallback 设置断开连接回调
func (c *PTYClient) SetDisconnectCallback(callback func(string)) {
	c.disconnectCallback = callback
}

// SetPromptCallback 设置prompt回调
func (c *PTYClient) SetPromptCallback(callback func(string, string)) {
	c.promptCallback = callback
}

// ShellCmd 处理交互式 shell 命令
func ShellCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive().Clone(consts.CalleePty)

	// 获取命令参数
	shellTypeFlag, _ := cmd.Flags().GetString("shell")
	sessionIDFlag, _ := cmd.Flags().GetString("session-id")
	cols, _ := cmd.Flags().GetInt("cols")
	rows, _ := cmd.Flags().GetInt("rows")

	// 使用辅助函数处理默认值
	shellType := getDefaultShellType(session, shellTypeFlag)
	sessionID := generateSessionID(sessionIDFlag)

	// 创建 PTY 客户端
	ptyClient := NewPTYClient(con.Rpc, session)

	// 创建 Shell 模型与处理器（处理器里需要引用模型）
	shellModel := tui.NewShell(sessionID, nil)
	handlers := &tui.ShellHandlers{
		OnCommand: func(command string) error {
			// 根据操作系统确定换行符
			newline := ptyClient.getNewline()
			// 若之前在 Tab 时已把当前缓冲注入远端，则此处仅发送换行即可
			if shellModel.RemoteBufferMatches(command) {
				shellModel.ClearInjectedBuffer() // 清除注入标记
				return ptyClient.SendInput(session.Context(), sessionID, newline)
			}
			// 发送完整命令到远端，并添加换行符
			return ptyClient.SendInput(session.Context(), sessionID, command+newline)
		},
		OnConnect: func() error {
			return ptyClient.StartShell(session.Context(), sessionID, shellType, cols, rows)
		},
		OnDisconnect: func() error {
			return ptyClient.StopShell(session.Context(), sessionID)
		},
		OnResize: func(newCols, newRows int) error {
			return ptyClient.ResizeShell(session.Context(), sessionID, newCols, newRows)
		},
		OnCtrlC: func() error {
			// 发送 Ctrl+C 信号到远程shell
			return ptyClient.SendInput(session.Context(), sessionID, "\x03")
		},
		OnCtrlL: func() error {
			// 发送 Ctrl+L 信号到远程shell (清屏)
			return ptyClient.SendInput(session.Context(), sessionID, "\x0c")
		},
		OnTabSend: func(current string) error {
			// 发送当前输入内容 + Tab 键，让远端有完整上下文进行补全
			if current != "" {
				// 标记已注入当前缓冲，供后续 Enter 使用
				shellModel.MarkInjectedBuffer(current)
				return ptyClient.SendInput(session.Context(), sessionID, current+"\t")
			}
			// 如果当前输入为空，仅发送 Tab
			return ptyClient.SendInput(session.Context(), sessionID, "\t")
		},
		OnArrowUpSend: func(current string) error {
			// 发送上箭头键到远端处理历史命令
			return ptyClient.SendInput(session.Context(), sessionID, "\x1b[A")
		},
		OnArrowDownSend: func(current string) error {
			// 发送下箭头键到远端处理历史命令
			return ptyClient.SendInput(session.Context(), sessionID, "\x1b[B")
		},
	}
	shellModel.SetHandlers(handlers)
	delete(intermediate.InternalFunctions, "pty")
	con.RegisterImplantFunc(
		"pty",
		setupOutputHandler(shellModel, sessionID),
		"",
		nil,
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			resp := ctx.Spite.GetPtyResponse()
			if resp != nil {
				// 如果在等待 Tab 补全结果，则用返回文本更新输入行，完全不进入输出区
				if shellModel.CompletionPending() {
					if resp.OutputText != "" {
						shellModel.ApplyCompletionText(resp.OutputText)
					} else {
						// 空返回也结束本次补全等待
						shellModel.ClearCompletionPending()
					}
					// Tab 补全响应不进入输出区，直接返回
					return "", nil
				}

				// 如果在等待历史命令结果，则用返回文本更新输入行，完全不进入输出区
				if shellModel.HistoryPending() {
					if resp.OutputText != "" {
						shellModel.ApplyHistoryText(resp.OutputText)
					} else {
						// 空返回也结束本次历史命令等待
						shellModel.ClearHistoryPending()
					}
					// 历史命令响应不进入输出区，直接返回
					return "", nil
				}

				// 非补全/历史状态：正常处理输出
				if resp.OutputText != "" {
					shellModel.AddOutput(resp.OutputText)
				}

				return resp.OutputText, nil
			}
			return "", nil
		},
		nil,
	)

	// 设置客户端回调
	ptyClient.SetOutputCallback(setupOutputHandler(shellModel, sessionID))
	ptyClient.SetErrorCallback(setupErrorHandler(shellModel, sessionID))
	ptyClient.SetDisconnectCallback(setupDisconnectHandler(shellModel, sessionID))
	ptyClient.SetPromptCallback(setupPromptHandler(shellModel, sessionID))

	err := shellModel.Run()
	if err != nil {
		return fmt.Errorf("shell session failed: %w", err)
	}

	return nil
}

// 辅助函数
// getDefaultShellType 根据操作系统获取默认shell类型
func getDefaultShellType(session *client.Session, shellType string) string {
	if shellType == "" {
		if session.Os.Name == "windows" {
			return "cmd"
		}
		return "/bin/bash"
	}
	return shellType
}

// generateSessionID 生成会话ID
func generateSessionID(sessionID string) string {
	if sessionID == "" {
		return fmt.Sprintf("shell_%d", time.Now().Unix())
	}
	return sessionID
}

// setupOutputHandler 设置统一的输出处理逻辑
func setupOutputHandler(shellModel *tui.ShellModel, sessionID string) func(string, string) {
	return func(sessID, output string) {
		if sessID == sessionID {
			// 如果正在等待补全或历史命令响应，不要将输出添加到输出区域
			if !shellModel.CompletionPending() && !shellModel.HistoryPending() {
				shellModel.AddOutput(output)
			}
		}
	}
}

// setupErrorHandler 设置统一的错误处理逻辑
func setupErrorHandler(shellModel *tui.ShellModel, sessionID string) func(string, string) {
	return func(sessID, errorMsg string) {
		if sessID == sessionID {
			shellModel.AddError(errorMsg)
		}
	}
}

// setupDisconnectHandler 设置统一的断开连接处理逻辑
func setupDisconnectHandler(shellModel *tui.ShellModel, sessionID string) func(string) {
	return func(sessID string) {
		if sessID == sessionID {
			shellModel.SetConnected(false)
		}
	}
}

// setupPromptHandler 设置统一的prompt处理逻辑
func setupPromptHandler(shellModel *tui.ShellModel, sessionID string) func(string, string) {
	return func(sessID, prompt string) {
		if sessID == sessionID && prompt != "" {
			shellModel.SetPrompt(prompt)
		}
	}
}
