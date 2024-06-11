package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func NewConfirm() *ConfirmModel {
	return &ConfirmModel{}
}

type ConfirmModel struct {
	confirmed bool
}

func (m *ConfirmModel) OK() bool {
	return m.confirmed
}

func (m *ConfirmModel) Init() tea.Cmd {
	return nil
}

func (m *ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y":
			m.confirmed = true
			return m, tea.Quit
		case "n":
			m.confirmed = false
			return m, tea.Quit
		case "q":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *ConfirmModel) View() string {
	if m.confirmed {
		return "Confirmed! Press any key to exit."
	}
	return "Press 'y' to confirm, 'q' to quit: "
}
