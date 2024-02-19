package tui

import (
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	"time"
)

const (
	padding  = 2
	maxWidth = 60
)

type progressWriter struct {
	total      int
	completed  int
	onProgress func(float64)
}

func NewBar(total int) *BarModel {
	bar := &BarModel{
		progress: progress.New(progress.WithDefaultGradient()),
		pw: &progressWriter{
			total: total,
		},
	}
	return bar
}

type progressMsg float64

type progressErrMsg struct{ err error }

func finalPause() tea.Cmd {
	return tea.Tick(time.Millisecond*750, func(_ time.Time) tea.Msg {
		return nil
	})
}

type BarModel struct {
	pw       *progressWriter
	progress progress.Model
	err      error
}

func (m *BarModel) Init() tea.Cmd {
	return nil
}

func (m *BarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
		return m, nil

	case progressErrMsg:
		m.err = msg.err
		return m, tea.Quit

	case progressMsg:
		var cmds []tea.Cmd

		if msg >= 1.0 {
			cmds = append(cmds, tea.Sequence(finalPause(), tea.Quit))
		}

		cmds = append(cmds, m.progress.SetPercent(float64(msg)))
		return m, tea.Batch(cmds...)

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}
}

func (m *BarModel) View() string {
	if m.err != nil {
		return m.err.Error()
	}

	pad := strings.Repeat(" ", padding)
	return "\n" +
		pad + m.progress.View() + "\n\n" +
		pad + HelpStyle("Press any key to quit")
}

func (m *BarModel) SetOnProgress(p *tea.Program) {
	m.pw.onProgress = func(f float64) {
		p.Send(progressMsg(f))
	}
}

func (m *BarModel) Incr() {
	m.pw.completed++
	if m.pw.total > 0 && m.pw.onProgress != nil {
		m.pw.onProgress(float64(m.pw.completed) / float64(m.pw.total))
	}
}
