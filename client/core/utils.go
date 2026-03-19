package core

import (
	"encoding/json"
	"fmt"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/services/localrpc"
	"github.com/chainreactors/malice-network/client/plugin"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/mals"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	"strings"
	"sync"
	"time"
)

var commandExecMu sync.Mutex

func RunCommand(con *Console, cmdline interface{}) (string, error) {
	// Console state (active session/menu/callee + stdout capture window) is shared.
	// Serialize command execution to avoid cross-request output mixing.
	commandExecMu.Lock()
	defer commandExecMu.Unlock()

	var args []string
	var err error
	switch c := cmdline.(type) {
	case string:
		args, err = shellquote.Split(c)
		if err != nil {
			return "", err
		}
	case []string:
		args = c
	}
	start := time.Now()

	err = con.App.Execute(con.Context(), con.App.ActiveMenu(), args, false)
	if err != nil {
		return "", err
	}
	return client.RemoveANSI(client.Stdout.Range(start, time.Now())), nil
}

// switchSessionWithCallee 切换session并设置callee
func switchSessionWithCallee(con *Console, sessionID, callee string) error {
	if sessionID != "" {
		session, ok := con.Sessions[sessionID]
		if !ok || session == nil {
			return fmt.Errorf("session %s not found", sessionID)
		}
		con.SwitchImplant(session, callee)
	} else if con.GetInteractive() != nil {
		con.GetInteractive().Callee = callee
	}
	return nil
}

// executeCommand executes a command with automatic task waiting for implant commands.
// It reuses the same task-wait logic as executeRPCCommand to properly capture async output.
func executeCommand(con *Console, command, sessionID, callee string) (string, error) {
	return executeCommandWithTaskWait(con, command, sessionID, callee)
}

func stripWaitFlag(args []string) []string {
	if len(args) == 0 {
		return args
	}

	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--wait" || strings.HasPrefix(arg, "--wait=") {
			continue
		}
		filtered = append(filtered, arg)
	}

	return filtered
}

func resolveSessionID(con *Console, sessionID string) (string, error) {
	if sessionID != "" {
		return sessionID, nil
	}

	if sess := con.GetInteractive(); sess != nil {
		return sess.SessionId, nil
	}

	return "", fmt.Errorf("session id is required")
}

func currentSessionID(con *Console, sessionID string) (string, bool) {
	if sessionID != "" {
		return sessionID, true
	}

	if sess := con.GetInteractive(); sess != nil {
		return sess.SessionId, true
	}

	return "", false
}

func getLatestTaskID(con *Console, sessionID string) (uint32, bool, error) {
	tasks, err := con.Rpc.GetTasks(con.Context(), &clientpb.TaskRequest{
		SessionId: sessionID,
		All:       true,
	})
	if err != nil {
		return 0, false, err
	}

	if tasks == nil || len(tasks.GetTasks()) == 0 {
		return 0, false, nil
	}

	var latest uint32
	for _, task := range tasks.GetTasks() {
		if task != nil && task.GetTaskId() > latest {
			latest = task.GetTaskId()
		}
	}

	if latest == 0 {
		return 0, false, nil
	}

	return latest, true, nil
}

func renderTaskOutput(taskCtx *clientpb.TaskContext) (string, error) {
	if taskCtx == nil || taskCtx.Task == nil {
		return "", fmt.Errorf("task context is nil")
	}

	if fn, ok := intermediate.InternalFunctions[taskCtx.Task.Type]; ok && fn.FinishCallback != nil {
		result, err := fn.FinishCallback(taskCtx)
		if err != nil {
			return "", err
		}
		return result, nil
	}

	status, err := output.ParseStatus(taskCtx)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", status), nil
}

// executeRPCCommand executes a command for LocalRPC without relying on global stdout range capture.
// It waits the task by task_id and renders output through task callbacks.
func executeRPCCommand(con *Console, command, sessionID string) (string, error) {
	return executeCommandWithTaskWait(con, command, sessionID, consts.CalleeRPC)
}

