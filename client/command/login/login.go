package login

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/utils"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/desertbit/grumble"
	"os"
	"path/filepath"
	"strings"
)

type model struct {
	choices      []string
	selectedItem int
}

type msg string

const (
	msgQuit     msg = "quit"
	msgSelect   msg = "select"
	msgUnselect msg = "unselect"
)

func RunInteractiveList() error {
	files, err := getYAMLFiles(assets.GetConfigDir())
	if err != nil {
		return err
	}

	m := model{
		choices: files,
	}

	p := tea.NewProgram(m)
	if err := p.Start(); err != nil {
		return err
	}

	return nil
}

func getYAMLFiles(directory string) ([]string, error) {
	var files []string

	// 遍历指定目录下的所有文件
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 如果是文件且扩展名是 .yaml 或 .yml，则加入文件列表
		if !info.IsDir() && (strings.HasSuffix(info.Name(), ".yaml") || strings.HasSuffix(info.Name(), ".yml")) {
			files = append(files, info.Name())
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "up":
			m.selectedItem--
			if m.selectedItem < 0 {
				m.selectedItem = len(m.choices) - 1
			}
			return m, nil
		case "down":
			m.selectedItem++
			if m.selectedItem >= len(m.choices) {
				m.selectedItem = 0
			}
			return m, nil
		case "enter":
			if m.selectedItem >= 0 && m.selectedItem < len(m.choices) {
				fmt.Println("You selected:", m.choices[m.selectedItem])
				// 在这里可以处理选中文件的逻辑
			}
			return m, nil
		}
	}

	return m, nil
}

func (m model) View() string {
	var view strings.Builder

	for i, choice := range m.choices {
		if i == m.selectedItem {
			view.WriteString("[x] ")
		} else {
			view.WriteString("[ ] ")
		}
		view.WriteString(choice)
		view.WriteRune('\n')
	}

	return view.String()
}

func LoginCmd(ctx *grumble.Context, con *console.Console) {
	// TODO : interactive choice config
	//config := &assets.ClientConfig{
	//	LHost: "127.0.0.1",
	//	LPort: 5004,
	//}
	//err := con.Login(config)
	//if err != nil {
	//	return
	//}
	loginServer(ctx, con)
}

func loginServer(ctx *grumble.Context, con *console.Console) {
	configFile := ctx.Flags.String("config")
	config, err := assets.ReadConfig(configFile)
	if err != nil {
		con.App.Println("Error reading config file:", err)
		return
	}
	rpc, ln, err := utils.MTLSConnect(config)
	req := &clientpb.LoginReq{
		Name: config.Operator,
		Host: config.LHost,
		Port: uint32(config.LPort),
	}
	res, err := rpc.AddClient(context.Background(), req)
	if err != nil {
		con.App.Println("Error login server: ", err)
		return
	}
	defer ln.Close()
	if res.Success != true {
		con.App.Println("Error login server")
		return
	}
}
