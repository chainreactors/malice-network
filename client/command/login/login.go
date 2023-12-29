package login

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
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

func (m *model) Init() tea.Cmd {
	m.selectedItem = -1
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			}
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *model) View() string {
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

func LoginCmd(ctx *grumble.Context, con *console.Console) error {
	files, err := getYAMLFiles(assets.GetConfigDir())
	if err != nil {
		con.App.Println("获取 YAML 文件时发生错误:", err)
		return err
	}

	// 为交互式列表创建模型
	m := &model{
		choices: files,
	}

	// 启动交互式列表
	p := tea.NewProgram(m)
	if err := p.Start(); err != nil {
		con.App.Println("启动交互式列表时发生错误:", err)
		return err
	}

	// 在交互式列表完成后，检查所选项目
	if m.selectedItem >= 0 && m.selectedItem < len(m.choices) {
		err := loginServer(ctx, con, m.choices[m.selectedItem])
		if err != nil {
			fmt.Println("执行 loginServer 时发生错误:", err)
		}
	}
	return nil
}

func loginServer(ctx *grumble.Context, con *console.Console, selectedFile string) error {
	configFile := filepath.Join(assets.GetConfigDir(), selectedFile)
	config, err := assets.ReadConfig(configFile)
	if err != nil {
		con.App.Println("Error reading config file:", err)
		return err
	}
	err = con.Login(config)
	if err != nil {
		con.App.Println("Error login:", err)
		return err
	}
	req := &clientpb.LoginReq{
		Name:  config.Operator,
		Host:  config.LHost,
		Port:  uint32(config.LPort),
		Token: config.Token,
	}
	res, err := con.Rpc.LoginClient(context.Background(), req)
	if err != nil {
		con.App.Println("Error login server: ", err)
		return err
	}
	if res.Success != true {
		con.App.Println("Error login server")
		return err
	}
	return nil
}
