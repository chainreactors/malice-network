package core

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
	"github.com/chainreactors/malice-network/client/repl"
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

	// AI Completion Engine
	aiCompletionEngine *AICompletionEngine
	aiCache            *AICompletionCache
	commandValidator   *CommandValidator
}

func (c *Console) NewConsole() {
	x.ClearStorage = func() {}
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

	// Register AI command generator for Alt+A.
	iom.SetAICommandGenerator(c.handleAIGenerateCommand)

	// Register AI smart completion for Tab (delayed trigger)
	iom.Shell().AISmartComplete = c.handleAISmartComplete

	// Register line hook to handle '?' prefix without space (e.g., '?你好' -> '?' '你好')
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

	// Initialize AI completion components after commands are registered
	c.initAICompletion()

	// 所有命令注册完成后，安全地启动MCP服务器和Local RPC服务器
	if c.Server != nil {
		c.InitMCPServer()
		c.InitLocalRPCServer()
	}

	if c.Session == nil {
		c.App.SwitchMenu(consts.ClientMenu)
	} else {
		c.SwitchImplant(c.GetInteractive(), consts.CalleeCMD)
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

func getValidAISettings() (*assets.AISettings, error) {
	settings, err := assets.GetSetting()
	if err != nil {
		return nil, fmt.Errorf("failed to load settings: %w", err)
	}
	if settings == nil || settings.AI == nil || !settings.AI.Enable {
		return nil, fmt.Errorf("AI not enabled. Use 'ai-config --enable --api-key <key>' to enable it")
	}
	if settings.AI.APIKey == "" {
		return nil, fmt.Errorf("AI API key not configured. Use 'ai-config --api-key <key>' to set it")
	}

	return settings.AI, nil
}

func (c *Console) handleAIGenerateCommand(line string, history []string) (string, error) {
	ai, err := getValidAISettings()
	if err != nil {
		return "", err
	}

	description := strings.TrimSpace(line)
	if description == "" {
		return "", fmt.Errorf("empty description")
	}

	if ai.HistorySize <= 0 {
		history = nil
	} else if len(history) > ai.HistorySize {
		history = history[len(history)-ai.HistorySize:]
	}

	aiClient := NewAIClient(ai)
	timeout := ai.Timeout
	if timeout <= 0 {
		timeout = 15
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	question := fmt.Sprintf(
		"Convert the following natural language description to a command.\n\n"+
			"Rules:\n"+
			"1) Return ONLY the command, no explanation.\n"+
			"2) Use the exact command syntax from the available commands.\n"+
			"3) Wrap the command in single backticks.\n"+
			"4) If you're unsure, return the most likely command.\n\n"+
			"Description: %s",
		description,
	)

	response, err := aiClient.Ask(ctx, question, history)
	if err != nil {
		return "", err
	}

	commands := ParseCommandSuggestions(response)
	if len(commands) == 0 {
		return "", fmt.Errorf("AI did not generate a valid command")
	}

	return commands[0].Command, nil
}

// initAICompletion initializes the AI completion engine
func (c *Console) initAICompletion() {
	settings, err := assets.GetSetting()
	if err != nil || settings == nil || settings.AI == nil || !settings.AI.Enable {
		return
	}

	// Initialize cache (500 entries, 30 minute TTL)
	c.aiCache = NewAICompletionCache(500, 30*time.Minute)

	// Initialize command validator from client menu
	clientMenu := c.App.Menu(consts.ClientMenu).Command
	if clientMenu != nil {
		c.commandValidator = NewCommandValidatorWithMenu(clientMenu, consts.ClientMenu)
	}

	// Also add implant menu commands to validator
	implantMenu := c.App.Menu(consts.ImplantMenu).Command
	if implantMenu != nil && c.commandValidator != nil {
		c.commandValidator.AddCommandsFromCobra(implantMenu, consts.ImplantMenu)
	}

	// Initialize AI client
	aiClient := NewAIClient(settings.AI)

	// Create completion engine
	c.aiCompletionEngine = NewAICompletionEngine(aiClient, c.aiCache, c.commandValidator)
}

// handleAISmartComplete handles AI smart completion for Tab key
func (c *Console) handleAISmartComplete(line string, history []string) ([]string, error) {
	// Lazy initialize if not done yet
	if c.aiCompletionEngine == nil {
		c.initAICompletion()
	}

	if c.aiCompletionEngine == nil {
		return nil, fmt.Errorf("AI completion not available")
	}

	// Use 3 second timeout for fast completion
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	activeMenu := ""
	if c != nil && c.App != nil {
		if m := c.App.ActiveMenu(); m != nil {
			activeMenu = m.Name()
		}
	}

	return c.aiCompletionEngine.SmartComplete(ctx, line, history, activeMenu)
}
