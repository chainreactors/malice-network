package server

import (
	"time"

	"github.com/chainreactors/logs"
)

const serverLogTimeFormat = "01.02 15:04:05"

// ConfigureDebugLogging configures readable server debug logs.
func ConfigureDebugLogging() {
	configureDebugLogger(logs.Log, time.Now)
}

func configureDebugLogger(logger *logs.Logger, now func() time.Time) {
	logger.SetLevel(logs.DebugLevel)
	logger.PrefixFunc = func() string {
		return "[" + now().Format(serverLogTimeFormat) + "] "
	}
	logger.SuffixFunc = func() string {
		return ""
	}

	logger.SetFormatter(map[logs.Level]string{
		logs.DebugLevel:     "{{prefix}}DBG %s\n",
		logs.WarnLevel:      "{{prefix}}WRN %s\n",
		logs.InfoLevel:      "{{prefix}}INF %s\n",
		logs.ErrorLevel:     "{{prefix}}ERR %s\n",
		logs.ImportantLevel: "{{prefix}}IMP %s\n",
	})
}
