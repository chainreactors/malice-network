package core

import (
	"fmt"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/mals"
	"github.com/kballard/go-shellquote"
	"strings"
	"time"
)

func RunCommand(con *Console, cmdline interface{}) (string, error) {
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

// executeCommand 执行命令的通用逻辑
func executeCommand(con *Console, command, sessionID, callee string) (string, error) {
	if command == "" {
		return "", fmt.Errorf("command is required")
	}

	if err := switchSessionWithCallee(con, sessionID, callee); err != nil {
		return "", err
	}

	return RunCommand(con, command)
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
