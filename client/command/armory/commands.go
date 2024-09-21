package armory

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	armoryCmd := &cobra.Command{
		Use:   consts.CommandArmory,
		Short: "List available armory packages",
		// Long:  help.FormatLongHelp("armory"),
		Run: func(cmd *cobra.Command, args []string) {
			ArmoryCmd(cmd, con)
		},
	}
	common.Bind("connection", true, armoryCmd, func(f *pflag.FlagSet) {
		f.BoolP("insecure", "I", false, "skip tls certificate validation")
		f.StringP("proxy", "p", "", "specify a proxy url (e.g. http://localhost:8080)")
		f.BoolP("ignore-cache", "c", false, "ignore metadata cache, force refresh")
		f.StringP("timeout", "t", "", "download timeout")
	})
	common.Bind("type", false, armoryCmd, func(f *pflag.FlagSet) {
		f.BoolP("bundle", "", false, "install bundle")
	})

	armoryInstallCmd := &cobra.Command{
		Use:   consts.CommandArmoryInstall + " [armory]",
		Short: "Install a command armory",
		// Long:  help.FormatLongHelp(consts.CommandArmory + " " + consts.CommandAliasInstall),
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ArmoryInstallCmd(cmd, con)
		},
	}
	common.Bind("connection", false, armoryInstallCmd, func(f *pflag.FlagSet) {
		f.BoolP("force", "f", false, "force installation of package, overwriting the package if it exists")
		f.StringP("armory", "a", "", "name of armory to install package from")
	})
	common.Bind("name", true, armoryInstallCmd, func(f *pflag.FlagSet) {
		f.StringP("armory", "a", "Default", "name of the armory to install from")
	})

	armoryUpdateCmd := &cobra.Command{
		Use:   consts.CommandArmoryUpdate,
		Short: "Update installed armory packages",
		// Long:  help.FormatLongHelp(consts.CommandArmory + " " + consts.CommandArmoryUpdate),
		Run: func(cmd *cobra.Command, args []string) {
			ArmoryUpdateCmd(cmd, con)
		},
	}
	common.Bind("connection", false, armoryUpdateCmd, func(f *pflag.FlagSet) {
		f.StringP("armory", "a", "", "name of armory to install package from")
	})
	common.Bind("name", true, armoryUpdateCmd, func(f *pflag.FlagSet) {
		f.StringP("armory", "a", "Default", "name of the armory to install from")
	})

	armorySearchCmd := &cobra.Command{
		Use:   consts.CommandArmorySearch + " [armory]",
		Short: "Search for armory packages",
		// Long:  help.FormatLongHelp(consts.CommandArmory + " " + consts.CommandArmorySearch),
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ArmorySearchCmd(cmd, con)
		},
	}

	common.BindArgCompletions(armorySearchCmd, nil, carapace.ActionValues().Usage("a name regular expression"))

	carapace.Gen(armorySearchCmd).PositionalCompletion(carapace.ActionValues().Usage("a name regular expression"))

	// Adding subcommands to the main command
	armoryCmd.AddCommand(armoryInstallCmd)
	armoryCmd.AddCommand(armoryUpdateCmd)
	armoryCmd.AddCommand(armorySearchCmd)

	return []*cobra.Command{armoryCmd}
}
