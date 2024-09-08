package command

import (
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/addon"
	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/exec"
	"github.com/chainreactors/malice-network/client/command/explorer"
	"github.com/chainreactors/malice-network/client/command/extension"
	"github.com/chainreactors/malice-network/client/command/file"
	"github.com/chainreactors/malice-network/client/command/filesystem"
	"github.com/chainreactors/malice-network/client/command/mal"
	"github.com/chainreactors/malice-network/client/command/modules"
	"github.com/chainreactors/malice-network/client/command/sys"
	"github.com/chainreactors/malice-network/client/command/tasks"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func BindImplantCommands(con *repl.Console) console.Commands {
	implantCommands := func() *cobra.Command {
		implant := &cobra.Command{
			Short: "implant commands",
			CompletionOptions: cobra.CompletionOptions{
				HiddenDefaultCmd: true,
			},
			GroupID: consts.ImplantMenu,
		}
		common.Bind("common flag", true, implant, func(f *pflag.FlagSet) {
			f.IntP("timeout", "t", consts.DefaultTimeout, "command timeout in seconds")
		})
		bind := makeBind(implant, con)
		bindCommonCommands(bind)
		bind(consts.ImplantGroup,
			tasks.Command,
			exec.Commands,
			file.Commands,
			filesystem.Commands,
			sys.Commands,
			modules.Commands,
			explorer.Commands,
			addon.Commands,
		)

		bind(consts.ArmoryGroup)
		bind(consts.AddonGroup)
		bind(consts.MalGroup)
		// Load Aliases
		aliasManifests := assets.GetInstalledAliasManifests()
		for _, manifest := range aliasManifests {
			manifest, err := alias.LoadAlias(manifest, con)
			if err != nil {
				con.Log.Errorf("Failed to load alias: %s", err)
				continue
			}
			err = alias.RegisterAlias(manifest, implant, con)
			if err != nil {
				con.Log.Errorf("Failed to register alias: %s", err)
				continue
			}
		}

		// Load Extensions
		extensionManifests := assets.GetInstalledExtensionManifests()
		for _, manifest := range extensionManifests {
			mext, err := extension.LoadExtensionManifest(manifest)
			// Absorb error in case there's no extensions manifest
			if err != nil {
				//con doesn't appear to be initialised here?
				//con.PrintErrorf("Failed to load extension: %s", err)
				con.Log.Errorf("Failed to load extension: %s\n", err)
				continue
			}

			for _, ext := range mext.ExtCommand {
				extension.ExtensionRegisterCommand(ext, implant, con)
			}
		}

		for _, malName := range assets.GetInstalledMalManifests() {
			manifest, err := mal.LoadMalManiFest(con, malName)
			// Absorb error in case there's no extensions manifest
			if err != nil {
				//con doesn't appear to be initialised here?
				//con.PrintErrorf("Failed to load extension: %s", err)
				con.Log.Errorf("Failed to load mal: %s\n", err)
				continue
			}

			if plug, err := con.Plugins.LoadPlugin(manifest, con); err == nil {
				err := plug.ReverseRegisterLuaFunctions(implant)
				if err != nil {
					con.Log.Errorf("Failed to register mal command: %s\n", err)
					continue
				}
			} else {
				con.Log.Errorf("Failed to load mal: %s\n", err)
				continue
			}
		}
		return implant
	}
	return implantCommands
}
