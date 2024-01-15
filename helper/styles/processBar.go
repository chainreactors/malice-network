package styles

import (
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"time"
)

const (
	padding  = 2
	maxWidth = 60
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

type tickMsg time.Time

type ProcessBarModel struct {
	Progress        progress.Model
	ProgressPercent float64
}

func (m ProcessBarModel) Init() tea.Cmd {
	return tickCmd()
}

func (m ProcessBarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.Progress.Width = msg.Width - padding*2 - 4
		if m.Progress.Width > maxWidth {
			m.Progress.Width = maxWidth
		}
		return m, nil

	case tickMsg:
		if m.Progress.Percent() == 1.0 {
			return m, tea.Quit
		}
		m.ProgressPercent += 0.1
		// Note that you can also use Progress.Model.SetPercent to set the
		// percentage value explicitly, too.
		cmd := m.Progress.SetPercent(m.ProgressPercent)
		return m, tea.Batch(tickCmd(), cmd)

	// FrameMsg is sent when the Progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.Progress.Update(msg)
		m.Progress = progressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}
}

func (m ProcessBarModel) View() string {
	pad := strings.Repeat(" ", padding)
	return "\n" +
		pad + m.Progress.View() + "\n\n" +
		pad + helpStyle("Press any key to quit")
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
