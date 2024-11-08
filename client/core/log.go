package core

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var (
	LogLevel = logs.Warn
	Log      = &Logger{Logger: logs.NewLogger(LogLevel)}
	MuteLog  = &Logger{Logger: logs.NewLogger(logs.Important)}
)

var (
	NewLine                    = "\n"
	Debug           logs.Level = 10
	Warn            logs.Level = 20
	Info            logs.Level = 30
	Error           logs.Level = 40
	Important       logs.Level = 50
	GroupStyle                 = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
	NameStyle                  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6"))
	DefaultLogStyle            = map[logs.Level]string{
		Debug:     NewLine + termenv.String(tui.Rocket+"[+]").Bold().Background(tui.Blue).String() + " %s ",
		Warn:      NewLine + termenv.String(tui.Zap+"[warn]").Bold().Background(tui.Yellow).String() + " %s ",
		Important: NewLine + termenv.String(tui.Fire+"[*]").Bold().Background(tui.Purple).String() + " %s ",
		Info:      NewLine + termenv.String(tui.HotSpring+"[i]").Bold().Background(tui.Green).String() + " %s ",
		Error:     NewLine + termenv.String(tui.Monster+"[-]").Bold().Background(tui.Red).String() + " %s ",
	}
)

type Logger struct {
	*logs.Logger
}
