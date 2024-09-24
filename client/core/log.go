package core

import (
	"github.com/chainreactors/logs"
)

var (
	LogLevel = logs.Warn
	Log      = &Logger{Logger: logs.NewLogger(LogLevel)}
	MuteLog  = &Logger{Logger: logs.NewLogger(logs.Important)}
)

type Logger struct {
	*logs.Logger
}
