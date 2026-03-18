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
	"sync/atomic"
	"syscall"
	"time"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
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

	// asyncPrint writes output above the current prompt when readline is idle,
	// or directly to stdout when a command is executing.
	// Initialized with tui.Down fallback; replaced by Console.Start with TransientPrintf.
	asyncPrint = func(format string, args ...any) {
		tui.Down(1)
		fmt.Printf(format, args...)
	}
)

// promptSafeWriter routes logger output through Console.TransientPrintf
// so that async log messages don't corrupt the readline prompt.
// It strips the \x1b[1E cursor-next-line escape that log format strings
// prepend, since TransientPrintf handles cursor positioning itself.
type promptSafeWriter struct {
	con *Console
}

func (w *promptSafeWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	// Strip cursor-next-line escape; TransientPrintf handles positioning.
	msg = strings.ReplaceAll(msg, "\x1b[1E", "")
	if msg == "" {
		return len(p), nil
	}
	_, err = w.con.App.TransientPrintf("%s", msg)
	return len(p), err
}

// BindCmds - Bind extra commands to the app object
type BindCmds func(console *Console) console.Commands

// Start - Console entrypoint
func NewConsole() (*Console, error) {
	//assets.Setup(false, false)
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

	// ConfigPath is the auth config file path used for the current login.
	// Populated by LoginCmd so the multiplexer can forward it to child processes.
	ConfigPath string

	// MuxChild indicates this process was spawned by the terminal multiplexer.
	MuxChild bool

	// Quiet suppresses notification event output, startup banners, and
	// MCP/LocalRPC initialization. Used by non-index mux child panes.
	// Task events (user command results) still display.
	Quiet bool

	forceNonInteractive atomic.Int32
	daemonMode          atomic.Int32
	replActive          atomic.Bool
}

