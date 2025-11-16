package repl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/carapace-sh/carapace/pkg/x"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/metadata"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/plugin"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/mals"
	"github.com/chainreactors/tui"
)

var (
	ErrNotFoundSession = errors.New("session not found")
	Prompt             = "IoM"
)

// BindCmds - Bind extra commands to the app object
type BindCmds func(console *Console) console.Commands

// Start - Console entrypoint
func NewConsole() (*Console, error) {
	//assets.Setup(false, false)

	tui.Reset()
	//settings, _ := assets.LoadSettings()
	//assets.SetInputrc()
	con := &Console{
		//ActiveTarget: &core.ActiveTarget{},
		//Settings:     settings,
		Log:     client.Log,
		CMDs:    make(map[string]*cobra.Command),
		Helpers: make(map[string]*cobra.Command),
	}
	con.NewConsole()
	_, err := assets.LoadProfile()
	if err != nil {
		return nil, err
	}
	return con, nil
}

type Console struct {
	//*core.ActiveTarget
	*core.Server
	Log        *client.Logger
	App        *console.Console
	Profile    *assets.Profile
	CMDs       map[string]*cobra.Command
	MCP        *MCPServer
	LocalRPC   *LocalRPC
	Helpers    map[string]*cobra.Command
	MalManager *plugin.MalManager
}

func (c *Console) NewConsole() {
	x.ClearStorage = func() {}
	iom := console.New("IoM")
	c.App = iom

	client := iom.NewMenu(consts.ClientMenu)
	client.Short = "client commands"
	client.Prompt().Primary = c.GetPrompt
	client.AddInterrupt(io.EOF, exitConsole)
	client.AddHistorySourceFile("history", filepath.Join(assets.GetRootAppDir(), "history"))

	implant := iom.NewMenu(consts.ImplantMenu)
	implant.Short = "Implant commands"
	implant.Prompt().Primary = c.GetPrompt
	implant.AddInterrupt(io.EOF, exitImplantMenu) // Ctrl-D
	implant.AddHistorySourceFile("history", filepath.Join(assets.GetRootAppDir(), "implant_history"))
}

func (c *Console) Start(bindCmds ...BindCmds) error {
	go func() {
		for {
			if c.Server != nil && !c.Server.EventStatus {
				c.EventHandler()
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	intermediate.RegisterBuiltin(c.Rpc)

	c.App.Menu(consts.ClientMenu).Command = bindCmds[0](c)()
	c.App.Menu(consts.ImplantMenu).Command = bindCmds[1](c)()

	// 所有命令注册完成后，安全地启动MCP服务器和Local RPC服务器
	if c.Server != nil {
		c.InitMCPServer()
		c.InitLocalRPCServer()
	}

	if c.GetInteractive() == nil {
		c.App.SwitchMenu(consts.ClientMenu)
	} else {
		c.SwitchImplant(c.GetInteractive())
	}
	err := c.App.Start()
	if err != nil {
		return err
	}
	return nil
}

func (c *Console) Context() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), consts.DefaultTimeout)

	return metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"client_id", fmt.Sprintf("%s_%d", c.Client.Name, c.Client.ID)),
	)
}

func (c *Console) SyncBuildContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), consts.SyncBuildTimeout)

	return metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"client_id", fmt.Sprintf("%s_%d", c.Client.Name, c.Client.ID)),
	)
}

func (c *Console) GetPrompt() string {
	session := c.ActiveTarget.Get()
	if session != nil {
		groupName := session.GroupName
		sessionID := session.SessionId
		return NewSessionColor(groupName, sessionID[:8])
	} else {
		return tui.AdaptTermColor("IOM")
	}
}

func (c *Console) RefreshActiveSession() {
	if c.ActiveTarget != nil {
		c.UpdateSession(c.ActiveTarget.Session.SessionId)
	}
}

func (c *Console) ImplantMenu() *cobra.Command {
	return c.App.Menu(consts.ImplantMenu).Command
}

func (c *Console) RefreshCmd(sess *client.Session) int {
	var count int
	for _, cmd := range c.CMDs {
		if cmd.Annotations["menu"] != consts.ImplantMenu {
			continue
		}
		cmd.Hidden = false
		if o, ok := cmd.Annotations["os"]; ok && !strings.Contains(o, sess.Os.Name) {
			cmd.Hidden = true
		}
		if arch, ok := cmd.Annotations["arch"]; ok && !strings.Contains(arch, sess.Os.Arch) {
			cmd.Hidden = true
		}
		if implantType, ok := cmd.Annotations["implant"]; ok && sess.Type != implantType {
			cmd.Hidden = true
		}
		if depend, ok := cmd.Annotations["depend"]; ok {
			for _, dep := range strings.Split(depend, ",") {
				if !slices.Contains(sess.Modules, dep) {
					cmd.Hidden = true
				}
			}
		}
		if cmd.Hidden == false {
			count++
		}
	}
	return count
}

func (c *Console) SwitchImplant(sess *client.Session) {
	current := c.GetInteractive()
	if current != nil && current.SessionId == sess.SessionId {
		return
	}
	c.ActiveTarget.Set(sess)
	c.App.SwitchMenu(consts.ImplantMenu)
}

