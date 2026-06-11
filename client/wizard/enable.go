package wizard

import (
	"fmt"

	"github.com/spf13/cobra"
)

// AddWizardFlag adds the --wizard flag to a command
func AddWizardFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("wizard", false, "Start interactive wizard mode")
}

// WrapPreRunEWithWizard wraps a command's PreRunE to support wizard mode.
// Usage: cmd.PreRunE = wizard.WrapPreRunEWithWizard(originalPreRunE, originalPreRun)
func WrapPreRunEWithWizard(
	originalPreRunE func(cmd *cobra.Command, args []string) error,
	originalPreRun func(cmd *cobra.Command, args []string),
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if wizardMode, _ := cmd.Flags().GetBool("wizard"); wizardMode {
			if _, err := RunWizard(cmd); err != nil {
				return fmt.Errorf("wizard failed: %w", err)
			}
		}
		if originalPreRunE != nil {
			return originalPreRunE(cmd, args)
		}
		if originalPreRun != nil {
			originalPreRun(cmd, args)
		}
		return nil
	}
}

// WrapRunEWithWizard wraps a command's RunE to support wizard mode
// Usage: cmd.RunE = wizard.WrapRunEWithWizard(cmd, originalRunE)
func WrapRunEWithWizard(originalRunE func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if wizardMode, _ := cmd.Flags().GetBool("wizard"); wizardMode {
			if _, err := RunWizard(cmd); err != nil {
				return fmt.Errorf("wizard failed: %w", err)
			}
		}
		return originalRunE(cmd, args)
	}
}

// EnableWizard adds --wizard flag and wraps PreRunE for a command
// This is a convenience function that combines AddWizardFlag and WrapPreRunEWithWizard
func EnableWizard(cmd *cobra.Command) {
	if cmd.RunE == nil && cmd.Run == nil {
		return
	}
	AddWizardFlag(cmd)
	originalPreRunE := cmd.PreRunE
	originalPreRun := cmd.PreRun
	cmd.PreRunE = WrapPreRunEWithWizard(originalPreRunE, originalPreRun)
}

// EnableWizardForCommands enables wizard for multiple commands
func EnableWizardForCommands(cmds ...*cobra.Command) {
	for _, cmd := range cmds {
		EnableWizard(cmd)
	}
}
