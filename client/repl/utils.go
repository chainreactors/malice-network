package repl

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-tty"
	"github.com/muesli/termenv"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
	"os"
)

var (
	Debug           logs.Level = 10
	Warn            logs.Level = 20
	Info            logs.Level = 30
	Error           logs.Level = 40
	Important       logs.Level = 50
	GroupStyle                 = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
	NameStyle                  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6"))
	DefaultLogStyle            = map[logs.Level]string{
		Debug:     termenv.String(tui.Rocket+"[+]").Bold().Background(tui.Blue).String() + " %s ",
		Warn:      termenv.String(tui.Zap+"[warn]").Bold().Background(tui.Yellow).String() + " %s ",
		Important: termenv.String(tui.Fire+"[*]").Bold().Background(tui.Purple).String() + " %s ",
		Info:      termenv.String(tui.HotSpring+"[i]").Bold().Background(tui.Green).String() + " %s ",
		Error:     termenv.String(tui.Monster+"[-]").Bold().Background(tui.Red).String() + " %s ",
	}
)

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
	root.SetArgs([]string{consts.CommandBackground})
	root.Execute()
}

func CmdExist(cmd *cobra.Command, name string) bool {
	for _, c := range cmd.Commands() {
		if name == c.Name() {
			return true
		}
	}
	return false
}

func GetCmd(cmd *cobra.Command, name string) *cobra.Command {
	for _, c := range cmd.Commands() {
		if name == c.Name() {
			return c
		}
	}
	return nil

}

func Login(con *Console, config *mtls.ClientConfig) error {
	conn, err := mtls.Connect(config)
	if err != nil {
		logs.Log.Errorf("Failed to connect: %v", err)
		return err
	}
	logs.Log.Importantf("Connected to server %s", config.Address())
	con.ServerStatus, err = core.InitServerStatus(conn, config)
	if err != nil {
		logs.Log.Errorf("init server failed : %v", err)
		return err
	}
	go con.EventHandler()
	var pipelineCount = 0
	for _, i := range con.Listeners {
		pipelineCount = pipelineCount + len(i.Pipelines.Pipelines)
	}
	var alive = 0
	for _, i := range con.Sessions {
		if i.IsAlive {
			alive++
		}
	}
	logs.Log.Importantf("%d listeners, %d pipelines, %d clients, %d sessions (%d alive)",
		len(con.Listeners), pipelineCount, len(con.Clients), len(con.Sessions), alive)
	return nil
}

func NewConfigLogin(con *Console, yamlFile string) error {
	config, err := mtls.ReadConfig(yamlFile)
	if err != nil {
		return err
	}
	err = Login(con, config)
	if err != nil {
		return err
	}
	err = assets.MvConfig(yamlFile)
	if err != nil {
		return err
	}
	return nil
}

func AdaptSessionColor(prePrompt, sId string) string {
	var sessionPrompt string
	runes := []rune(sId)
	if termenv.HasDarkBackground() {
		sessionPrompt = fmt.Sprintf("\033[37m%s [%s]> \033[0m", prePrompt, string(runes))
	} else {
		sessionPrompt = fmt.Sprintf("\033[30m%s [%s]> \033[0m", prePrompt, string(runes))
	}
	return sessionPrompt
}

func NewSessionColor(prePrompt, sId string) string {
	var sessionPrompt string
	runes := []rune(sId)
	if termenv.HasDarkBackground() {
		sessionPrompt = fmt.Sprintf("%s [%s]> ", GroupStyle.Render(prePrompt), NameStyle.Render(string(runes)))
	} else {
		sessionPrompt = fmt.Sprintf("%s [%s]> ", GroupStyle.Render(prePrompt), NameStyle.Render(string(runes)))
	}
	return sessionPrompt
}

// From the x/exp source code - gets a slice of keys for a map
func Keys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}

	return r
}
