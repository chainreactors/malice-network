package wizard

import (
	"fmt"
	"sort"

	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/plugin"
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
		Use:   "run [wizard-name]",
		Short: "Run a wizard by name or from a spec file",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			specFile, _ := cmd.Flags().GetString("file")
			if specFile != "" {
				return RunWizardFileCmd(cmd, con, specFile)
			}
			if len(args) != 1 {
				return fmt.Errorf("wizard name is required (or use --file)")
			}
			return RunWizardCmd(cmd, con, args[0])
		},
		Example: `~~~
wizard run listener_setup
wizard run tcp_pipeline
wizard run profile_create
wizard run --file ./wizards/priv_esc.yaml
~~~`,
	}
	runCmd.Flags().StringP("file", "f", "", "run wizard from a JSON/YAML spec file (path or embed://...)")

	common.BindFlagCompletions(runCmd, func(comp carapace.ActionMap) {
		comp["file"] = carapace.ActionFiles().Usage("wizard spec file (JSON/YAML)")
	})
	common.BindArgCompletions(runCmd, nil, carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		_ = plugin.GetGlobalMalManager()

		templates := wizardfw.ListTemplates()
		results := make([]string, 0, len(templates)*2)
		for _, name := range templates {
			desc := ""
			if wiz, ok := wizardfw.GetTemplate(name); ok && wiz != nil {
				desc = wiz.Description
			}
			results = append(results, name, desc)
		}
		return carapace.ActionValuesDescribed(results...).Tag("wizard template")
	}))

	wizardCmd.AddCommand(listCmd, runCmd)

	// Add category commands (build, pipeline, cert, config)
	for _, cat := range wizardfw.Categories {
		catCmd := createCategoryCommand(con, cat)
		wizardCmd.AddCommand(catCmd)
	}

	// Add standalone wizard commands (listener, profile, infra)
	for _, sw := range wizardfw.StandaloneWizards {
		swCmd := createStandaloneCommand(con, sw)
		wizardCmd.AddCommand(swCmd)
	}

	return []*cobra.Command{wizardCmd}
}

// createCategoryCommand creates a command for a wizard category
func createCategoryCommand(con *core.Console, cat wizardfw.WizardCategory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   cat.Name + " [type]",
		Short: cat.Description,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = plugin.GetGlobalMalManager()

			var wizardID string

			if len(args) == 1 {
				// Direct type specified: wizard build beacon
				typeName := args[0]
				for _, w := range cat.Wizards {
					if w.ID == typeName {
						wizardID = w.FullID
						break
					}
				}
				if wizardID == "" {
					return fmt.Errorf("unknown %s type: %s", cat.Name, typeName)
				}
			} else {
				// No type specified: show interactive menu
				options := make([]wizardfw.SelectOption, len(cat.Wizards))
				for i, w := range cat.Wizards {
					options[i] = wizardfw.SelectOption{
						Value:       w.FullID,
						Label:       w.ID,
						Description: w.Description,
					}
				}

				selected, err := wizardfw.RunSelect(fmt.Sprintf("Select %s type", cat.Title), options)
				if err != nil {
					return err
				}
				wizardID = selected
			}

			wiz, ok := wizardfw.GetTemplate(wizardID)
			if !ok {
				return fmt.Errorf("wizard '%s' not found", wizardID)
			}

			return runWizard(con, wiz)
		},
	}

	if len(cat.Wizards) > 0 {
		results := make([]string, 0, len(cat.Wizards)*2)
		for _, w := range cat.Wizards {
			results = append(results, w.ID, w.Description)
		}
		common.BindArgCompletions(cmd, nil, carapace.ActionValuesDescribed(results...).Tag(cat.Title+" wizard"))
	}

	// Add valid types to help text
	var types []string
	for _, w := range cat.Wizards {
		types = append(types, w.ID)
	}
	if len(types) > 0 {
		cmd.Example = fmt.Sprintf("~~~\nwizard %s\nwizard %s %s\n~~~", cat.Name, cat.Name, types[0])
	} else {
		cmd.Example = fmt.Sprintf("~~~\nwizard %s\n~~~", cat.Name)
	}

	return cmd
}

// createStandaloneCommand creates a command for a standalone wizard
func createStandaloneCommand(con *core.Console, sw wizardfw.WizardEntry) *cobra.Command {
	return &cobra.Command{
		Use:   sw.ID,
		Short: sw.Description,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = plugin.GetGlobalMalManager()

			wiz, ok := wizardfw.GetTemplate(sw.FullID)
			if !ok {
				return fmt.Errorf("wizard '%s' not found", sw.FullID)
			}

			return runWizard(con, wiz)
		},
		Example: fmt.Sprintf("~~~\nwizard %s\n~~~", sw.ID),
	}
}

