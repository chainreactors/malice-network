package cli

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

type Model struct {
	Choices      []string
	SelectedItem int
	KeyHandler   KeyHandler
	NewKey       string
	IsQuit       bool
}

func (m *Model) Init() tea.Cmd {
	m.SelectedItem = -1
	return nil
}

func (m *Model) View() string {
	var view strings.Builder

	for i, choice := range m.Choices {
		if i == m.SelectedItem {
			view.WriteString("[x] ")
		} else {
			view.WriteString("[ ] ")
		}
		view.WriteString(choice)
		view.WriteRune('\n')
	}

	return view.String()
}

type KeyHandler func(*Model, tea.Msg) (tea.Model, tea.Cmd)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "up":
			m.SelectedItem--
			if m.SelectedItem < 0 {
				m.SelectedItem = len(m.Choices) - 1
			}
			return m, nil
		case "down":
			m.SelectedItem++
			if m.SelectedItem >= len(m.Choices) {
				m.SelectedItem = 0
			}
			return m, nil
		case "enter":
			if m.SelectedItem >= 0 && m.SelectedItem < len(m.Choices) {
			}
			return m, tea.Quit
		case m.NewKey:
			newModel, _ := m.KeyHandler(m, msg)
			if m.IsQuit {
				return newModel, tea.Quit
			}
			return newModel, nil
		}
	}

	return m, nil
}
