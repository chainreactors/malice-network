package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

func NewSelect(choices []string) *SelectModel {
	return &SelectModel{
		Choices: choices,
	}
}

type SelectModel struct {
	Choices      []string
	SelectedItem int
	KeyHandler   KeyHandler
	NewKey       string
	IsQuit       bool
}

func (m *SelectModel) Init() tea.Cmd {
	m.SelectedItem = -1
	return nil
}

func (m *SelectModel) View() string {
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

type KeyHandler func(*SelectModel, tea.Msg) (tea.Model, tea.Cmd)

func (m *SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
