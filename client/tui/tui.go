package tui

import tea "github.com/charmbracelet/bubbletea"

func Run(model tea.Model) error {
	p := tea.NewProgram(model)
	_, err := p.Run()
	return err
}

func AsyncRun(model tea.Model) *tea.Program {
	p := tea.NewProgram(model)
	go p.Run()
	return p
}
