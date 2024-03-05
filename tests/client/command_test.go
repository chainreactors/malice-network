package client

import (
	"fmt"
	"github.com/erikgeiser/promptkit/confirmation"
	"os"
	"testing"
)

func TestConfirm(T *testing.T) {
	input := confirmation.New("Do you want to try out promptkit?",
		confirmation.NewValue(true))
	input.Template = confirmation.TemplateYN
	input.ResultTemplate = confirmation.ResultTemplateYN
	input.KeyMap.SelectYes = append(input.KeyMap.SelectYes, "+")
	input.KeyMap.SelectNo = append(input.KeyMap.SelectNo, "-")

	ready, err := input.RunPrompt()
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		os.Exit(1)
	}

	// do something with the result
	fmt.Println(ready)
}

//	columns := []table.Column{
//		{Title: "ID", Width: 4},
//		{Title: "Name", Width: 4},
//		{Title: "Transport", Width: 10},
//		{Title: "Remote Address", Width: 15},
//		{Title: "Hostname", Width: 10},
//		{Title: "Username", Width: 10},
//		{Title: "Operating System", Width: 20},
//		{Title: "Locale", Width: 10},
//		{Title: "Last Message", Width: 15},
//		{Title: "Health", Width: 10},
//	}
//
//	rows := []table.Row{
//		{"08d6c05a", "", "", "", "", "", "", "windows/", "Thu, 01 Jan 1970 08:00:00CST", "[ALIVE]"},
//	}
//
//	t := table.New(
//		table.WithColumns(columns),
//		table.WithRows(rows),
//		table.WithFocused(true),
//		table.WithHeight(7),
//	)
//
//	s := table.DefaultStyles()
//	s.Header = s.Header.
//		BorderStyle(lipgloss.NormalBorder()).
//		BorderForeground(lipgloss.Color("240")).
//		BorderBottom(true).
//		Bold(false)
//	s.Selected = s.Selected.
//		Foreground(lipgloss.Color("229")).
//		Background(lipgloss.Color("57")).
//		Bold(false)
//	t.SetStyles(s)
//
//	m := model{t}
//	if _, err := tea.NewProgram(m).Run(); err != nil {
//		fmt.Println("Error running program:", err)
//		os.Exit(1)
//	}
//}
//
//var baseStyle = lipgloss.NewStyle().
//	BorderStyle(lipgloss.NormalBorder()).
//	BorderForeground(lipgloss.Color("240"))
//
//type model struct {
//	table table.Model
//}
//
//func (m model) Init() tea.Cmd { return nil }
//
//func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
//	var cmd tea.Cmd
//	switch msg := msg.(type) {
//	case tea.KeyMsg:
//		switch msg.String() {
//		case "esc":
//			if m.table.Focused() {
//				m.table.Blur()
//			} else {
//				m.table.Focus()
//			}
//		case "q", "ctrl+c":
//			return m, tea.Quit
//		case "enter":
//			return m, tea.Batch(
//				tea.Printf("Let's go to %s!", m.table.SelectedRow()[1]),
//			)
//		}
//	}
//	m.table, cmd = m.table.Update(msg)
//	return m, cmd
//}
//
//func (m model) View() string {
//	return baseStyle.Render(m.table.View()) + "\n"
//}

//const (
//	padding  = 2
//	maxWidth = 60
//)
//
//var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render
//
//func TestCommand(t *testing.T) {
//	implant := common.NewImplant(common.DefaultListenerAddr, common.TestSid)
//	implant.Register()
//	time.Sleep(1 * time.Second)
//	fmt.Println(hash.Md5Hash([]byte(implant.Sid)))
//	go func() {
//
//		upload, err := implant.Request(nil)
//		if err != nil {
//			fmt.Println(err.Error())
//			return
//		}
//		taskid := upload.(* implantpb.Spites).Spites[0].TaskId
//		fmt.Printf("res %v %v\n", upload, err)
//		time.Sleep(1 * time.Second)
//
//		implant.Request(implant.BuildCommonSpite(common.StatusSpite, taskid))
//		time.Sleep(1 * time.Second)
//		block, err := implant.Request(nil)
//		if err != nil {
//			fmt.Println(err)
//			return
//		}
//		implant.Request(implant.BuildCommonSpite(common.AckSpite, taskid))
//		fmt.Println(block)
//	}()
//	meta := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("session_id", hash.Md5Hash(common.TestSid)))
//	rpc := common.NewClient(common.DefaultGRPCAddr, common.TestSid)
//	res, err := rpc.Client.Upload(meta, &implantpb.UploadRequest{
//		Name:   "test.txt",
//		Target: "C:\\Temp\\test.txt",
//		Priv:   0o644,
//		Data:   make([]byte, 1000)})
//	if err != nil {
//		fmt.Println(err)
//	}
//	m := model{
//		progress:       progress.New(progress.WithDefaultGradient()),
//		processPercent: float64(res.Cur / res.Total),
//	}
//
//	if _, err := tea.NewProgram(m).Run(); err != nil {
//		fmt.Println("Oh no!", err)
//		os.Exit(1)
//	}
//
//}
//
//type tickMsg time.Time
//
//type model struct {
//	progress       progress.Model
//	processPercent float64
//}
//
//func (m model) Init() tea.Cmd {
//	return tickCmd()
//}
//
//func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
//	switch msg := msg.(type) {
//	case tea.KeyMsg:
//		return m, tea.Quit
//
//	case tea.WindowSizeMsg:
//		m.progress.Width = msg.Width - padding*2 - 4
//		if m.progress.Width > maxWidth {
//			m.progress.Width = maxWidth
//		}
//		return m, nil
//
//	case tickMsg:
//		if m.progress.Percent() == 1.0 {
//			return m, tea.Quit
//		}
//		// Note that you can also use progress.Model.SetPercent to set the
//		// percentage value explicitly, too.
//		cmd := m.progress.SetPercent(m.processPercent)
//		return m, tea.Batch(tickCmd(), cmd)
//
//	// FrameMsg is sent when the progress bar wants to animate itself
//	case progress.FrameMsg:
//		progressModel, cmd := m.progress.Update(msg)
//		m.progress = progressModel.(progress.Model)
//		return m, cmd
//
//	default:
//		return m, nil
//	}
//}
//
//func (m model) View() string {
//	pad := strings.Repeat(" ", padding)
//	return "\n" +
//		pad + m.progress.View() + "\n\n" +
//		pad + helpStyle("Press any key to quit")
//}
//
//func tickCmd() tea.Cmd {
//	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
//		return tickMsg(t)
//	})
//}
