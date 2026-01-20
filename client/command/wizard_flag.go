package command

import (
	"fmt"

	"github.com/chainreactors/malice-network/client/wizard"
	"github.com/spf13/cobra"
)

// WizardFlagName is the name of the global wizard flag
const WizardFlagName = "wizard"

// AddWizardFlag adds the --wizard flag to a command
func AddWizardFlag(cmd *cobra.Command) {
	cmd.Flags().Bool(WizardFlagName, false, "Start interactive wizard mode")
}

// ShouldRunWizard checks if the command should run in wizard mode
func ShouldRunWizard(cmd *cobra.Command) bool {
	wizardMode, _ := cmd.Flags().GetBool(WizardFlagName)
	return wizardMode
}

// RunWizardIfEnabled checks if wizard mode is enabled and runs it
// Returns true if wizard was run, false otherwise
func RunWizardIfEnabled(cmd *cobra.Command) (bool, error) {
	if !ShouldRunWizard(cmd) {
		return false, nil
	}

	// Run wizard - this handles everything including applying results to flags
	_, err := wizard.RunWizard(cmd)
	if err != nil {
		return true, fmt.Errorf("wizard failed: %w", err)
	}

	return true, nil
}
