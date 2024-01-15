package console

import (
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/helper"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/fatih/color"
	"path/filepath"
	"sync"
)

const (
	// ANSI Colors
	Normal    = "\033[0m"
	Black     = "\033[30m"
	Red       = "\033[31m"
	Green     = "\033[32m"
	Orange    = "\033[33m"
	Blue      = "\033[34m"
	Purple    = "\033[35m"
	Cyan      = "\033[36m"
	Gray      = "\033[37m"
	Bold      = "\033[1m"
	Clearln   = "\r\x1b[2K"
	UpN       = "\033[%dA"
	DownN     = "\033[%dB"
	Underline = "\033[4m"

	// Info - Display colorful information
	Info = Bold + Cyan + "[*] " + Normal
	// Warn - Warn a user
	Warn = Bold + Red + "[!] " + Normal
	// Debug - Display debug information
	Debug = Bold + Purple + "[-] " + Normal
	// Woot - Display success
	Woot = Bold + Green + "[$] " + Normal
	// Success - Diplay success
	Success = Bold + Green + "[+] " + Normal
)

var Log = logs.NewLogger(logs.Warn)

type TaskCallback func(task *clientpb.Task)

// BindCmds - Bind extra commands to the app object
type BindCmds func(console *Console)

// Start - Console entrypoint
func Start(bindCmds ...BindCmds) error {
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
		Settings: settings,
	}
	con.App.SetPrintASCIILogo(func(_ *grumble.App) {
		//con.PrintLogo()
	})
	con.UpdatePrompt()
	for _, bind := range bindCmds {
		bind(con)
	}

	con.ActiveTarget.AddObserver(func(_ *clientpb.Session) {
		con.UpdatePrompt()
	})

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
	Callbacks    *sync.Map
	*ServerStatus
}

func (c *Console) Login(config *assets.ClientConfig) error {
	conn, err := mtls.Connect(config)
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

func (c *Console) AddAliasCommand(cmd *grumble.Command) {
	group := c.App.Groups().Find(consts.AliasesGroup)
	group.AddCommand(cmd)
}
