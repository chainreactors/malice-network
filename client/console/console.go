package console

import (
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/mattn/go-tty"
	"github.com/reeflective/console"
	"github.com/reeflective/readline"
	"google.golang.org/protobuf/proto"
	"io"
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
	}
	con.NewConsole(bindCmds...)
	//con.App.SetPrintASCIILogo(func(_ *grumble.App) {
	//con.PrintLogo()
	//})
	//con.UpdatePrompt()

	con.readConfig()

	con.ActiveTarget.callback = func(sess *clientpb.Session) {
		con.ActiveTarget.activeObserver = NewObserver(sess)
	}
	con.App.SwitchMenu(consts.ClientMenu)
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
	logs.Log.Importantf("Connected to server %s", config.Address())
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
	open, err := tty.Open()
	if err != nil {
		panic(err)
	}
	defer open.Close()

	fmt.Print("Press 'Y/y'  or 'Ctrl+C' to confirm exit: ")

	for {
		readRune, err := open.ReadRune()
		if err != nil {
			panic(err)
		}

		switch readRune {
		case 'Y', 'y':
			os.Exit(0)
		case 3: // ASCII code for Ctrl+C
			os.Exit(0)
		}
	}
}

// exitImplantMenu uses the background command to detach from the implant menu.
func (c *Console) exitImplantMenu(_ *console.Console) {
	root := c.App.Menu(consts.ImplantMenu).Command
	root.SetArgs([]string{"background"})
	root.Execute()
}

func (c *Console) NewConsole(bindCmds ...BindCmds) {
	iom := console.New("IoM")
	iom.NewlineBefore = true
	iom.NewlineAfter = true
	c.App = iom

	client := iom.NewMenu(consts.ClientMenu)
	client.Short = "client commands"
	client.Prompt().Primary = c.GetPrompt
	client.AddInterrupt(readline.ErrInterrupt, c.exitConsole)
	client.AddHistorySourceFile("history", filepath.Join(assets.GetRootAppDir(), "history"))
	client.Command = bindCmds[0](c)()

	implant := iom.NewMenu(consts.ImplantMenu)
	implant.Short = "Implant commands"
	implant.Prompt().Primary = c.GetPrompt
	implant.AddInterrupt(io.EOF, c.exitImplantMenu) // Ctrl-D
	implant.AddHistorySourceFile("history", filepath.Join(assets.GetRootAppDir(), "implant_history"))
	implant.Command = bindCmds[1](c)()
}
