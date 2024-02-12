package tui

import (
	"github.com/chainreactors/logs"
<<<<<<< HEAD
<<<<<<< HEAD
=======
	"github.com/chainreactors/malice-network/helper/consts"
>>>>>>> 9b152bf (refactor tui)
=======
>>>>>>> c1668d4 (refactor tui.bar)
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// base styles
var (
	HeaderStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			BorderBottom(true).
			Bold(false)
	FootStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
	SelectStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(false)
	HelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render
)

// Default Styles
var (
	DefaultLogStyle = map[logs.Level]string{
<<<<<<< HEAD
<<<<<<< HEAD
=======
>>>>>>> c1668d4 (refactor tui.bar)
		logs.Debug:     termenv.String(Rocket+"[+]").Bold().Background(Blue).String() + " %s ",
		logs.Warn:      termenv.String(Zap+"[warn]").Bold().Background(Yellow).String() + " %s ",
		logs.Important: termenv.String(Fire+"[*]").Bold().Background(Purple).String() + " %s ",
		logs.Info:      termenv.String(HotSpring+"[i]").Bold().Background(Green).String() + " %s ",
		logs.Error:     termenv.String(Monster+"[-]").Bold().Background(Red).String() + " %s ",
<<<<<<< HEAD
=======
		logs.Debug:     termenv.String(consts.Rocket+"[+]").Bold().Background(Blue).String() + " %s ",
		logs.Warn:      termenv.String(consts.Zap+"[warn]").Bold().Background(Yellow).String() + " %s ",
		logs.Important: termenv.String(consts.Fire+"[*]").Bold().Background(Purple).String() + " %s ",
		logs.Info:      termenv.String(consts.HotSpring+"[i]").Bold().Background(Green).String() + " %s ",
		logs.Error:     termenv.String(consts.Monster+"[-]").Bold().Background(Red).String() + " %s ",
>>>>>>> 9b152bf (refactor tui)
=======
>>>>>>> c1668d4 (refactor tui.bar)
	}

	DefaultTableStyle = &table.Styles{
		Selected: SelectStyle,
		Header:   HeaderStyle,
		Cell:     lipgloss.NewStyle().Padding(0, 1),
	}
)
