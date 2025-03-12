package core

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/lipgloss"
	"os"
	"regexp"
)

var (
	LogLevel = logs.WarnLevel
	Log      = &Logger{Logger: logs.NewLogger(LogLevel)}
	MuteLog  = &Logger{Logger: logs.NewLogger(logs.ImportantLevel + 1)}
)

var (
	NewLine                    = "\x1b[1E"
	Debug           logs.Level = 10
	Warn            logs.Level = 20
	Info            logs.Level = 30
	Error           logs.Level = 40
	Important       logs.Level = 50
	GroupStyle                 = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
	NameStyle                  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6"))
	DefaultLogStyle            = map[logs.Level]string{
		Debug:     NewLine + tui.BlueBg.Bold(true).Render(tui.Rocket+"[+]") + " %s",
		Warn:      NewLine + tui.YellowBg.Bold(true).Render(tui.Zap+"[warn]") + " %s",
		Important: NewLine + tui.PurpleBg.Bold(true).Render(tui.Fire+"[*]") + " %s",
		Info:      NewLine + tui.GreenBg.Bold(true).Render(tui.HotSpring+"[i]") + " %s",
		Error:     NewLine + tui.RedBg.Bold(true).Render(tui.Monster+"[-]") + " %s",
	}
)

type Logger struct {
	*logs.Logger
	logFile *os.File
}

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func (l *Logger) FileLog(s string) {
	if l.logFile != nil {
		l.logFile.WriteString(ansi.ReplaceAllString(s, ""))
		l.logFile.Sync()
	}
}
