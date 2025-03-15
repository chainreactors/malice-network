package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Exit returns a command to exit the console application.
// The command will prompt the user to confirm quitting.
func Exit() *cobra.Command {
	exitCmd := &cobra.Command{
		Use:     "exit",
		Short:   "Exit the console application",
		GroupID: "core",
		Run: func(_ *cobra.Command, _ []string) {
			exitCtrlD()
		},
	}

	return exitCmd
}

// exitCtrlD is a custom interrupt handler to use when the shell
// readline receives an io.EOF error, which is returned with CtrlD.
func exitCtrlD() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Confirm exit (Y/y): ")

	text, _ := reader.ReadString('\n')
	answer := strings.TrimSpace(text)

	if strings.EqualFold(answer, "y") {
		os.Exit(0)
	}
}
