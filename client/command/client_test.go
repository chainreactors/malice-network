package command

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/spf13/cobra"
)

func TestShouldStartConsoleRequiresTTYForLogin(t *testing.T) {
	old := common.StdinIsTerminal
	common.StdinIsTerminal = func() bool { return false }
	t.Cleanup(func() { common.StdinIsTerminal = old })

	cmd := &cobra.Command{Use: consts.CommandLogin}
	cmd.Flags().Bool("console", false, "")
	cmd.Flags().Bool("daemon", false, "")

	if shouldStartConsole(cmd) {
		t.Fatal("login should not start an interactive console in non-interactive mode")
	}
}

func TestShouldStartConsoleAllowsExplicitConsoleFlag(t *testing.T) {
	old := common.StdinIsTerminal
	common.StdinIsTerminal = func() bool { return false }
	t.Cleanup(func() { common.StdinIsTerminal = old })

	cmd := &cobra.Command{Use: "version"}
	cmd.Flags().Bool("console", false, "")
	cmd.Flags().Bool("daemon", false, "")
	if err := cmd.Flags().Set("console", "true"); err != nil {
		t.Fatalf("failed to set console flag: %v", err)
	}

	if !shouldStartConsole(cmd) {
		t.Fatal("--console should force console startup")
	}
}

func TestShouldStartConsoleDoesNotStartRootInNonInteractiveMode(t *testing.T) {
	old := common.StdinIsTerminal
	common.StdinIsTerminal = func() bool { return false }
	t.Cleanup(func() { common.StdinIsTerminal = old })

	cmd := &cobra.Command{Use: consts.ClientMenu}
	cmd.Flags().Bool("console", false, "")
	cmd.Flags().Bool("daemon", false, "")

	if shouldStartConsole(cmd) {
		t.Fatal("root command should not start the console in non-interactive mode")
	}
}

func TestShouldStartDaemonAllowsHeadlessRuntime(t *testing.T) {
	old := common.StdinIsTerminal
	common.StdinIsTerminal = func() bool { return false }
	t.Cleanup(func() { common.StdinIsTerminal = old })

	cmd := &cobra.Command{Use: consts.CommandLogin}
	cmd.Flags().Bool("console", false, "")
	cmd.Flags().Bool("daemon", false, "")
	if err := cmd.Flags().Set("daemon", "true"); err != nil {
		t.Fatalf("failed to set daemon flag: %v", err)
	}

	if !common.ShouldStartDaemon(cmd) {
		t.Fatal("--daemon should enable headless runtime mode")
	}
	if !common.ShouldStartRuntime(cmd) {
		t.Fatal("--daemon should keep runtime alive even without REPL")
	}
	if common.ShouldSuppressStartupOutput(cmd) {
		t.Fatal("--daemon should preserve startup output for diagnostics")
	}
}

func TestValidateExecutionModeFlagsRejectsConsoleAndDaemon(t *testing.T) {
	cmd := &cobra.Command{Use: consts.CommandLogin}
	cmd.Flags().Bool("console", false, "")
	cmd.Flags().Bool("daemon", false, "")
	if err := cmd.Flags().Set("console", "true"); err != nil {
		t.Fatalf("failed to set console flag: %v", err)
	}
	if err := cmd.Flags().Set("daemon", "true"); err != nil {
		t.Fatalf("failed to set daemon flag: %v", err)
	}

	if err := common.ValidateExecutionModeFlags(cmd); err == nil {
		t.Fatal("expected --console and --daemon to conflict")
	}
}
