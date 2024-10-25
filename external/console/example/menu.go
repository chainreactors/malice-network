package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/reeflective/console"
)

// In here we create some menus which hold different command trees.
func createMenus(c *console.Console) {
	clientMenu := c.NewMenu("client")

	// Here, for the sake of demonstrating custom interrupt
	// handlers and for sparing use to write a dedicated command,
	// we use a custom interrupt handler to switch back to main menu.
	clientMenu.AddInterrupt(io.EOF, errorCtrlSwitchMenu)

	// Add some commands to our client menu.
	// This is an example of binding "traditionally defined" cobra.Commands.
	clientMenu.SetCommands(makeClientCommands(c))
}

// errorCtrlSwitchMenu is a custom interrupt handler which will
// switch back to the main menu when the current menu receives
// a CtrlD (io.EOF) error.
func errorCtrlSwitchMenu(c *console.Console) {
	fmt.Println("Switching back to main menu")
	c.SwitchMenu("")
}

// A little set of commands for the client menu, (wrapped so that
// we can pass the console to them, because the console is local).
func makeClientCommands(app *console.Console) console.Commands {
	return func() *cobra.Command {
		root := &cobra.Command{}

		ticker := &cobra.Command{
			Use:   "ticker",
			Short: "Triggers some asynchronous notifications to the shell, demonstrating async logging",
			Run: func(cmd *cobra.Command, args []string) {
				menu := app.ActiveMenu()
				timer := time.Tick(2 * time.Second)
				messages := []string{
					"Info:    notification 1",
					"Info:    notification 2",
					"Warning: notification 3",
					"Info:    notification 4",
					"Error:   done notifying",
				}
				go func() {
					count := 0
					for {
						<-timer
						if count == 5 {
							app.Printf("This message is more important, printing it below the prompt and in every menu")
							return
						}
						menu.TransientPrintf(messages[count])
						count++
					}
				}()
			},
		}
		root.AddCommand(ticker)

		main := &cobra.Command{
			Use:   "main",
			Short: "A command to return to the main menu (you can also use CtrlD for the same result)",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("Switching back to main menu")
				app.SwitchMenu("")
			},
		}
		root.AddCommand(main)

		shell := &cobra.Command{
			Use:                "!",
			Short:              "Execute the remaining arguments with system shell",
			DisableFlagParsing: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				if len(args) == 0 {
					return errors.New("command requires one or more arguments")
				}

				path, err := exec.LookPath(args[0])
				if err != nil {
					return err
				}

				shellCmd := exec.Command(path, args[1:]...)

				// Load OS environment
				shellCmd.Env = os.Environ()

				out, err := shellCmd.CombinedOutput()
				if err != nil {
					return err
				}

				fmt.Print(string(out))

				return nil
			},
		}
		root.AddCommand(shell)

		interruptible := &cobra.Command{
			Use:                "interrupt",
			Short:              "A command which prints a few status messages, but can be interrupted with CtrlC",
			DisableFlagParsing: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				menu := app.ActiveMenu()
				timer := time.Tick(2 * time.Second)
				messages := []string{
					"Info:    notification 1",
					"Info:    notification 2",
					"Warning: notification 3",
					"Info:    notification 4",
					"Error:   done notifying",
				}
				count := 0
				for {
					select {
					case <-menu.Context().Done():
						menu.TransientPrintf("Interrupted")
						return nil
					case <-timer:
						if count == 5 {
							return nil
						}
						menu.TransientPrintf(messages[count] + "\n")
						count++
					}
				}
			},
		}
		root.AddCommand(interruptible)

		return root
	}
}

// setupPrompt is a function which sets up the prompts for the main menu.
func setupPrompt(m *console.Menu) {
	p := m.Prompt()

	p.Primary = func() string {
		prompt := "\x1b[33mexample\x1b[0m [main] in \x1b[34m%s\x1b[0m\n> "
		wd, _ := os.Getwd()

		dir, err := filepath.Rel(os.Getenv("HOME"), wd)
		if err != nil {
			dir = filepath.Base(wd)
		}

		return fmt.Sprintf(prompt, dir)
	}

	p.Secondary = func() string { return ">" }
	p.Right = func() string {
		return "\x1b[1;30m" + time.Now().Format("03:04:05.000") + "\x1b[0m"
	}

	p.Transient = func() string { return "\x1b[1;30m" + ">> " + "\x1b[0m" }
}
