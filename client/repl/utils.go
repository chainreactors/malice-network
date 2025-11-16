package repl

import (
	"fmt"
	"github.com/chainreactors/IoM-go/client"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

func CmdExist(cmd *cobra.Command, name string) bool {
	for _, c := range cmd.Commands() {
		if name == c.Name() {
			return true
		}
	}
	return false
}

func GetCmd(cmd *cobra.Command, name string) *cobra.Command {
	for _, c := range cmd.Commands() {
		if name == c.Name() {
			return c
		}
	}
	return nil

}

func NewSessionColor(prePrompt, sId string) string {
	var sessionPrompt string
	runes := []rune(sId)
	if termenv.HasDarkBackground() {
		sessionPrompt = fmt.Sprintf("%s [%s]> ", client.GroupStyle.Render(prePrompt), client.NameStyle.Render(string(runes)))
	} else {
		sessionPrompt = fmt.Sprintf("%s [%s]> ", client.GroupStyle.Render(prePrompt), client.NameStyle.Render(string(runes)))
	}
	return sessionPrompt
}

// From the x/exp source code - gets a slice of keys for a map
func Keys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}

	return r
}
