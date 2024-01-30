package styles

import "github.com/chainreactors/logs"

type LogStyle struct {
	Debug     string
	Warn      string
	Info      string
	Error     string
	Important string
}

// DefaultLogFormatter Default log style
func DefaultLogFormatter(logger *logs.Logger) {
	logStyle := map[logs.Level]string{
		logs.Debug:     DefaultLogStyle.Debug,
		logs.Warn:      DefaultLogStyle.Warn,
		logs.Important: DefaultLogStyle.Important,
		logs.Info:      DefaultLogStyle.Info,
		logs.Error:     DefaultLogStyle.Error,
	}
	logger.SetFormatter(logStyle)
}

// LogFormatter logs formatter
func LogFormatter(logger logs.Logger, style LogStyle) {
	logStyle := map[logs.Level]string{
		logs.Debug:     style.Debug,
		logs.Warn:      style.Warn,
		logs.Important: style.Important,
		logs.Info:      style.Info,
		logs.Error:     style.Error,
	}
	logger.SetFormatter(logStyle)
}
