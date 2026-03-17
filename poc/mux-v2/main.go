// POC v2: Multiplexer demo using the external/tui/mux library.
//
// This demo spawns bash (or cmd.exe on Windows) subprocesses in PTY-backed
// terminal panes managed by the mux library.
//
// Build & run:
//
//	go build -o mux-v2.exe ./poc/mux-v2 && ./mux-v2.exe
//
// Keybindings:
//
//	Ctrl+B n     next tab
//	Ctrl+B p     previous tab
//	Ctrl+B c     new tab
//	Ctrl+B x     close focused pane
//	Ctrl+B "     split vertically
//	Ctrl+B %     split horizontally
//	Ctrl+B o     cycle focus between panes
//	Ctrl+B q     quit
package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/chainreactors/tui/mux"
)

func main() {
	m := mux.New(
		mux.WithSidebarWidth(22),
		mux.WithPaneFactory(shellFactory),
	)

	if err := m.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

var paneCounter int

func shellFactory(id int, width, height int) (*mux.TermPane, error) {
	paneCounter++
	name := fmt.Sprintf("shell-%d", paneCounter)

	shell, args := defaultShell()
	return mux.NewTermPane(id, name, shell, args, width, height)
}

func defaultShell() (string, []string) {
	if runtime.GOOS == "windows" {
		// Prefer bash under MSYS/Git Bash if available.
		if bash, err := exec.LookPath("bash"); err == nil {
			return bash, []string{"--login"}
		}
		return "cmd.exe", nil
	}

	// Unix: use SHELL env or fallback to /bin/sh.
	sh := os.Getenv("SHELL")
	if sh == "" {
		sh = "/bin/sh"
	}
	return sh, nil
}