// executeCommandWithTaskWait is the shared implementation for executeCommand and executeRPCCommand.
// It strips --wait, executes the command, detects new tasks, and waits for their output.
func executeCommandWithTaskWait(con *Console, command, sessionID, callee string) (string, error) {
	if command == "" {
		return "", fmt.Errorf("command is required")
	}

	commandExecMu.Lock()
	defer commandExecMu.Unlock()

	restore := con.WithNonInteractiveExecution(true)
	defer restore()

	if err := switchSessionWithCallee(con, sessionID, callee); err != nil {
		return "", err
	}

	resolvedSessionID, hasSession := currentSessionID(con, sessionID)

	var (
		beforeTaskID uint32
		beforeExists bool
		err          error
	)
	if hasSession {
		beforeTaskID, beforeExists, err = getLatestTaskID(con, resolvedSessionID)
		if err != nil {
			return "", err
		}
	}

	args, err := shellquote.Split(command)
	if err != nil {
		return "", err
	}

	args = stripWaitFlag(args)
	start := time.Now()
	if err := con.App.Execute(con.Context(), con.App.ActiveMenu(), args, false); err != nil {
		return "", err
	}
	syncOutput := strings.TrimSpace(client.RemoveANSI(client.Stdout.Range(start, time.Now())))

	if !hasSession {
		return syncOutput, nil
	}

	var targetTaskID uint32
	found := false
	deadline := time.Now().Add(3 * time.Second)
	for {
		latestTaskID, latestExists, latestErr := getLatestTaskID(con, resolvedSessionID)
		if latestErr != nil {
			return "", latestErr
		}

		if latestExists && (!beforeExists || latestTaskID > beforeTaskID) {
			targetTaskID = latestTaskID
			found = true
			break
		}

		if time.Now().After(deadline) {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	if !found {
		client.Log.Debugf("LocalRPC: no new task detected (session=%s, command=%q)\n", resolvedSessionID, command)
		return syncOutput, nil
	}

	taskCtx, err := con.Rpc.WaitTaskFinish(con.Context(), &clientpb.Task{
		SessionId: resolvedSessionID,
		TaskId:    targetTaskID,
	})
	if err != nil {
		return "", err
	}

	rendered, err := renderTaskOutput(taskCtx)
	if err != nil {
		return "", err
	}
	rendered = strings.TrimSpace(rendered)

	eventMessage := strings.TrimSpace(con.popTaskMessage(resolvedSessionID, targetTaskID))
	if rendered == "" && eventMessage != "" {
		return client.RemoveANSI(eventMessage), nil
	}
	if rendered == "" {
		return syncOutput, nil
	}

	return client.RemoveANSI(rendered), nil
}

// ExecuteLuaScript 执行Lua脚本并返回结果
func ExecuteLuaScript(con *Console, script string) (string, error) {
	vmPool := con.MalManager.GetLuaVMPool()
	if vmPool == nil {
		return "", fmt.Errorf("Lua VM Pool not initialized")
	}

	wrapper, err := vmPool.AcquireVM()
	if err != nil {
		return "", fmt.Errorf("failed to acquire VM: %w", err)
	}
	defer vmPool.ReleaseVM(wrapper)

	if err := wrapper.DoString(script); err != nil {
		return "", fmt.Errorf("failed to execute Lua script: %w", err)
	}

	var results []string
	top := wrapper.GetTop()
	for i := 1; i <= top; i++ {
		val := wrapper.Get(i)
		goVal := mals.ConvertLuaValueToGo(val)
		results = append(results, fmt.Sprintf("%v", goVal))
	}
	wrapper.Pop(top)

	if len(results) == 0 {
		return "Script executed successfully (no return value)", nil
	}

	return strings.Join(results, "\n"), nil
}

// executeLua 执行Lua脚本的通用逻辑
func executeLua(con *Console, script, sessionID, callee string) (string, error) {
	// Keep LocalRPC/Lua execution serialized with command execution.
	commandExecMu.Lock()
	defer commandExecMu.Unlock()

	if script == "" {
		return "", fmt.Errorf("script is required")
	}

	if err := switchSessionWithCallee(con, sessionID, callee); err != nil {
		return "", err
	}

	return ExecuteLuaScript(con, script)
}

// getHistory 获取历史记录的通用逻辑
func getHistory(con *Console, taskID uint32, sessionID string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("session_id is required")
	}

	session, ok := con.Sessions[sessionID]
	if !ok || session == nil {
		return "", fmt.Errorf("session %s not found", sessionID)
	}

	taskCtx, err := con.Rpc.GetTaskContent(session.Context(), &clientpb.Task{
		SessionId: sessionID,
		TaskId:    taskID,
	})
	if err != nil {
		return "", err
	}

	fn, ok := intermediate.InternalFunctions[taskCtx.Task.Type]
	if !ok || fn.FinishCallback == nil {
		return "", fmt.Errorf("task type %s not found or no callback", taskCtx.Task.Type)
	}

	return fn.FinishCallback(taskCtx)
}

// getSchemas 从指定的 cobra group 中获取 schemas 并返回 JSON 字符串
func getSchemas(con *Console, group string) (string, error) {
	if con == nil {
		return "", fmt.Errorf("console not initialized")
	}

	if group == "" {
		return "", fmt.Errorf("group is required")
	}

	// 获取 implant menu 的根命令
	rootCmd := con.App.Menu(consts.ImplantMenu)
	if rootCmd == nil {
		return "", fmt.Errorf("implant menu not found")
	}

	// 收集指定 group 的所有命令
	var commands []*cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.GroupID == group {
			commands = append(commands, cmd)
		}
	}

	if len(commands) == 0 {
		return "", fmt.Errorf("no commands found for group: %s", group)
	}

	// 使用统一 API 生成 schemas
	schemas, err := plugin.GenerateSchemasFromCommands(commands)
	if err != nil {
		return "", fmt.Errorf("failed to generate schemas: %w", err)
	}

	// 返回格式: map[groupName]map[commandName]*CommandSchema
	result := make(map[string]map[string]*plugin.CommandSchema)
	result[group] = schemas

	// 转换为 JSON 字符串
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal schemas to JSON: %w", err)
	}

	return string(jsonData), nil
}

