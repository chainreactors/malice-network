package console

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/reeflective/console"
	"github.com/reeflective/readline"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"io"
	"os"
	"path/filepath"
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
	implantGroups      = []string{consts.ImplantGroup, consts.AliasesGroup, consts.ExtensionGroup}
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
		App:          console.New("IoM"),
		ActiveTarget: &ActiveTarget{},
		Settings:     settings,
		Observers:    map[string]*Observer{},
	}
	//con.App.SetPrintASCIILogo(func(_ *grumble.App) {
	//con.PrintLogo()
	//})
	//con.UpdatePrompt()
	con.App.NewlineBefore = true
	con.App.NewlineAfter = true
	client := con.App.NewMenu(consts.ClientGroup)
	client.Short = "client commands"
	client.Prompt().Primary = con.GetPrompt
	client.AddInterrupt(readline.ErrInterrupt, con.exitConsole)
	client.AddHistorySourceFile("history", filepath.Join(assets.GetRootAppDir(), "history"))

	implant := con.App.NewMenu(consts.ImplantGroup)
	implant.Short = "Implant commands"
	implant.Prompt().Primary = con.GetPrompt
	implant.AddInterrupt(io.EOF, con.exitImplantMenu) // Ctrl-D
	implant.AddHistorySourceFile("history", filepath.Join(assets.GetRootAppDir(), "implant_history"))
	con.readConfig()
	client.SetCommands(bindCmds[0](con))
	for _, bindCmd := range bindCmds {
		implant.SetCommands(bindCmd(con))
	}

	con.ActiveTarget.callback = func(sess *clientpb.Session) {
		con.ActiveTarget.activeObserver = NewObserver(sess)
	}
	con.Plugins = NewPlugins()

	con.App.SwitchMenu(consts.ClientGroup)
	err := con.App.Start()
	if err != nil {
		logs.Log.Errorf("Run loop returned error: %v", err)
	}
	return err
}

type Console struct {
	App          *console.Console
	ActiveTarget *ActiveTarget
	Settings     *assets.Settings
	Callbacks    *sync.Map
	Observers    map[string]*Observer
	*ServerStatus
	*Plugins
}

func (c *Console) Login(config *mtls.ClientConfig) error {
	conn, err := mtls.Connect(config)
	if err != nil {
		logs.Log.Errorf("Failed to connect: %v", err)
		return err
	}
	logs.Log.Importantf("Connected to server %s:%d", config.LHost, config.LPort)
	c.ServerStatus, err = InitServerStatus(conn)
	if err != nil {
		logs.Log.Errorf("init server failed : %v", err)
		return err
	}
	logs.Log.Importantf("%d listeners, %d clients , %d sessions", len(c.Listeners), len(c.Clients), len(c.Sessions))
	return nil
}

//func (c *Console) UpdatePrompt() {
//	c.App.Config().NoColor = true
//	if c.ActiveTarget.session != nil {
//		groupName := c.ActiveTarget.session.GroupName
//		if c.ActiveTarget.session.Note != "" {
//			c.App.SetPrompt(tui.AdaptSessionColor(groupName, c.ActiveTarget.session.Note))
//		} else {
//			sessionID := c.ActiveTarget.session.SessionId
//			c.App.SetPrompt(tui.AdaptSessionColor(groupName, sessionID[:8]))
//		}
//
//	} else {
//		c.App.SetPrompt(tui.AdaptTermColor(Prompt))
//	}
//}

func (c *Console) GetPrompt() string {
	session := c.ActiveTarget.Get()
	if session != nil {
		groupName := session.GroupName
		if session.Note != "" {
			return tui.NewSessionColor(groupName, session.Note)
		} else {
			sessionID := session.SessionId
			return tui.NewSessionColor(groupName, sessionID[:8])
		}

	} else {
		return tui.AdaptTermColor("IOM")
	}
}

