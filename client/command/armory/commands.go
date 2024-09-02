package armory

import (
	"github.com/chainreactors/malice-network/client/command/flags"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *console.Console) []*cobra.Command {
	armoryCmd := &cobra.Command{
		Use:   consts.CommandArmory,
		Short: "List available armory packages",
		Long:  help.GetHelpFor("armory"),
		Run: func(cmd *cobra.Command, args []string) {
			ArmoryCmd(cmd, con)
		},
	}
	flags.Bind("connection", true, armoryCmd, func(f *pflag.FlagSet) {
		f.BoolP("insecure", "I", false, "skip tls certificate validation")
		f.StringP("proxy", "p", "", "specify a proxy url (e.g. http://localhost:8080)")
		f.BoolP("ignore-cache", "c", false, "ignore metadata cache, force refresh")
		f.StringP("timeout", "t", "", "download timeout")
	})
	flags.Bind("type", false, armoryCmd, func(f *pflag.FlagSet) {
		f.BoolP("bundle", "r", false, "install bundle")
	})

	armoryInstallCmd := &cobra.Command{
		Use:   consts.CommandAliasInstall,
		Short: "Install a command armory",
		Long:  help.GetHelpFor(consts.CommandArmory + " " + consts.CommandAliasInstall),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ArmoryInstallCmd(cmd, con)
		},
	}
	flags.Bind("connection", false, armoryInstallCmd, func(f *pflag.FlagSet) {
		f.BoolP("force", "f", false, "force installation of package, overwriting the package if it exists")
		f.StringP("armory", "a", "", "name of armory to install package from")
	})
	flags.Bind("name", true, armoryInstallCmd, func(f *pflag.FlagSet) {
		f.StringP("armory", "a", "Default", "name of the armory to install from")
	})

	armoryUpdateCmd := &cobra.Command{
		Use:   consts.CommandArmoryUpdate,
		Short: "Update installed armory packages",
		Long:  help.GetHelpFor(consts.CommandArmory + " " + consts.CommandArmoryUpdate),
		Run: func(cmd *cobra.Command, args []string) {
			ArmoryUpdateCmd(cmd, con)
		},
	}
	flags.Bind("connection", false, armoryUpdateCmd, func(f *pflag.FlagSet) {
		f.StringP("armory", "a", "", "name of armory to install package from")
	})
	flags.Bind("name", true, armoryUpdateCmd, func(f *pflag.FlagSet) {
		f.StringP("armory", "a", "Default", "name of the armory to install from")
	})

	armorySearchCmd := &cobra.Command{
		Use:   consts.CommandArmorySearch + " [name]",
		Short: "Search for armory packages",
		Long:  help.GetHelpFor(consts.CommandArmory + " " + consts.CommandArmorySearch),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ArmorySearchCmd(cmd, con)
		},
	}
	carapace.Gen(armorySearchCmd).PositionalCompletion(carapace.ActionValues().Usage("a name regular expression"))

	// Adding subcommands to the main command
	armoryCmd.AddCommand(armoryInstallCmd)
	armoryCmd.AddCommand(armoryUpdateCmd)
	armoryCmd.AddCommand(armorySearchCmd)

	return []*cobra.Command{armoryCmd}
}