// ListWizardsCmd lists all available wizards
func ListWizardsCmd(cmd *cobra.Command, con *core.Console) error {
	// Ensure mal plugins are loaded so any resources/wizards specs are registered.
	_ = plugin.GetGlobalMalManager()

	con.Log.Infof("Available wizards:\n\n")

	// Show categories
	for _, cat := range wizardfw.Categories {
		con.Log.Infof("  %s (%s):\n", cat.Title, cat.Name)
		for _, w := range cat.Wizards {
			con.Log.Infof("    %-12s - %s\n", w.ID, w.Description)
		}
		con.Log.Infof("\n")
	}

	// Show standalone wizards
	con.Log.Infof("  Standalone:\n")
	for _, sw := range wizardfw.StandaloneWizards {
		con.Log.Infof("    %-12s - %s\n", sw.ID, sw.Description)
	}
	con.Log.Infof("\n")

	// Show plugin wizards (those not in categories)
	templates := wizardfw.ListTemplates()
	knownIDs := make(map[string]bool)
	for _, cat := range wizardfw.Categories {
		for _, w := range cat.Wizards {
			knownIDs[w.FullID] = true
		}
	}
	for _, sw := range wizardfw.StandaloneWizards {
		knownIDs[sw.FullID] = true
	}

	var pluginWizards []string
	for _, name := range templates {
		if !knownIDs[name] {
			pluginWizards = append(pluginWizards, name)
		}
	}

	if len(pluginWizards) > 0 {
		sort.Strings(pluginWizards)
		con.Log.Infof("  Plugin wizards:\n")
		for _, name := range pluginWizards {
			wiz, _ := wizardfw.GetTemplate(name)
			if wiz != nil {
				con.Log.Infof("    %-20s - %s\n", name, wiz.Description)
			} else {
				con.Log.Infof("    %s\n", name)
			}
		}
		con.Log.Infof("\n")
	}

	con.Log.Infof("Usage:\n")
	con.Log.Infof("  wizard <category>           - Select from category (e.g., wizard build)\n")
	con.Log.Infof("  wizard <category> <type>    - Run directly (e.g., wizard build beacon)\n")
	con.Log.Infof("  wizard <standalone>         - Run standalone wizard (e.g., wizard listener)\n")
	con.Log.Infof("  wizard run <full-name>      - Run by full name (e.g., wizard run build_beacon)\n")

	return nil
}

// RunWizardCmd runs a specific wizard
func RunWizardCmd(cmd *cobra.Command, con *core.Console, name string) error {
	// Ensure mal plugins are loaded so any resources/wizards specs are registered.
	_ = plugin.GetGlobalMalManager()

	wiz, ok := wizardfw.GetTemplate(name)
	if !ok {
		return fmt.Errorf("wizard '%s' not found. Use 'wizard list' to see available wizards", name)
	}

	return runWizard(con, wiz)
}

func RunWizardFileCmd(cmd *cobra.Command, con *core.Console, path string) error {
	wiz, err := wizardfw.NewWizardFromFile(path)
	if err != nil {
		return fmt.Errorf("failed to load wizard spec %q: %w", path, err)
	}

	return runWizard(con, wiz)
}

// setupDynamicProviders sets up OptionsProvider for known dynamic fields
func setupDynamicProviders(wiz *wizardfw.Wizard) {
	for _, f := range wiz.Fields {
		switch f.Name {
		case "profile":
			f.OptionsProvider = ProfileOptionsProvider()
		case "listener_id":
			f.OptionsProvider = ListenerOptionsProvider()
		case "pipeline", "pipeline_id":
			f.OptionsProvider = PipelineOptionsProvider()
		case "addresses", "address":
			f.OptionsProvider = AddressOptionsProvider()
		case "beacon_artifact_id":
			f.OptionsProvider = ArtifactOptionsProvider("beacon")
		}
	}
}

func runWizard(con *core.Console, wiz *wizardfw.Wizard) error {
	// Set dynamic options providers for known fields
	setupDynamicProviders(wiz)

	// Prepare dynamic options before running
	wiz.PrepareOptions(con)

	runner := wizardfw.NewRunner(wiz)
	result, err := runner.RunTwoPhase()
	if err != nil {
		return fmt.Errorf("wizard failed: %w", err)
	}

	con.Log.Infof("\nWizard completed successfully!\n")
	con.Log.Infof("Results:\n")
	values := result.ToMap()
	for _, f := range wiz.Fields {
		if v, ok := values[f.Name]; ok {
			con.Log.Infof("  %-20s: %v\n", f.Name, v)
		}
	}

	// Check if there's an executor for this wizard
	if executor, ok := GetExecutor(wiz.ID); ok {
		con.Log.Infof("\nExecuting wizard actions...\n")
		if err := executor(con, result); err != nil {
			return fmt.Errorf("wizard execution failed: %w", err)
		}
	} else {
		con.Log.Warnf("\nNo executor registered for wizard '%s'. Results are display-only.\n", wiz.ID)
	}

	return nil
}
