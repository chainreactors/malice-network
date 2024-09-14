package repl

import (
	"context"
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/client/utils"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/tui"
	"github.com/reeflective/console"
	"github.com/rsteube/carapace/pkg/x"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"io"
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
func NewConsole() (*Console, error) {
	//assets.Setup(false, false)
	tui.Reset()
	//settings, _ := assets.LoadSettings()
	//assets.SetInputrc()
	con := &Console{
		ActiveTarget: &ActiveTarget{},
		//Settings:     settings,
		Observers: map[string]*Observer{},
		Log:       Log,
		Plugins:   NewPlugins(),
	}

	con.ActiveTarget.callback = func(sess *Session) {
		con.ActiveTarget.activeObserver = NewObserver(sess)
	}

	con.NewConsole()
	return con, nil
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
	c.App.Menu(consts.ClientMenu).Command = bindCmds[0](c)()
	c.App.SwitchMenu(consts.ClientMenu)
	c.App.Menu(consts.ImplantMenu).Command = bindCmds[1](c)()

	err := c.App.Start()
	if err != nil {
		return err
	}
	return nil
}

func (c *Console) Context() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), consts.DefaultTimeout)

	return metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"client_id", c.ClientConfig.Operator),
	)
}

func (c *Console) GetSession(sessionID string) *Session {
	if sess, ok := c.Sessions[sessionID]; ok {
		return sess
	}
	return nil
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

func (c *Console) RegisterImplantFunc(name string, fn interface{}, bname string, bfn interface{}, callback ImplantCallback) {
	if fn != nil {
		intermediate.RegisterInternalFunc(name, WrapImplantFunc(c, fn, callback))
	}
	if bfn != nil {
		intermediate.RegisterInternalFunc(bname, WrapImplantFunc(c, fn, callback))
	}
}

func (c *Console) RegisterServerFunc(name string, fn interface{}) error {
	return intermediate.RegisterInternalFunc(name, WrapServerFunc(c, fn))
}