func (c *Console) AddAliasCommand(cmd *cobra.Command) {
	found := false
	for _, grp := range c.App.ActiveMenu().Groups() {
		if grp.Title == consts.AliasesGroup {
			found = true
			break
		}

		if !found {
			c.App.ActiveMenu().AddGroup(&cobra.Group{
				ID:    consts.AliasesGroup,
				Title: consts.AliasesGroup,
			})
		}
	}
	found = false
	for _, grp := range c.App.Menu(consts.ImplantGroup).Groups() {
		if grp.Title == consts.AliasesGroup {
			found = true
			break
		}

		if !found {
			c.App.Menu(consts.ImplantGroup).AddGroup(&cobra.Group{
				ID:    consts.AliasesGroup,
				Title: consts.AliasesGroup,
			})
		}
	}
	c.App.ActiveMenu().AddCommand(cmd)
	c.App.Menu(consts.ImplantGroup).AddCommand(cmd)
}

func (c *Console) AddExtensionCommand(cmd *cobra.Command) {
	found := false
	for _, grp := range c.App.ActiveMenu().Groups() {
		if grp.Title == consts.ExtensionGroup {
			found = true
			break
		}

		if !found {
			c.App.ActiveMenu().AddGroup(&cobra.Group{
				ID:    consts.ExtensionGroup,
				Title: consts.ExtensionGroup,
			})
		}
	}
	found = false
	for _, grp := range c.App.Menu(consts.ImplantGroup).Groups() {
		if grp.Title == consts.ExtensionGroup {
			found = true
			break
		}

		if !found {
			c.App.Menu(consts.ImplantGroup).AddGroup(&cobra.Group{
				ID:    consts.ExtensionGroup,
				Title: consts.ExtensionGroup,
			})
		}
	}
	c.App.ActiveMenu().AddCommand(cmd)
	c.App.Menu(consts.ImplantGroup).AddCommand(cmd)
}

// AddObserver - Observers to notify when the active session changes
func (c *Console) AddObserver(session *clientpb.Session) string {
	Log.Infof("Add observer to %s", session.SessionId)
	c.Observers[session.SessionId] = NewObserver(session)
	return session.SessionId
}

func (c *Console) RemoveObserver(observerID string) {
	delete(c.Observers, observerID)
}

func (c *Console) GetInteractive() *clientpb.Session {
	if c.ActiveTarget != nil {
		return c.ActiveTarget.GetInteractive()
	}
	return nil
}

func (c *Console) RefreshActiveSession() {
	if c.ActiveTarget != nil {
		c.UpdateSession(c.ActiveTarget.session.SessionId)
	}
}

func (c *Console) SessionLog(sid string) *logs.Logger {
	if ob, ok := c.Observers[sid]; ok {
		return ob.log
	} else if c.ActiveTarget.GetInteractive() != nil {
		return c.ActiveTarget.activeObserver.log
	} else {
		return MuteLog
	}
}

// readConfig
func (c *Console) readConfig() {
	var yamlFile string

	if len(os.Args) > 1 {
		yamlFile = os.Args[1]
	} else {
		return
	}
	clientFile, err := mtls.ReadConfig(yamlFile)
	if err != nil {
		logs.Log.Errorf("Error reading config file: %v", err)
		return
	}
	err = c.Login(clientFile)
	if err != nil {
		logs.Log.Errorf("Error login: %v", err)
		return
	}
	err = assets.MvConfig(yamlFile)
	if err != nil {
		return
	}
}

func (c *Console) exitConsole(_ *console.Console) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Confirm exit (Y/y): ")
	text, _ := reader.ReadString('\n')
	answer := strings.TrimSpace(text)

	if (answer == "Y") || (answer == "y") {
		os.Exit(0)
	}
}

// exitImplantMenu uses the background command to detach from the implant menu.
func (c *Console) exitImplantMenu(_ *console.Console) {
	root := c.App.Menu(consts.ImplantGroup).Command
	root.SetArgs([]string{"background"})
	root.Execute()
}
