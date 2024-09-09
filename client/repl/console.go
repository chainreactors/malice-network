package repl

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/client/utils"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/tui"
	"github.com/mattn/go-tty"
	"github.com/reeflective/console"
	"github.com/rsteube/carapace/pkg/x"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

var (
	ErrNotFoundTask    = errors.New("task not found")
	ErrNotFoundSession = errors.New("session not found")
	Prompt             = "IOM"
	LogLevel           = logs.Warn
	Log                = logs.NewLogger(LogLevel)
	MuteLog            = logs.NewLogger(logs.Important)
)

type TaskCallback func(resp proto.Message)

// BindCmds - Bind extra commands to the app object
type BindCmds func(console *Console) console.Commands

// Start - Console entrypoint
func Start(bindCmds ...BindCmds) error {
	//assets.Setup(false, false)
	tui.Reset()
	settings, _ := assets.LoadSettings()
	con := &Console{
		ActiveTarget: &ActiveTarget{},
		Settings:     settings,
		Observers:    map[string]*Observer{},
		Plugins:      NewPlugins(),
		Log:          Log,
	}
	if len(os.Args) > 1 {
		con.newConfigLogin(os.Args[1])
	}

	con.ActiveTarget.callback = func(sess *Session) {
		con.ActiveTarget.activeObserver = NewObserver(sess)
	}

	con.NewConsole(bindCmds...)
	err := con.App.Start()
	if err != nil {
		logs.Log.Errorf("Run loop returned error: %v", err)
	}
	return err
}

type Console struct {
	*ActiveTarget
	*ServerStatus
	*Plugins
	Log          *logs.Logger
	App          *console.Console
	Settings     *assets.Settings
	ClientConfig *mtls.ClientConfig
	Callbacks    *sync.Map
	Observers    map[string]*Observer
}

func (c *Console) NewConsole(bindCmds ...BindCmds) {
	x.ClearStorage = func() {}
	iom := console.New("IoM")
	c.App = iom

	x.ClearStorage = func() {

	}
	client := iom.NewMenu(consts.ClientMenu)
	client.Short = "client commands"
	client.Prompt().Primary = c.GetPrompt
	client.AddInterrupt(io.EOF, exitConsole)
	client.AddHistorySourceFile("history", filepath.Join(assets.GetRootAppDir(), "history"))
	//client.SetCommands(bindCmds[0](c))
	client.Command = bindCmds[0](c)()
	c.App.SwitchMenu(consts.ClientMenu)

	implant := iom.NewMenu(consts.ImplantMenu)
	implant.Short = "Implant commands"
	implant.Prompt().Primary = c.GetPrompt
	implant.AddInterrupt(io.EOF, exitImplantMenu) // Ctrl-D
	implant.AddHistorySourceFile("history", filepath.Join(assets.GetRootAppDir(), "implant_history"))
	//implant.SetCommands(bindCmds[1](c))
	implant.Command = bindCmds[1](c)()
}

func (c *Console) Context() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), consts.DefaultTimeout)

	return metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"client_id", c.ClientConfig.Operator),
	)
}

func (c *Console) Login(config *mtls.ClientConfig) error {
	conn, err := mtls.Connect(config)
	if err != nil {
		logs.Log.Errorf("Failed to connect: %v", err)
		return err
	}
	logs.Log.Importantf("Connected to server %s", config.Address())
	c.ServerStatus, err = InitServerStatus(conn)
	if err != nil {
		logs.Log.Errorf("init server failed : %v", err)
		return err
	}
	c.ClientConfig = config
	logs.Log.Importantf("%d listeners, %d clients , %d sessions", len(c.Listeners), len(c.Clients), len(c.Sessions))
	return nil
}

func (c *Console) newConfigLogin(yamlFile string) {
	config, err := mtls.ReadConfig(yamlFile)
	if err != nil {
		logs.Log.Errorf("Error reading config file: %v", err)
		return
	}
	err = c.Login(config)
	if err != nil {
		logs.Log.Errorf("Error login: %v", err)
		return
	}
	err = assets.MvConfig(yamlFile)
	if err != nil {
		return
	}
}

func (c *Console) GetPrompt() string {
	session := c.ActiveTarget.Get()
	if session != nil {
		groupName := session.GroupName
		if session.Note != "" {
			return utils.NewSessionColor(groupName, session.Note)
		} else {
			sessionID := session.SessionId
			return utils.NewSessionColor(groupName, sessionID[:8])
		}

	} else {
		return tui.AdaptTermColor("IOM")
	}
}

// AddObserver - Observers to notify when the active session changes
func (c *Console) AddObserver(session *Session) string {
	Log.Infof("Add observer to %s", session.SessionId)
	c.Observers[session.SessionId] = &Observer{session}
	return session.SessionId
}

func (c *Console) RemoveObserver(observerID string) {
	delete(c.Observers, observerID)
}

func (c *Console) RefreshActiveSession() {
	if c.ActiveTarget != nil {
		c.UpdateSession(c.ActiveTarget.session.SessionId)
	}
}

func (c *Console) ImplantMenu() *cobra.Command {
	return c.App.Menu(consts.ImplantMenu).Command
}

func (c *Console) SwitchImplant(sess *Session) {
	c.ActiveTarget.Set(sess)
	c.App.SwitchMenu(consts.ImplantMenu)

	for _, cmd := range c.ImplantMenu().Commands() {
		cmd.Hidden = false
		if o, ok := cmd.Annotations["os"]; ok && !strings.Contains(o, sess.Os.Name) {
			cmd.Hidden = true
		}
		if arch, ok := cmd.Annotations["arch"]; ok && !strings.Contains(arch, sess.Os.Arch) {
			cmd.Hidden = true
		}
		if depend, ok := cmd.Annotations["depend"]; ok {
			for _, dep := range strings.Split(depend, ",") {
				if !slices.Contains(sess.Modules, dep) {
					cmd.Hidden = true
				}
			}
		}
	}
}

func (c *Console) RegisterImplantFunc(name string, fn interface{}, callback ImplantCallback) error {
	return intermediate.RegisterInternalFunc(name, WrapImplantFunc(c, fn, callback))
}

func (c *Console) RegisterServerFunc(name string, fn interface{}) error {
	return intermediate.RegisterInternalFunc(name, WrapServerFunc(c, fn))
}

func exitConsole(c *console.Console) {
	open, err := tty.Open()
	if err != nil {
		panic(err)
	}
	defer open.Close()
	var isExit = false
	fmt.Print("Press 'Y/y'  or 'Ctrl+D' to confirm exit: ")

	for {
		readRune, err := open.ReadRune()
		if err != nil {
			panic(err)
		}
		if readRune == 0 {
			continue
		}
		switch readRune {
		case 'Y', 'y':
			os.Exit(0)
		case 4: // ASCII code for Ctrl+C
			os.Exit(0)
		default:
			isExit = true
		}
		if isExit {
			break
		}
	}
}

// exitImplantMenu uses the background command to detach from the implant menu.
func exitImplantMenu(c *console.Console) {
	root := c.Menu(consts.ImplantMenu).Command
	root.SetArgs([]string{"background"})
	root.Execute()
}
