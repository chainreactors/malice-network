package console

import (
	"errors"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/fatih/color"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
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
type BindCmds func(console *Console)

// Start - Console entrypoint
func Start(bindCmds ...BindCmds) error {
	//assets.Setup(false, false)
	tui.Reset()
	settings, _ := assets.LoadSettings()
	con := &Console{
		App: grumble.New(&grumble.Config{
			Name:                  tui.AdaptTermColor(Prompt),
			Description:           "Internet of Malice",
			HistoryFile:           filepath.Join(assets.GetRootAppDir(), "history"),
			PromptColor:           color.New(),
			HelpHeadlineColor:     color.New(),
			HelpHeadlineUnderline: true,
			HelpSubCommands:       true,
			//VimMode:               settings.VimMode,
		}),
		ActiveTarget: &ActiveTarget{},
		Settings:     settings,
		Observers:    map[string]*Observer{},
	}
	//con.App.SetPrintASCIILogo(func(_ *grumble.App) {
	//con.PrintLogo()
	//})
	//con.UpdatePrompt()
	con.readConfig()
	for _, bind := range bindCmds {
		bind(con)
	}
	con.DisableImplantCommands()
	con.ActiveTarget.callback = func(sess *clientpb.Session) {
		con.ActiveTarget.activeObserver = NewObserver(sess)
		con.UpdatePrompt()
	}

	//go core.TunnelLoop(rpc)
	os.Args = []string{}
	err := con.App.Run()
	if err != nil {
		logs.Log.Errorf("Run loop returned error: %v", err)
	}
	return err
}

type Console struct {
	App          *grumble.App
	ActiveTarget *ActiveTarget
	Settings     *assets.Settings
	Callbacks    *sync.Map
	Observers    map[string]*Observer
	*ServerStatus
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

func (c *Console) UpdatePrompt() {
	c.App.Config().NoColor = true
	if c.ActiveTarget.session != nil {
		groupName := c.ActiveTarget.session.GroupName
		if c.ActiveTarget.session.Note != "" {
			c.App.SetPrompt(tui.AdaptSessionColor(groupName, c.ActiveTarget.session.Note))
		} else {
			sessionID := c.ActiveTarget.session.SessionId
			c.App.SetPrompt(tui.AdaptSessionColor(groupName, sessionID[:8]))
		}

	} else {
		c.App.SetPrompt(tui.AdaptTermColor(Prompt))
	}
}

func (c *Console) AddAliasCommand(cmd *grumble.Command) {
	group := c.App.Groups().Find(consts.AliasesGroup)
	group.AddCommand(cmd)
}

func (c *Console) AddExtensionCommand(cmd *grumble.Command) {
	group := c.App.Groups().Find(consts.ExtensionGroup)
	group.AddCommand(cmd)
}

func (c *Console) EnableImplantCommands() {
	for _, g := range implantGroups {
		c.App.Groups().Find(g).Enable()
	}
}

func (c *Console) DisableImplantCommands() {
	for _, g := range implantGroups {
		c.App.Groups().Find(g).Disable()
	}
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
