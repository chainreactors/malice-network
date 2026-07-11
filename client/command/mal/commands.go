package mal

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *core.Console) []*cobra.Command {
	cmd := &cobra.Command{
		Use:   consts.CommandMal,
		Short: "mal commands",
		Long:  "Manage MAL plugin manifests installed in the client.",
		Example: `~~~
mal list
mal install ./example.yaml
mal load example
~~~`,
		Annotations: map[string]string{
			"thirdParty": "true",
			"static":     "true",
		},
		//Long:  help.GetHelpFor(consts.CommandExtension),
		RunE: func(cmd *cobra.Command, args []string) error {
			return MalCmd(cmd, con)
		},
	}

	installCmd := &cobra.Command{
		Use:   consts.CommandMalInstall + " [mal_file]",
		Short: "Install a mal manifest",
		Long:  "Install a MAL manifest from a local file or configured remote source.",
		Example: `~~~
mal install ./example.yaml
mal install ./example.yaml --version latest
~~~`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return MalInstallCmd(cmd, con)
		},
	}
	common.BindFlag(installCmd, common.MalHttpFlagset, func(f *pflag.FlagSet) {
		f.String("version", "latest", "mal version to install")
	})

	common.BindArgCompletions(installCmd,
		nil,
		carapace.ActionFiles().Usage("path the mal file to load"))

	cmd.AddCommand(installCmd)

	loadCmd := &cobra.Command{
		Use:   consts.CommandMalLoad + " [mal]",
		Short: "Load a mal manifest",
		Long:  "Load an installed MAL manifest and register its commands.",
		Example: `~~~
mal load example
~~~`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return MalLoadCmd(cmd, con)
		},
	}

	common.BindArgCompletions(loadCmd, nil, common.ExternalMalFileCompleter(con))

	cmd.AddCommand(loadCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   consts.CommandMalList,
		Short: "List mal manifests",
		Long:  "List embedded and externally installed MAL manifests.",
		Example: `~~~
mal list
~~~`,
		Annotations: map[string]string{
			"static": "true",
		},
		Run: func(cmd *cobra.Command, args []string) {
			ListMalManifest(cmd, con)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   consts.CommandMalRemove + " [mal]",
		Short: "Remove a mal manifest",
		Long:  "Remove an externally installed MAL manifest.",
		Example: `~~~
mal remove example
~~~`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RemoveMalCmd(cmd, con)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   consts.CommandMalRefresh,
		Short: "Refresh mal manifests",
		Long:  "Reload installed MAL manifests from local storage.",
		Example: `~~~
mal refresh
~~~`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RefreshMalCmd(cmd, con)
		},
	})

	updateCmd := &cobra.Command{
		Use:   consts.CommandMalUpdate + " [mal]",
		Short: "Update a mal or all mals",
		Long:  "Update one MAL manifest by name, or update every installed manifest with --all.",
		Example: `~~~
mal update example
mal update --all
~~~`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return UpdateMalCmd(cmd, con)
		},
	}

	common.BindFlag(updateCmd, common.MalHttpFlagset, func(f *pflag.FlagSet) {
		f.BoolP("all", "a", false, "update all mal")
	})

	cmd.AddCommand(updateCmd)
	return []*cobra.Command{cmd}
}
