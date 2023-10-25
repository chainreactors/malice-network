package console

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/desertbit/grumble"
	"github.com/fatih/color"
	"google.golang.org/grpc"
	"path/filepath"
)

var Log = logs.NewLogger(logs.Warn)

// BindCmds - Bind extra commands to the app object
type BindCmds func(console *Console)

// Start - Console entrypoint
func Start(bindCmds BindCmds) error {
	//assets.Setup(false, false)
	settings, _ := assets.LoadSettings()
	con := &Console{
		App: grumble.New(&grumble.Config{
			Name:                  consts.ClientPrompt,
			Description:           "Internet of Malice",
			HistoryFile:           filepath.Join(assets.GetRootAppDir(), "history"),
			PromptColor:           color.New(),
			HelpHeadlineColor:     color.New(),
			HelpHeadlineUnderline: true,
			HelpSubCommands:       true,
			//VimMode:               settings.VimMode,
		}),
		ActiveTarget: &ActiveTarget{
			observers:  map[int]Observer{},
			observerID: 0,
		},
		//BeaconTaskCallbacks:      map[string]BeaconTaskCallback{},
		//BeaconTaskCallbacksMutex: &sync.Mutex{},
		Settings: settings,
	}
	con.App.SetPrintASCIILogo(func(_ *grumble.App) {
		//con.PrintLogo()
	})
	con.UpdatePrompt()
	bindCmds(con)
	//extraCmds(con)

	con.ActiveTarget.AddObserver(func(_ *clientpb.Session) {
		con.UpdatePrompt()
	})

	//go con.EventLoop()
	//go core.TunnelLoop(rpc)

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
	*ServerStatus
}

func (c *Console) Login(config *assets.ClientConfig) error {
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", config.LHost, config.LPort), grpc.WithInsecure())

	if err != nil {
		logs.Log.Errorf("Failed to connect: %v", err)
		return err
	}
	logs.Log.Importantf("Connected to server grpc %s:%d", config.LHost, config.LPort)
	c.ServerStatus, err = InitServerStatus(conn)
	if err != nil {
		logs.Log.Errorf("init server failed : %v", err)
		return err
	}
	logs.Log.Importantf("%d listeners, %d clients , %d sessions", len(c.Listeners), len(c.Clients), len(c.Sessions))
	return nil
}

func (c *Console) UpdatePrompt() {
	if c.ActiveTarget.session != nil {
		c.App.SetPrompt(fmt.Sprintf("%s [%s] > ", consts.ClientPrompt, helper.ShortSessionID(c.ActiveTarget.session.SessionId)))
	} else {
		c.App.SetPrompt(consts.ClientPrompt + " > ")
	}
}
