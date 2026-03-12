package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
	"golang.org/x/term"
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
	*Server
	Log     *client.Logger
	App     *console.Console
	Profile *assets.Profile

	MCPAddr  string
	RPCAddr  string
	MCP      *MCPServer
	LocalRPC *LocalRPC

	CMDs    map[string]*cobra.Command
	Helpers map[string]*cobra.Command

	MalManager *plugin.MalManager
}

func (c *Console) NewConsole() {
	iom := console.New("IoM")
	c.App = iom

	client := iom.NewMenu(consts.ClientMenu)
	client.Short = "client commands"
	client.Prompt().Primary = c.GetPrompt
	client.AddInterrupt(io.EOF, repl.ExitConsole)
	client.AddHistorySourceFile("history", filepath.Join(assets.GetRootAppDir(), "history"))

	implant := iom.NewMenu(consts.ImplantMenu)
	implant.Short = "Implant commands"
	implant.Prompt().Primary = c.GetPrompt
	implant.AddInterrupt(io.EOF, repl.ExitImplantMenu) // Ctrl-D
	implant.AddHistorySourceFile("history", filepath.Join(assets.GetRootAppDir(), "implant_history"))

	// Register line hook to handle '?' prefix without space (e.g., '?hello' -> '?' 'hello')
	iom.PreCmdRunLineHooks = append(iom.PreCmdRunLineHooks, func(args []string) ([]string, error) {
		if len(args) > 0 && len(args[0]) > 1 && strings.HasPrefix(args[0], "?") {
			// Split '?xxx' into '?' and 'xxx'
			question := args[0][1:]
			newArgs := make([]string, 0, len(args)+1)
			newArgs = append(newArgs, "?", question)
			newArgs = append(newArgs, args[1:]...)
			return newArgs, nil
		}
		return args, nil
	})
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

	// After all commands are registered, safely start MCP server and Local RPC server
	if c.Server != nil {
		c.InitMCPServer()
		c.InitLocalRPCServer()
	}

	// Initialize active menu BEFORE headless check.
	// MCP/LocalRPC depend on ActiveMenu() returning the correct menu
	// (RunCommand calls con.App.Execute(ctx, con.App.ActiveMenu(), args, false)).
	if c.Session == nil {
		c.App.SwitchMenu(consts.ClientMenu)
	} else {
		c.SwitchImplant(c.GetInteractive(), consts.CalleeCMD)
	}

	// Headless mode: stdin is not a terminal (e.g., launched by GUI with /dev/null).
	// Skip readline loop to avoid busy-spin on MakeRaw(ENOTTY), block on signal instead.
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		logs.Log.Importantf("running in headless mode (no terminal detected), waiting for signal...")
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		logs.Log.Importantf("received exit signal, shutting down")
		return nil
	}

	return c.App.Start()
}

func (c *Console) Context() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), consts.DefaultTimeout)
	_ = cancel

	return metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"client_id", fmt.Sprintf("%s_%d", c.Client.Name, c.Client.ID)),
	)
}

func (c *Console) SyncBuildContext() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), consts.SyncBuildTimeout)
	_ = cancel

	return metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"client_id", fmt.Sprintf("%s_%d", c.Client.Name, c.Client.ID)),
	)
}

func (c *Console) GetPrompt() string {
	session := c.ActiveTarget.Get()
	if session != nil {
		groupName := session.GroupName
		sessionID := session.SessionId
		return tui.NewSessionColor(groupName, sessionID[:8])
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
		refreshCmdVisibility(cmd, sess)

		if cmd.Hidden == false {
			count++
		}
	}
	return count
}

// refreshCmdVisibility sets Hidden on a command (and its subcommands recursively)
// based on session os/arch/type/modules. For parent commands without a "depend"
// annotation, they are hidden when all their subcommands are hidden.
func refreshCmdVisibility(cmd *cobra.Command, sess *client.Session) {
	// Recursively refresh subcommands first
	for _, sub := range cmd.Commands() {
		refreshCmdVisibility(sub, sess)
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

	// For parent commands without "depend" annotation, hide them if all
	// their subcommands are hidden (e.g. "pipe" when no pipe modules exist)
	if _, hasDep := cmd.Annotations["depend"]; !hasDep && cmd.HasSubCommands() {
		allSubHidden := true
		for _, sub := range cmd.Commands() {
			if !sub.Hidden {
				allSubHidden = false
				break
			}
		}
		if allSubHidden {
			cmd.Hidden = true
		}
	}
}

func (c *Console) SwitchImplant(sess *client.Session, callee string) {
	current := c.Session
	if current != nil && current.SessionId == sess.SessionId {
		return
	}
	sess.Callee = callee
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

func (c *Console) GetRecentHistory(limit int) []string {
	if limit <= 0 || c == nil || c.App == nil {
		return nil
	}

	shell := c.App.Shell()
	if shell == nil || shell.History == nil || shell.History.Current() == nil {
		return nil
	}

	hist := shell.History.Current()
	count := hist.Len()
	start := count - limit
	if start < 0 {
		start = 0
	}

	capacity := limit
	if count-start < capacity {
		capacity = count - start
	}
	history := make([]string, 0, capacity)
	for i := start; i < count; i++ {
		if line, err := hist.GetLine(i); err == nil && line != "" {
			history = append(history, line)
		}
	}

	if len(history) > limit {
		history = history[len(history)-limit:]
	}

	return history
}
