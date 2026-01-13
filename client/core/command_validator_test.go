package core

import (
	"sort"
	"testing"

	"github.com/spf13/cobra"
)

func TestGetCommandsForMenu(t *testing.T) {
	// Create mock client commands
	clientRoot := &cobra.Command{Use: "client"}
	clientRoot.AddCommand(
		&cobra.Command{Use: "wizard", Short: "wizard command"},
		&cobra.Command{Use: "website", Short: "website command"},
		&cobra.Command{Use: "listener", Short: "listener command"},
	)

	// Create mock implant commands
	implantRoot := &cobra.Command{Use: "implant"}
	implantRoot.AddCommand(
		&cobra.Command{Use: "whoami", Short: "whoami command"},
		&cobra.Command{Use: "wmi_query", Short: "wmi query command"},
		&cobra.Command{Use: "ps", Short: "process list"},
	)

	// Create validator with client menu
	v := NewCommandValidatorWithMenu(clientRoot, "client")
	// Add implant commands
	v.AddCommandsFromCobra(implantRoot, "implant")

	// Test: Get client commands
	clientCmds := v.GetCommandsForMenu("client")
	sort.Strings(clientCmds)

	expectedClient := []string{"client", "listener", "website", "wizard"}
	sort.Strings(expectedClient)

	if len(clientCmds) != len(expectedClient) {
		t.Errorf("Expected %d client commands, got %d: %v", len(expectedClient), len(clientCmds), clientCmds)
	}

	for i, cmd := range expectedClient {
		if clientCmds[i] != cmd {
			t.Errorf("Expected client command %q, got %q", cmd, clientCmds[i])
		}
	}

	// Test: Get implant commands
	implantCmds := v.GetCommandsForMenu("implant")
	sort.Strings(implantCmds)

	expectedImplant := []string{"implant", "ps", "whoami", "wmi_query"}
	sort.Strings(expectedImplant)

	if len(implantCmds) != len(expectedImplant) {
		t.Errorf("Expected %d implant commands, got %d: %v", len(expectedImplant), len(implantCmds), implantCmds)
	}

	for i, cmd := range expectedImplant {
		if implantCmds[i] != cmd {
			t.Errorf("Expected implant command %q, got %q", cmd, implantCmds[i])
		}
	}

	// Test: Verify whoami is NOT in client menu
	for _, cmd := range clientCmds {
		if cmd == "whoami" {
			t.Error("whoami should NOT be in client menu commands")
		}
	}

	// Test: Verify wizard is NOT in implant menu
	for _, cmd := range implantCmds {
		if cmd == "wizard" {
			t.Error("wizard should NOT be in implant menu commands")
		}
	}

	// Test: Empty menu returns all commands
	allCmds := v.GetCommandsForMenu("")
	if len(allCmds) != len(expectedClient)+len(expectedImplant) {
		t.Errorf("Expected %d total commands, got %d", len(expectedClient)+len(expectedImplant), len(allCmds))
	}
}

func TestAddCommandWithMenu(t *testing.T) {
	v := NewCommandValidator(nil)

	// Add commands with menu context
	v.AddCommandWithMenu("client", "sessions", "ss")
	v.AddCommandWithMenu("implant", "shell", "sh")

	// Verify client command
	clientCmds := v.GetCommandsForMenu("client")
	found := false
	for _, cmd := range clientCmds {
		if cmd == "sessions" {
			found = true
			break
		}
	}
	if !found {
		t.Error("sessions should be in client menu")
	}

	// Verify implant command
	implantCmds := v.GetCommandsForMenu("implant")
	found = false
	for _, cmd := range implantCmds {
		if cmd == "shell" {
			found = true
			break
		}
	}
	if !found {
		t.Error("shell should be in implant menu")
	}

	// Verify shell is NOT in client menu
	for _, cmd := range clientCmds {
		if cmd == "shell" {
			t.Error("shell should NOT be in client menu")
		}
	}
}
