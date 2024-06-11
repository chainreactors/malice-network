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

func NewBar() *BarModel {
	bar := &BarModel{
		Model: progress.New(progress.WithDefaultGradient()),
	}
	return bar
}

type progressMsg float64

type progressErrMsg struct{ err error }

type ViewMsg struct {
}

func finalPause() tea.Cmd {
	return tea.Tick(time.Millisecond*750, func(_ time.Time) tea.Msg {
		return nil
	})
}

type BarModel struct {
	Model           progress.Model
	progressPercent float64
	err             error
}

func (m *BarModel) Init() tea.Cmd {
	return setPercentMsg(m.progressPercent)
}

func (m *BarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.Model.Width = msg.Width - padding*2 - 4
		if m.Model.Width > maxWidth {
			m.Model.Width = maxWidth
		}
		return m, nil

	case progressErrMsg:
		m.err = msg.err
		return m, tea.Quit

	case progressMsg:
		//var cmds []tea.Cmd
		//
		//if msg >= 1.0 {
		//	cmds = append(cmds, tea.Sequence(finalPause(), tea.Quit))
		//}
		//
		//cmds = append(cmds, m.Model.SetPercent(float64(msg)))

		//return m, tea.Batch(cmds...)
		if m.Model.Percent() == 1.0 {
			return m, tea.Quit
		}
		m.progressPercent = float64(msg)
		cmd := m.Model.SetPercent(m.progressPercent)
		return m, tea.Batch(setPercentMsg(m.progressPercent), cmd)
	// FrameMsg is sent when the Model bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.Model.Update(msg)
		m.Model = progressModel.(progress.Model)
		return m, cmd
	case ViewMsg:
		m.View()
		return m, nil
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
		pad + m.Model.ViewAs(m.progressPercent) + "\n\n" +
		pad + HelpStyle("Press any key to quit")
}

func setPercentMsg(percent float64) tea.Cmd {
	return func() tea.Msg {
		return progressMsg(percent)
	}
}

func (m *BarModel) SetProgressPercent(percent float64) {
	m.progressPercent = percent
}

//func (m *BarModel) SetOnProgress(p *tea.Program) {
//	m.pw.onProgress = func(f float64) {
//		p.Send(progressMsg(f))
//	}
//}

//func (m *BarModel) Incr() {
//	m.pw.completed++
//	if m.pw.total > 0 && m.pw.onProgress != nil {
//		m.pw.onProgress(float64(m.pw.completed) / float64(m.pw.total))
//	}
//}