// searchCommands searches for commands by name and description with case-insensitive substring matching.
// If sessionID is provided, switches to that session first so that Hidden flags reflect actual module availability.
func searchCommands(con *Console, query, group, sessionID string) ([]*localrpc.CommandInfo, error) {
	if con == nil {
		return nil, fmt.Errorf("console not initialized")
	}

	// Switch session context so implant commands reflect actual module availability
	if sessionID != "" {
		if err := switchSessionWithCallee(con, sessionID, consts.CalleeMCP); err != nil {
			return nil, fmt.Errorf("failed to switch session: %w", err)
		}
	}

	queryLower := strings.ToLower(query)

	var results []*localrpc.CommandInfo

	// search through both menus
	menuNames := []string{consts.ImplantMenu, consts.ClientMenu}

	for _, menuName := range menuNames {
		menu := con.App.Menu(menuName)
		if menu == nil {
			continue
		}
		for _, cmd := range menu.Commands() {
			if cmd.Hidden {
				continue
			}
			if group != "" && cmd.GroupID != group {
				continue
			}
			if matchCommand(cmd, queryLower) {
				results = append(results, commandToInfo(cmd))
			}
		}
	}

	return results, nil
}

// matchCommand checks if a command matches the query by name, description, or subcommand names
func matchCommand(cmd *cobra.Command, queryLower string) bool {
	if strings.Contains(strings.ToLower(cmd.Name()), queryLower) {
		return true
	}
	if strings.Contains(strings.ToLower(cmd.Short), queryLower) {
		return true
	}
	if strings.Contains(strings.ToLower(cmd.Long), queryLower) {
		return true
	}
	for _, alias := range cmd.Aliases {
		if strings.Contains(strings.ToLower(alias), queryLower) {
			return true
		}
	}
	for _, sub := range cmd.Commands() {
		if strings.Contains(strings.ToLower(sub.Name()), queryLower) {
			return true
		}
	}
	return false
}

// commandToInfo converts a cobra.Command to a lightweight CommandInfo proto
func commandToInfo(cmd *cobra.Command) *localrpc.CommandInfo {
	info := &localrpc.CommandInfo{
		Name:        cmd.Name(),
		Group:       cmd.GroupID,
		Description: cmd.Short,
		Usage:       cmd.UseLine(),
	}

	if ttp, ok := cmd.Annotations["ttp"]; ok {
		info.Ttp = ttp
	}
	if opsecStr, ok := cmd.Annotations["opsec"]; ok {
		var opsec float64
		if _, err := fmt.Sscanf(opsecStr, "%f", &opsec); err == nil {
			info.Opsec = int32(opsec)
		}
	}

	for _, sub := range cmd.Commands() {
		if !sub.Hidden {
			info.Subcommands = append(info.Subcommands, sub.Name())
		}
	}

	return info
}

// getGroups 获取所有 group 的基本信息（group_id -> group_title）
func getGroups(con *Console) (map[string]string, error) {
	if con == nil {
		return nil, fmt.Errorf("console not initialized")
	}

	// 获取 implant menu 的根命令
	rootCmd := con.App.Menu(consts.ImplantMenu)
	if rootCmd == nil {
		return nil, fmt.Errorf("implant menu not found")
	}

	// 收集所有 group 的 ID 和 Title
	groupMap := make(map[string]string)

	for _, grp := range rootCmd.Groups() {
		groupMap[grp.ID] = grp.Title
	}

	return groupMap, nil
}
