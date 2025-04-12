package repl

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/kballard/go-shellquote"
	"github.com/mattn/go-tty"
	"github.com/muesli/termenv"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
	"os"
	"time"
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
		sessionPrompt = fmt.Sprintf("%s [%s]> ", core.GroupStyle.Render(prePrompt), core.NameStyle.Render(string(runes)))
	} else {
		sessionPrompt = fmt.Sprintf("%s [%s]> ", core.GroupStyle.Render(prePrompt), core.NameStyle.Render(string(runes)))
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

func RunCommand(con *Console, cmdline interface{}) (string, error) {
	var args []string
	var err error
	switch c := cmdline.(type) {
	case string:
		args, err = shellquote.Split(c)
		if err != nil {
			return "", err
		}
	case []string:
		args = c
	}
	start := time.Now()
	err = con.App.Execute(con.Context(), con.App.ActiveMenu(), args, false)
	if err != nil {
		return "", err
	}
	return core.RemoveANSI(core.Stdout.Range(start, time.Now())), nil
}
