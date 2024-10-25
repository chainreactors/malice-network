package main

import (
	"fmt"
	"io"

	"github.com/reeflective/console"
)

const (
	shortUsage = "Console application example, with cobra commands/flags/completions generated from structs"
)

func main() {
	// Instantiate a new app, with a single, default menu.
	// All defaults are set, and nothing is needed to make it work.
	app := console.New("example")

	// Global Setup ------------------------------------------------- //
	app.NewlineBefore = true
	app.NewlineAfter = true

	app.SetPrintLogo(func(_ *console.Console) {
		fmt.Print(`
  _____            __ _           _   _              _____                      _
 |  __ \          / _| |         | | (_)            / ____|                    | |
 | |__) |___  ___| |_| | ___  ___| |_ ___   _____  | |     ___  _ __  ___  ___ | | ___
 |  _  // _ \/ _ \  _| |/ _ \/ __| __| \ \ / / _ \ | |    / _ \| '_ \/ __|/ _ \| |/ _ \
 | | \ \  __/  __/ | | |  __/ (__| |_| |\ V /  __/ | |___| (_) | | | \__ \ (_) | |  __/
 |_|  \_\___|\___|_| |_|\___|\___|\__|_| \_/ \___|  \_____\___/|_| |_|___/\___/|_|\___|

`)
	})

	// Main Menu Setup ---------------------------------------------- //

	// By default the shell as created a single menu and
	// made it current, so you can access it and set it up.
	menu := app.ActiveMenu()

	// Set some custom prompt handlers for this menu.
	setupPrompt(menu)

	// All menus currently each have a distinct, in-memory history source.
	// Replace the main (current) menu's history with one writing to our
	// application history file. The default history is named after its menu.
	hist, _ := embeddedHistory(".example-history")
	menu.AddHistorySource("local history", hist)

	// We bind a special handler for this menu, which will exit the
	// application (with confirm), when the shell readline receives
	// a Ctrl-D keystroke. You can map any error to any handler.
	menu.AddInterrupt(io.EOF, exitCtrlD)

	// Make a command yielder for our main menu.
	// menu.SetCommands(makeflagsCommands(app))
	// Thanks ChatGPT for generating this for us!
	menu.SetCommands(mainMenuCommands(app))

	// Client Menu Setup -------------------------------------------- //

	// Create another menu, different from the main one.
	// It will have its own command tree, prompt engine, history sources, etc.
	clientMenu := app.NewMenu("client")

	// Here, for the sake of demonstrating custom interrupt
	// handlers and for sparing use to write a dedicated command,
	// we use a custom interrupt handler to switch back to main menu.
	clientMenu.AddInterrupt(io.EOF, errorCtrlSwitchMenu)

	// Add some commands to our client menu.
	// This is an example of binding "traditionally defined" cobra.Commands.
	clientMenu.SetCommands(makeClientCommands(app))

	// Run the app -------------------------------------------------- //

	// Everything is ready for a tour.
	// Run the console and take a look around.
	app.Start()
}
