package tui

import tea "github.com/charmbracelet/bubbletea"

func Run(model tea.Model) error {
<<<<<<< HEAD
<<<<<<< HEAD
	p := tea.NewProgram(model)
	_, err := p.Run()
	return err
}

func AsyncRun(model tea.Model) *tea.Program {
	p := tea.NewProgram(model)
	go p.Run()
	return p
}
=======
	_, err := tea.NewProgram(model).Run()
	return err
}
>>>>>>> 9b152bf (refactor tui)
=======
	p := tea.NewProgram(model)
	_, err := p.Run()
	return err
}

func AsyncRun(model tea.Model) *tea.Program {
	p := tea.NewProgram(model)
	go p.Run()
	return p
}
>>>>>>> c1668d4 (refactor tui.bar)
