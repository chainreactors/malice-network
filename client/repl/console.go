package repl

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/tui"
	"github.com/reeflective/console"
	"github.com/rsteube/carapace/pkg/x"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/metadata"
	"io"
	"path/filepath"
	"slices"
	"strings"
)

var (
	ErrNotFoundSession = errors.New("session not found")
	Prompt             = "IOM"
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
		Log:     core.Log,
		Plugins: NewPlugins(),
	}

	con.NewConsole()
	return con, nil
}

type Console struct {
	//*core.ActiveTarget
	*core.ServerStatus
	*Plugins
	Log     *core.Logger
	App     *console.Console
	Profile *assets.Profile
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
	intermediate.RegisterBuiltin(c.Rpc)
	//c.App.Menu(consts.ClientMenu).SetCommands(bindCmds[0](c))
	//c.App.Menu(consts.ImplantMenu).SetCommands(bindCmds[1](c))
	c.App.Menu(consts.ClientMenu).Command = bindCmds[0](c)()
	c.App.Menu(consts.ImplantMenu).Command = bindCmds[1](c)()
	c.App.SwitchMenu(consts.ClientMenu)
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

func (c *Console) GetSession(sessionID string) *core.Session {
	if sess, ok := c.Sessions[sessionID]; ok {
		return sess
	}
	return nil
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

func (c *Console) SwitchImplant(sess *core.Session) {
	c.ActiveTarget.Set(sess)
	c.App.SwitchMenu(consts.ImplantMenu)

	var count int
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
		if cmd.Annotations["menu"] == consts.ImplantMenu && cmd.Hidden == false {
			count++
		}
	}
	c.Log.Importantf("os: %s, arch: %s, process: %d %s, pipeline: %s", sess.Os.Name, sess.Os.Arch, sess.Process.Ppid, sess.Process.Name, sess.ListenerId)
	c.Log.Importantf("%d modules, %d available cmds, %d addons", len(sess.Modules), count, len(sess.Addons.Addons))
	c.Log.Infof("Active session %s (%s), group: %s\n", sess.Note, sess.SessionId, sess.GroupName)
}

func (c *Console) RegisterImplantFunc(name string, fn interface{},
	bname string, bfn interface{},
	pluginCallback ImplantPluginCallback, implantCallback intermediate.ImplantCallback) {

	if implantCallback == nil {
		implantCallback = WrapImplantCallback(pluginCallback)
	}

	if fn != nil {
		intermediate.RegisterInternalFunc(intermediate.BuiltinPackage, name, WrapImplantFunc(c, fn, pluginCallback), implantCallback)
	}

	if bfn != nil {
		intermediate.RegisterInternalFunc(intermediate.BeaconPackage, bname, WrapImplantFunc(c, bfn, pluginCallback), implantCallback)
	}
}

func (c *Console) RegisterServerFunc(name string, fn interface{}) error {
	return intermediate.RegisterInternalFunc(intermediate.BuiltinPackage, name, WrapServerFunc(c, fn), nil)
}
