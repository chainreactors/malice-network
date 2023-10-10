package console

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/desertbit/grumble"
	"github.com/fatih/color"
	"path/filepath"
	"sync"
)

type GRPCOptions struct {
}

type MTLSOptions struct {
}

type SessionOptions struct {
}

type GenerateOptions struct {
}

// Observer - A function to call when the sessions changes
type Observer func(*clientpb.Malefic, *clientpb.Malignant)

//type BeaconTaskCallback func(*clientpb.BeaconTask)

type ActiveTarget struct {
	session    *clientpb.Malefic
	beacon     *clientpb.Malignant
	observers  map[int]Observer
	observerID int
}

type Console struct {
	App          *grumble.App
	Rpc          clientrpc.MaliceRPCClient
	ActiveTarget *ActiveTarget
	//BeaconTaskCallbacks      map[string]BeaconTaskCallback
	BeaconTaskCallbacksMutex *sync.Mutex
	IsServer                 bool
	Settings                 *assets.Settings
}

// BindCmds - Bind extra commands to the app object
type BindCmds func(console *Console)

// Start - Console entrypoint
func Start(rpc clientrpc.MaliceRPCClient, bindCmds BindCmds, isServer bool) error {
	//assets.Setup(false, false)
	settings, _ := assets.LoadSettings()
	con := &Console{
		App: grumble.New(&grumble.Config{
			Name:                  "IoM",
			Description:           "Internet of Malice",
			HistoryFile:           filepath.Join(assets.GetRootAppDir(), "history"),
			PromptColor:           color.New(),
			HelpHeadlineColor:     color.New(),
			HelpHeadlineUnderline: true,
			HelpSubCommands:       true,
			//VimMode:               settings.VimMode,
		}),
		Rpc: rpc,
		ActiveTarget: &ActiveTarget{
			observers:  map[int]Observer{},
			observerID: 0,
		},
		//BeaconTaskCallbacks:      map[string]BeaconTaskCallback{},
		BeaconTaskCallbacksMutex: &sync.Mutex{},
		IsServer:                 isServer,
		Settings:                 settings,
	}
	con.App.SetPrintASCIILogo(func(_ *grumble.App) {
		//con.PrintLogo()
	})
	//con.App.SetPrompt(con.GetPrompt())
	bindCmds(con)
	//extraCmds(con)

	//con.ActiveTarget.AddObserver(func(_ *clientpb.Session, _ *clientpb.Beacon) {
	//	con.App.SetPrompt(con.GetPrompt())
	//})

	//go con.EventLoop()
	//go core.TunnelLoop(rpc)

	err := con.App.Run()
	if err != nil {
		logs.Log.Errorf("Run loop returned error: %v", err)
	}
	return err
}
