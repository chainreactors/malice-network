//go:build unix
// +build unix

package display

import (
	"github.com/reeflective/readline/internal/core"
)

// WatchResize redisplays the interface on terminal resize events.
func WatchResize(eng *Engine) chan<- bool {
	resizeChannel := core.GetTerminalResize(eng.keys)
	done := make(chan bool, 1)

	go func() {
		for {
			select {
			case <-resizeChannel:
				eng.completer.GenerateCached()
				eng.Refresh()
			case <-done:
				return
			}
		}
	}()

	return done
}
