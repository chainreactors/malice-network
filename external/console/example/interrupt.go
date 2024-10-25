package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/reeflective/console"
)

// exitCtrlD is a custom interrupt handler to use when the shell
// readline receives an io.EOF error, which is returned with CtrlD.
func exitCtrlD(c *console.Console) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Confirm exit (Y/y): ")
	text, _ := reader.ReadString('\n')
	answer := strings.TrimSpace(text)

	if (answer == "Y") || (answer == "y") {
		os.Exit(0)
	}
}

func switchMenu(c *console.Console) {
	fmt.Println("Switching to client menu")
	c.SwitchMenu("client")
}
