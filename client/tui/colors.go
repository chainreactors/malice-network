package tui

import (
	"fmt"
	"github.com/muesli/termenv"
	"os"
)

var (
	output = termenv.NewOutput(os.Stdout)
	Blue   = termenv.ColorProfile().Color("#3398DA")
	Yellow = termenv.ColorProfile().Color("#F1C40F")
	Purple = termenv.ColorProfile().Color("#8D44AD")
	Green  = termenv.ColorProfile().Color("#2FCB71")
	Red    = termenv.ColorProfile().Color("#E74C3C")
) // You can use ANSI color codes directly

var (
	Reset = output.Reset
	Clear = output.ClearLine
	UpN   = output.CursorPrevLine
	Down  = output.CursorNextLine
)

var ClientPrompt = adaptTermColor()

// adaptTermColor - Adapt term color
// TODO: Adapt term color by term(fork grumble ColorTableFg)
func adaptTermColor() string {
	var color string
	if termenv.HasDarkBackground() {
		color = "\033[37mIOM> \033[0m"
	} else {
		color = "\033[30mIOM> \033[0m"
	}
	return color
}

func AdaptSessionColor(sId string) string {
	var sessionPrompt string
	if termenv.HasDarkBackground() {
		sessionPrompt = fmt.Sprintf("\033[37mIOM [%s]> \033[0m", sId[0:5])
	} else {
		sessionPrompt = fmt.Sprintf("\033[30mIOM [%s]> \033[0m", sId[0:5])
	}
	return sessionPrompt
}
