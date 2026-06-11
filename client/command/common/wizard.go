package common

import (
	"github.com/chainreactors/malice-network/client/wizard"
	"github.com/spf13/cobra"
)

// AddWizardFlag adds the --wizard flag to a command
// Deprecated: Use wizard.AddWizardFlag instead
func AddWizardFlag(cmd *cobra.Command) {
	wizard.AddWizardFlag(cmd)
}

// WrapPreRunEWithWizard wraps a command's PreRunE to support wizard mode.
// Deprecated: Use wizard.WrapPreRunEWithWizard instead
func WrapPreRunEWithWizard(
	originalPreRunE func(cmd *cobra.Command, args []string) error,
	originalPreRun func(cmd *cobra.Command, args []string),
) func(cmd *cobra.Command, args []string) error {
	return wizard.WrapPreRunEWithWizard(originalPreRunE, originalPreRun)
}

// WrapRunEWithWizard wraps a command's RunE to support wizard mode
// Deprecated: Use wizard.WrapRunEWithWizard instead
func WrapRunEWithWizard(originalRunE func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return wizard.WrapRunEWithWizard(originalRunE)
}

// EnableWizard adds --wizard flag and wraps PreRunE for a command
// Deprecated: Use wizard.EnableWizard instead
func EnableWizard(cmd *cobra.Command) {
	wizard.EnableWizard(cmd)
}

// EnableWizardForCommands enables wizard for multiple commands
// Deprecated: Use wizard.EnableWizardForCommands instead
func EnableWizardForCommands(cmds ...*cobra.Command) {
	wizard.EnableWizardForCommands(cmds...)
}
