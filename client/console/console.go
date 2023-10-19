package console

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/desertbit/grumble"
	"github.com/fatih/color"
	"google.golang.org/grpc"
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

type Console struct {
	App                      *grumble.App
	Rpc                      clientrpc.MaliceRPCClient
	ActiveTarget             *ActiveTarget
	BeaconTaskCallbacksMutex *sync.Mutex
	Settings                 *assets.Settings
}

func (c *Console) Login(config *assets.ClientConfig) {
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", config.LHost, config.LPort), grpc.WithInsecure())

	if err != nil {
		logs.Log.Errorf("Failed to connect: %v", err)
	}
	logs.Log.Importantf("Connected to server grpc %s:%d", config.LHost, config.LPort)
	c.Rpc = clientrpc.NewMaliceRPCClient(conn)
}

// BindCmds - Bind extra commands to the app object
type BindCmds func(console *Console)

// Start - Console entrypoint
func Start(bindCmds BindCmds) error {
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
		ActiveTarget: &ActiveTarget{
			observers:  map[int]Observer{},
			observerID: 0,
		},
		//BeaconTaskCallbacks:      map[string]BeaconTaskCallback{},
		BeaconTaskCallbacksMutex: &sync.Mutex{},
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
