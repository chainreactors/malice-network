package command

import (
	"fmt"

	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/wizard"
	"github.com/spf13/cobra"
)

// WizardFlagName is the name of the global wizard flag
const WizardFlagName = "wizard"

// RegisterWizardFlag registers the global --wizard flag on the root command
func RegisterWizardFlag(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().Bool(WizardFlagName, false, "Start interactive wizard mode")
}

// WrapWithWizardSupport wraps the PersistentPreRunE to add wizard support
// Returns new pre and post runner functions
func WrapWithWizardSupport(
	con *core.Console,
	originalPre, originalPost func(cmd *cobra.Command, args []string) error,
) (pre, post func(cmd *cobra.Command, args []string) error) {

	pre = func(cmd *cobra.Command, args []string) error {
		// Check if wizard mode is enabled
		wizardMode, _ := cmd.Flags().GetBool(WizardFlagName)
		if !wizardMode {
			// Not wizard mode, execute original logic
			if originalPre != nil {
				return originalPre(cmd, args)
			}
			return nil
		}

		// Wizard mode: convert command flags to wizard
		wiz := wizard.CobraToWizard(cmd)
		if wiz == nil {
			return fmt.Errorf("cannot create wizard for command %s", cmd.Name())
		}

		// Check if there are any fields to display
		if len(wiz.Fields) == 0 {
			cmd.Printf("Command %s has no configurable parameters\n", cmd.Name())
			// Continue with original PreRunE
			if originalPre != nil {
				return originalPre(cmd, args)
			}
			return nil
		}

		// Prepare dynamic options if console is available
		if con != nil {
			wiz.PrepareOptions(con)
		}

		// Run wizard
		runner := wizard.NewRunner(wiz)
		result, err := runner.Run()
		if err != nil {
			return fmt.Errorf("wizard cancelled or failed: %w", err)
		}

		// Apply wizard results to flags
		if err := wizard.ApplyWizardResultToFlags(cmd, result); err != nil {
			return fmt.Errorf("failed to apply wizard result: %w", err)
		}

		// Execute original PreRunE (if any)
		if originalPre != nil {
			return originalPre(cmd, args)
		}
		return nil
	}

	post = originalPost
	return pre, post
}

// ShouldRunWizard checks if the command should run in wizard mode
func ShouldRunWizard(cmd *cobra.Command) bool {
	wizardMode, _ := cmd.Flags().GetBool(WizardFlagName)
	return wizardMode
}

// HandleWizardFlag handles the --wizard flag for console mode
// This is called in PersistentPreRunE for interactive console commands
func HandleWizardFlag(cmd *cobra.Command, con *core.Console) error {
	// Check if wizard mode is enabled
	wizardMode, _ := cmd.Flags().GetBool(WizardFlagName)
	if !wizardMode {
		return nil
	}

	// Wizard mode: convert command flags to wizard
	wiz := wizard.CobraToWizard(cmd)
	if wiz == nil {
		return fmt.Errorf("cannot create wizard for command %s", cmd.Name())
	}

	// Check if there are any fields to display
	if len(wiz.Fields) == 0 {
		cmd.Printf("Command %s has no configurable parameters\n", cmd.Name())
		return nil
	}

	// Prepare dynamic options if console is available
	if con != nil {
		wiz.PrepareOptions(con)
	}

	// Run wizard
	runner := wizard.NewRunner(wiz)
	result, err := runner.Run()
	if err != nil {
		return fmt.Errorf("wizard cancelled or failed: %w", err)
	}

	// Apply wizard results to flags
	if err := wizard.ApplyWizardResultToFlags(cmd, result); err != nil {
		return fmt.Errorf("failed to apply wizard result: %w", err)
	}

	return nil
}

// RunWizardForCommand runs wizard for a specific command and applies results
// This can be used by subcommands that want to handle wizard mode themselves
func RunWizardForCommand(cmd *cobra.Command, con *core.Console) error {
	wiz := wizard.CobraToWizard(cmd)
	if wiz == nil {
		return fmt.Errorf("cannot create wizard for command %s", cmd.Name())
	}

	if len(wiz.Fields) == 0 {
		return nil // No fields to configure
	}

	if con != nil {
		wiz.PrepareOptions(con)
	}

	runner := wizard.NewRunner(wiz)
	result, err := runner.Run()
	if err != nil {
		return err
	}

	return wizard.ApplyWizardResultToFlags(cmd, result)
}
