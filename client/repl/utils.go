package repl

import (
	"fmt"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
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
		sessionPrompt = fmt.Sprintf("%s [%s]> ", client.GroupStyle.Render(prePrompt), client.NameStyle.Render(string(runes)))
	} else {
		sessionPrompt = fmt.Sprintf("%s [%s]> ", client.GroupStyle.Render(prePrompt), client.NameStyle.Render(string(runes)))
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

	// 智能路由：如果有活动的 session，使用 ImplantMenu，否则使用 ClientMenu
	menu := con.App.Menu(consts.ClientMenu)
	if con.ActiveTarget != nil && con.ActiveTarget.Get() != nil {
		menu = con.App.Menu(consts.ImplantMenu)
	}

	// 自动为带有 isStatic 注解的命令添加 --static flag
	args = autoAddStaticFlag(menu.Command, args)

	err = con.App.Execute(con.Context(), menu, args, false)
	if err != nil {
		return "", err
	}
	return client.RemoveANSI(client.Stdout.Range(start, time.Now())), nil
}

// autoAddStaticFlag 自动为定义了 --static flag 的命令添加该 flag
// 这样可以避免交互式命令在 MCP 中超时
func autoAddStaticFlag(rootCmd *cobra.Command, args []string) []string {
	if len(args) == 0 {
		return args
	}

	// 查找命令
	cmd, _, err := rootCmd.Find(args)
	if err != nil || cmd == nil {
		return args
	}

	// 检查命令是否定义了 --static flag
	staticFlag := cmd.Flags().Lookup("static")
	if staticFlag != nil {
		// 检查是否已经有 --static flag
		hasStatic := false
		for _, arg := range args {
			if arg == "--static" {
				hasStatic = true
				break
			}
		}

		// 如果没有 --static flag，自动添加
		if !hasStatic {
			args = append(args, "--static")
		}
	}

	return args
}
