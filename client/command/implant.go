package command

import (
	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/armory"
	"github.com/chainreactors/malice-network/client/command/exec"
	"github.com/chainreactors/malice-network/client/command/explorer"
	"github.com/chainreactors/malice-network/client/command/extension"
	"github.com/chainreactors/malice-network/client/command/file"
	"github.com/chainreactors/malice-network/client/command/filesystem"
	"github.com/chainreactors/malice-network/client/command/listener"
	"github.com/chainreactors/malice-network/client/command/login"
	"github.com/chainreactors/malice-network/client/command/modules"
	"github.com/chainreactors/malice-network/client/command/observe"
	"github.com/chainreactors/malice-network/client/command/sessions"
	"github.com/chainreactors/malice-network/client/command/sys"
	"github.com/chainreactors/malice-network/client/command/use"
	"github.com/chainreactors/malice-network/client/command/version"
	cc "github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
)

func BindImplantCommands(con *cc.Console) console.Commands {
	implantCommands := func() *cobra.Command {
		implant := &cobra.Command{
			Short: "implant commands",
			CompletionOptions: cobra.CompletionOptions{
				HiddenDefaultCmd: true,
			},
			GroupID: consts.ImplantGroup,
		}
		bind := makeBind(implant, con)

		bind("",
			version.Command)

		bind(consts.GenericGroup,
			login.Command,
			sessions.Commands,
			use.Command,
			//tasks.Command,
			alias.Commands,
			extension.Commands,
			armory.Commands,
			observe.Command,
			explorer.Commands,
		)

		bind(consts.ListenerGroup,
			listener.Commands,
		)

		bind(consts.ImplantGroup,
			exec.Commands,
			file.Commands,
			filesystem.Commands,
			sys.Commands,
			modules.Commands,
		)

		bind(consts.AliasesGroup)
		bind(consts.ExtensionGroup)
		return implant
	}
	return implantCommands
}