func (c *Console) RegisterImplantFunc(name string, fn interface{},
	bname string, bfn interface{}, // return to plugin
	internalCallback ImplantFuncCallback, callback intermediate.ImplantCallback) {

	if callback == nil {
		callback = WrapClientCallback(internalCallback)
	}

	if fn != nil {
		intermediate.RegisterInternalFunc(intermediate.BuiltinPackage, name, WrapImplantFunc(c, fn, internalCallback), callback)
	}

	if bfn != nil {
		intermediate.RegisterInternalFunc(intermediate.BeaconPackage, bname, WrapImplantFunc(c, bfn, internalCallback), callback)
	}
}

func (c *Console) RegisterAggressiveFunc(name string, fn interface{}, internalCallback ImplantFuncCallback, callback intermediate.ImplantCallback) {
	if callback == nil {
		callback = WrapClientCallback(internalCallback)
	}

	intermediate.RegisterInternalFunc(intermediate.BuiltinPackage, name, WrapImplantFunc(c, fn, internalCallback), callback)
}

func (c *Console) RegisterBuiltinFunc(pkg, name string, fn interface{}, callback ImplantFuncCallback) error {
	var implantCallback intermediate.ImplantCallback
	if callback == nil {
		implantCallback = WrapClientCallback(callback)
	}

	return intermediate.RegisterInternalFunc(pkg, name, WrapImplantFunc(c, fn, callback), implantCallback)
}

func (c *Console) RegisterServerFunc(name string, fn interface{}, helper *mals.Helper) error {
	err := intermediate.RegisterInternalFunc(intermediate.BuiltinPackage, name, WrapServerFunc(c, fn), nil)
	if helper != nil {
		return intermediate.AddHelper(name, helper)
	}
	return err
}

func (c *Console) AddCommandFuncHelper(cmdName string, funcName string, example string, input, output []string) error {
	cmd, ok := c.CMDs[cmdName]
	if !ok {
		cmd, ok = c.Helpers[cmdName]
	}
	if ok {
		var group string
		if cmd.GroupID == "" {
			group = cmd.Parent().GroupID
		} else {
			group = cmd.GroupID
		}
		return intermediate.AddHelper(funcName, &mals.Helper{
			CMDName: cmdName,
			Group:   group,
			Short:   cmd.Short,
			Long:    cmd.Long,
			Input:   input,
			Output:  output,
			Example: example,
		})
	} else {
		return intermediate.AddHelper(funcName, &mals.Helper{
			CMDName: cmdName,
			Input:   input,
			Output:  output,
			Example: example,
		})
	}
}

// ExecuteCommandWithSession executes a command with optional session context
// This method is used by the local gRPC server to execute commands
// When sessionId is provided, it temporarily switches to that session context
// RunCommand will automatically select the appropriate menu (ClientMenu or ImplantMenu)
func (c *Console) ExecuteCommandWithSession(command string, sessionId string) (string, error) {
	if sessionId != "" {
		session, ok := c.Sessions[sessionId]
		if !ok || session == nil {
			return "", fmt.Errorf("session %s not found", sessionId)
		}
		c.SwitchImplant(session)
	}

	return RunCommand(c, command)
}

// ExecuteLuaWithSession executes a Lua script with optional session context
// This method is used by the local gRPC server to execute Lua scripts
// When sessionId is provided, it temporarily switches to that session context
func (c *Console) ExecuteLuaWithSession(script string, sessionId string) (string, error) {
	if sessionId != "" {
		session, ok := c.Sessions[sessionId]
		if !ok || session == nil {
			return "", fmt.Errorf("session %s not found", sessionId)
		}
		c.SwitchImplant(session)
	} else if c.GetInteractive() == nil {
		return "", fmt.Errorf("no active session, please provide session_id or use 'use' command to select a session")
	}
	// Execute the Lua script with the current session context
	return c.ExecuteLuaScript(script)
}

// ExecuteLuaScript executes a Lua script and returns the result
func (c *Console) ExecuteLuaScript(script string) (string, error) {
	// Get shared Lua VM Pool from MalManager
	vmPool := c.MalManager.GetLuaVMPool()
	if vmPool == nil {
		return "", fmt.Errorf("Lua VM Pool not initialized")
	}

	// Acquire VM from pool
	wrapper, err := vmPool.AcquireVM()
	if err != nil {
		return "", fmt.Errorf("failed to acquire VM: %w", err)
	}
	defer vmPool.ReleaseVM(wrapper)

	// Execute script
	if err := wrapper.DoString(script); err != nil {
		return "", fmt.Errorf("failed to execute Lua script: %w", err)
	}

	// Collect return values
	var results []string
	top := wrapper.GetTop()
	for i := 1; i <= top; i++ {
		val := wrapper.Get(i)
		goVal := mals.ConvertLuaValueToGo(val)
		results = append(results, fmt.Sprintf("%v", goVal))
	}

	// Clean up stack
	wrapper.Pop(top)

	if len(results) == 0 {
		return "Script executed successfully (no return value)", nil
	}

	return strings.Join(results, "\n"), nil
}
