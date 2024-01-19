package styles

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"time"
)

var MonkeyCatchMoon = spinner.Spinner{
	Frames: []string{"🌑", "🌒", "🙈", "🌓", "🌔", "🌕", "🙉", "🌖", "🌗", "🌘", "🙊"},
	FPS:    time.Second / 11, //nolint:gomnd
}

var (
	// Available spinners
	spinners = []spinner.Spinner{
		spinner.Line,
		spinner.Dot,
		spinner.MiniDot,
		spinner.Jump,
		spinner.Pulse,
		spinner.Points,
		spinner.Globe,
		spinner.Moon,
		spinner.Monkey,
		MonkeyCatchMoon,
	}
	textStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
)

type SpinnerModel struct {
	spinner  spinner.Model
	Quitting bool
	err      error
	index    int
}

func (s SpinnerModel) Init() tea.Cmd {
	s.index = 9
	return s.spinner.Tick
}

func (s SpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return s, tea.Quit
		case "h", "left":
			s.index--
			if s.index < 0 {
				s.index = len(spinners) - 1
			}
			s.resetSpinner()
			return s, s.spinner.Tick
		case "l", "right":
			s.index++
			if s.index >= len(spinners) {
				s.index = 0
			}
			s.resetSpinner()
			return s, s.spinner.Tick
		default:
			return s, nil
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd
	default:
		return s, nil
	}
}

func (s *SpinnerModel) resetSpinner() {
	s.spinner = spinner.New()
	s.spinner.Style = spinnerStyle
	s.spinner.Spinner = spinners[s.index]
}

func (s SpinnerModel) View() (str string) {
	if s.Quitting {
		return ""
	}
	var gap string
	switch s.index {
	case 1:
		gap = ""
	default:
		gap = " "
	}

	str += fmt.Sprintf("\n %s%s%s\n\n", s.spinner.View(), gap, textStyle("Spinning..."))
	str += helpStyle("h/l, ←/→: change spinner • q: exit\n")
	return
}

func (s SpinnerModel) Run() (tea.Model, error) {
	model, err := tea.NewProgram(s).Run()
	if err != nil {
		return nil, err
	}
	return model, nil
}
