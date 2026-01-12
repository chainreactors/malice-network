package wizard

import (
	"fmt"
	"sort"

	"github.com/chainreactors/malice-network/client/core"
	wizardfw "github.com/chainreactors/malice-network/client/wizard"
	"github.com/spf13/cobra"
)

// Commands returns the wizard commands
func Commands(con *core.Console) []*cobra.Command {
	wizardCmd := &cobra.Command{
		Use:   "wizard",
		Short: "Interactive wizard system",
		Long:  "Run interactive wizards for configuration and setup",
	}

	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List available wizards",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListWizardsCmd(cmd, con)
		},
		Example: `~~~
wizard list
~~~`,
	}

	runCmd := &cobra.Command{
		Use:   "run <wizard-name>",
		Short: "Run a wizard by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunWizardCmd(cmd, con, args[0])
		},
		Example: `~~~
wizard run listener_setup
wizard run tcp_pipeline
wizard run profile_create
~~~`,
	}

	wizardCmd.AddCommand(listCmd, runCmd)

	return []*cobra.Command{wizardCmd}
}

// ListWizardsCmd lists all available wizards
func ListWizardsCmd(cmd *cobra.Command, con *core.Console) error {
	templates := wizardfw.ListTemplates()
	sort.Strings(templates)

	con.Log.Infof("Available wizards:\n")
	for _, name := range templates {
		wiz, _ := wizardfw.GetTemplate(name)
		if wiz != nil {
			con.Log.Infof("  %-20s - %s\n", name, wiz.Description)
		} else {
			con.Log.Infof("  %s\n", name)
		}
	}
	return nil
}

// RunWizardCmd runs a specific wizard
func RunWizardCmd(cmd *cobra.Command, con *core.Console, name string) error {
	wiz, ok := wizardfw.GetTemplate(name)
	if !ok {
		return fmt.Errorf("wizard '%s' not found. Use 'wizard list' to see available wizards", name)
	}

	runner := wizardfw.NewRunner(wiz)
	result, err := runner.Run()
	if err != nil {
		return fmt.Errorf("wizard failed: %w", err)
	}

	con.Log.Infof("\nWizard completed successfully!\n")
	con.Log.Infof("Results:\n")
	for k, v := range result.ToMap() {
		con.Log.Infof("  %-20s: %v\n", k, v)
	}

	return nil
}
