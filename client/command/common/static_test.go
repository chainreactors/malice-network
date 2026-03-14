package common

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestShouldUseStaticOutputWhenFlagSet(t *testing.T) {
	old := stdinIsTerminal
	stdinIsTerminal = func() bool { return true }
	t.Cleanup(func() { stdinIsTerminal = old })

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("static", false, "")
	if err := cmd.Flags().Set("static", "true"); err != nil {
		t.Fatalf("set static flag: %v", err)
	}

	if !ShouldUseStaticOutput(cmd) {
		t.Fatal("expected static output when --static is set")
	}
}

func TestShouldUseStaticOutputWhenStdinIsNotTerminal(t *testing.T) {
	old := stdinIsTerminal
	stdinIsTerminal = func() bool { return false }
	t.Cleanup(func() { stdinIsTerminal = old })

	cmd := &cobra.Command{Use: "test"}

	if !ShouldUseStaticOutput(cmd) {
		t.Fatal("expected static output when stdin is not a terminal")
	}
}

func TestShouldUseStaticOutputAllowsInteractiveTerminal(t *testing.T) {
	old := stdinIsTerminal
	stdinIsTerminal = func() bool { return true }
	t.Cleanup(func() { stdinIsTerminal = old })

	cmd := &cobra.Command{Use: "test"}

	if ShouldUseStaticOutput(cmd) {
		t.Fatal("did not expect static output for interactive terminal without --static")
	}
}
