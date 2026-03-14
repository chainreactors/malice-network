package common

import (
	"testing"

	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func TestShouldUseStaticOutputDefaultsToNonInteractiveOutsideREPL(t *testing.T) {
	old := StdinIsTerminal
	StdinIsTerminal = func() bool { return true }
	t.Cleanup(func() { StdinIsTerminal = old })

	con := &core.Console{}

	if !ShouldUseStaticOutput(con) {
		t.Fatal("expected static output outside the REPL")
	}
}

func TestShouldUseStaticOutputUsesREPLState(t *testing.T) {
	old := StdinIsTerminal
	StdinIsTerminal = func() bool { return true }
	t.Cleanup(func() { StdinIsTerminal = old })

	con := &core.Console{}
	restore := con.WithREPLExecution(true)
	t.Cleanup(restore)

	if ShouldUseStaticOutput(con) {
		t.Fatal("did not expect static output while running inside the REPL")
	}
}

func TestShouldUseStaticOutputWhenConsoleForcesNonInteractive(t *testing.T) {
	old := StdinIsTerminal
	StdinIsTerminal = func() bool { return true }
	t.Cleanup(func() { StdinIsTerminal = old })

	con := &core.Console{}
	restoreREPL := con.WithREPLExecution(true)
	t.Cleanup(restoreREPL)
	restoreForce := con.WithNonInteractiveExecution(true)
	t.Cleanup(restoreForce)

	if !ShouldUseStaticOutput(con) {
		t.Fatal("expected forced non-interactive execution to override REPL state")
	}
}

func TestShouldUseStaticOutputFallsBackToTerminalCheckWithoutConsole(t *testing.T) {
	old := StdinIsTerminal
	StdinIsTerminal = func() bool { return false }
	t.Cleanup(func() { StdinIsTerminal = old })

	if !ShouldUseStaticOutput(nil) {
		t.Fatal("expected static output when stdin is not a terminal")
	}
}

func TestConfirmFailsClosedWithoutYesInNonInteractiveMode(t *testing.T) {
	old := StdinIsTerminal
	StdinIsTerminal = func() bool { return false }
	t.Cleanup(func() { StdinIsTerminal = old })

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("yes", false, "")

	confirmed, err := Confirm(cmd, nil, "confirm?")
	if err == nil {
		t.Fatal("expected non-interactive confirmation to fail without --yes")
	}
	if confirmed {
		t.Fatal("did not expect confirmation to succeed")
	}
}

func TestConfirmAllowsYesInNonInteractiveMode(t *testing.T) {
	old := StdinIsTerminal
	StdinIsTerminal = func() bool { return false }
	t.Cleanup(func() { StdinIsTerminal = old })

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("yes", false, "")
	if err := cmd.Flags().Set("yes", "true"); err != nil {
		t.Fatalf("set yes flag: %v", err)
	}

	confirmed, err := Confirm(cmd, nil, "confirm?")
	if err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}
	if !confirmed {
		t.Fatal("expected --yes to skip confirmation")
	}
}
