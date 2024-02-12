package tui

import tea "github.com/charmbracelet/bubbletea"

func Run(model tea.Model) error {
	_, err := tea.NewProgram(model).Run()
	return err
}
