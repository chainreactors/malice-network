package armory

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
)

// Commands - The main armory command
func Commands(con *console.Console) []*grumble.Command {
	armoryCmd := &grumble.Command{
		Name: consts.CommandArmory,
		Help: "List available armory packages",
		Flags: func(f *grumble.Flags) {
			f.String("p", "proxy", "", "proxy URL")
			f.String("t", "timeout", "", "timeout")
			f.Bool("i", "insecure", false, "disable TLS validation")
			f.Bool("", "ignore-cache", false, "ignore cache")
		},
		Run: func(ctx *grumble.Context) error {
			ArmoryCmd(ctx, con)
			return nil
		},
		HelpGroup: consts.GenericGroup,
	}
	armoryCmd.AddCommand(&grumble.Command{
		Name: consts.CommandAliasInstall,
		Help: "Install a command armory",
		Args: func(a *grumble.Args) {
			a.String("name", "package or bundle name to install")
		},
		Flags: func(f *grumble.Flags) {
			f.String("a", "armory", "Default", "name of the armory to install from")
			f.Bool("f", "force", false,
				"force installation of package, overwriting the package if it exists")
			f.String("p", "proxy", "", "proxy URL")
		},
		Run: func(ctx *grumble.Context) error {
			ArmoryInstallCmd(ctx, con)
			return nil
		},
		HelpGroup: consts.GenericGroup,
	})
	armoryCmd.AddCommand(&grumble.Command{
		Name: consts.CommandArmoryUpdate,
		Help: "Update installed armory packages",
		Flags: func(f *grumble.Flags) {
			f.String("a", "armory", "", "name of the armory to update")
		},
		Run: func(ctx *grumble.Context) error {
			ArmoryUpdateCmd(ctx, con)
			return nil
		},
		HelpGroup: consts.GenericGroup,
	})
	armoryCmd.AddCommand(&grumble.Command{
		Name: consts.CommandArnorySearch,
		Help: "Search for armory packages",
		Args: func(a *grumble.Args) {
			a.String("name", "name of the package to search for")
		},
		Run: func(ctx *grumble.Context) error {
			ArmorySearchCmd(ctx, con)
			return nil
		},
		HelpGroup: consts.GenericGroup,
	})
	return []*grumble.Command{armoryCmd}
}