// IsMuxIndex returns true when this process is the index (first) pane of the
// terminal multiplexer. The index pane acts as the notification bus — it
// receives all global events and intercepts `use` to open new panes via OSC.
func (c *Console) IsMuxIndex() bool {
	return c.MuxChild && !c.Quiet
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
	tui.Reset()

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

	// After all commands are registered, safely start MCP server and Local RPC server.
	// In quiet mode (non-index mux pane), skip these to avoid resource waste.
	if c.Server != nil && !c.Quiet {
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
	// Daemon mode follows the same runtime path even when a terminal is available.
	// Skip readline loop to avoid busy-spin on MakeRaw(ENOTTY), block on signal instead.
	if c.IsDaemonExecution() || !term.IsTerminal(int(os.Stdin.Fd())) {
		if c.IsDaemonExecution() {
			logs.Log.Importantf("running in daemon mode, waiting for signal...")
		} else {
			logs.Log.Importantf("running in headless mode (no terminal detected), waiting for signal...")
		}
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		logs.Log.Importantf("received exit signal, shutting down")
		return nil
	}

	// Wire asyncPrint so HandlerTask uses TransientPrintf for async output.
	asyncPrint = func(format string, args ...any) {
		c.App.TransientPrintf(format, args...)
	}

	// Route all logger output through TransientPrintf for prompt-safe async display.
	// This ensures background events (session register, task callbacks, etc.)
	// don't corrupt the readline prompt.
	safeWriter := &promptSafeWriter{con: c}
	client.Stdout.SetWriter(safeWriter)
	logs.Log.SetOutput(client.Stdout)

	restoreREPL := c.WithREPLExecution(true)
	defer restoreREPL()

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

func (c *Console) WithNonInteractiveExecution(enabled bool) func() {
	if c == nil {
		return func() {}
	}

	if enabled {
		c.forceNonInteractive.Add(1)
	}
	return func() {
		if enabled {
			c.forceNonInteractive.Add(-1)
		}
	}
}

func (c *Console) WithDaemonExecution(enabled bool) func() {
	if c == nil {
		return func() {}
	}

	prev := c.daemonMode.Load()
	if enabled {
		c.daemonMode.Store(1)
	} else {
		c.daemonMode.Store(0)
	}

	return func() {
		c.daemonMode.Store(prev)
	}
}

func (c *Console) IsDaemonExecution() bool {
	if c == nil {
		return false
	}

	return c.daemonMode.Load() > 0
}

func (c *Console) WithREPLExecution(enabled bool) func() {
	if c == nil {
		return func() {}
	}

	prev := c.replActive.Load()
	c.replActive.Store(enabled)

	return func() {
		c.replActive.Store(prev)
	}
}

func (c *Console) IsNonInteractiveExecution() bool {
	if c == nil {
		return !term.IsTerminal(int(os.Stdin.Fd()))
	}

	if c.forceNonInteractive.Load() > 0 {
		return true
	}

	return !c.replActive.Load()
}

func (c *Console) GetPrompt() string {
	statusLine := c.getStatusLine()
	promptLine := c.getPromptChar()

	session := c.interactiveSession()
	if session != nil {
		promptLine = tui.DefaultNameStyle.Render(shortPromptSessionID(session.SessionId)) + " " + promptLine
	}

	if statusLine == "" {
		return promptLine
	}
	return statusLine + "\n" + promptLine
}

// getPromptChar returns the ❯ prompt character in green.
func (c *Console) getPromptChar() string {
	return tui.GreenFg.Render("❯") + " "
}

func shortPromptSessionID(sessionID string) string {
	if len(sessionID) <= 8 {
		return sessionID
	}
	return sessionID[:8]
}

func (c *Console) interactiveSession() *client.Session {
	if c == nil || c.Server == nil || c.Server.ServerState == nil || c.Server.ActiveTarget == nil {
		return nil
	}
	return c.Server.ActiveTarget.Get()
}

// formatCheckinAge formats a Unix timestamp into a compact relative time string.
func formatCheckinAge(timestamp int64) string {
	if timestamp <= 0 {
		return "never"
	}
	diff := time.Now().Unix() - timestamp
	if diff <= 0 {
		return "now"
	}
	switch {
	case diff < 60:
		return fmt.Sprintf("%ds", diff)
	case diff < 3600:
		return fmt.Sprintf("%dm%ds", diff/60, diff%60)
	case diff < 86400:
		return fmt.Sprintf("%dh%dm", diff/3600, (diff/60)%60)
	default:
		return fmt.Sprintf("%dd%dh", diff/86400, (diff/3600)%24)
	}
}

// getStatusLine returns the Starship-style status line above the prompt.
func (c *Console) getStatusLine() string {
	if c == nil || c.Server == nil {
		return ""
	}

	session := c.interactiveSession()
	if session == nil {
		// Client menu: user on v0.5.0 sessions alive/total
		version := ""
		if c.Info != nil {
			version = c.Info.Version
		}
		name := ""
		if c.Client != nil {
			name = c.Client.Name
		}
		sessionInfo := fmt.Sprintf("%d", len(c.Sessions))
		if c.Rpc != nil {
			count, err := c.Rpc.GetSessionCount(context.Background(), &clientpb.Empty{})
			if err == nil && count != nil {
				sessionInfo = fmt.Sprintf("%d/%d", count.Alive, count.Total)
			}
		}
		return fmt.Sprintf("%s %s %s %s %s",
			tui.CyanFg.Render(name),
			tui.DarkGrayFg.Render("on"),
			tui.GreenFg.Render(version),
			tui.DarkGrayFg.Render("sessions"),
			tui.YellowFg.Render(sessionInfo),
		)
	}

	// Implant menu: name on hostname os/arch via pipeline age (group)
	parts := make([]string, 0, 8)
	hostname := ""
	osInfo := ""
	if session.Os != nil {
		hostname = session.Os.Hostname
		osInfo = session.Os.Name + "/" + session.Os.Arch
	}
	parts = append(parts,
		tui.WhiteFg.Bold(true).Render(session.Name),
		tui.DarkGrayFg.Render("on"),
		tui.CyanFg.Render(hostname),
		tui.GreenFg.Render(osInfo),
		tui.DarkGrayFg.Render("via"),
		tui.PurpleFg.Render(session.PipelineId),
		tui.YellowFg.Render(formatCheckinAge(session.LastCheckin)),
		tui.DarkGrayFg.Render("("+session.GroupName+")"),
	)
	return strings.Join(parts, " ")
}

func (c *Console) RefreshActiveSession() {
	if c == nil || c.Server == nil || c.Server.ServerState == nil {
		return
	}
	if session := c.interactiveSession(); session != nil {
		c.UpdateSession(session.SessionId)
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

	// Tell the mux to rename this pane to the session identity.
	if c.MuxChild {
		fmt.Printf("\x1b]0;MuxRename=%s@%s\x07", sess.Note, sess.Target)
	}
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
