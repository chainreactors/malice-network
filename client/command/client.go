package command

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/armory"
	"github.com/chainreactors/malice-network/client/command/explorer"
	"github.com/chainreactors/malice-network/client/command/extension"
	"github.com/chainreactors/malice-network/client/command/jobs"
	"github.com/chainreactors/malice-network/client/command/listener"
	"github.com/chainreactors/malice-network/client/command/login"
	"github.com/chainreactors/malice-network/client/command/observe"
	"github.com/chainreactors/malice-network/client/command/sessions"
	"github.com/chainreactors/malice-network/client/command/tasks"
	"github.com/chainreactors/malice-network/client/command/use"
	"github.com/chainreactors/malice-network/client/command/version"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/mal"

	"github.com/chainreactors/malice-network/helper/consts"
)

func BindClientsCommands(con *console.Console) {
	bind := makeBind(con)

	bind("",
		version.Command)

	bind(consts.GenericGroup,
		login.Command,
		sessions.Commands,
		use.Command,
		tasks.Command,
		jobs.Command,
		alias.Commands,
		extension.Commands,
		armory.Commands,
		observe.Command,
		explorer.Commands,
		mal.Commands,
	)

	bind(consts.ListenerGroup,
		listener.Commands,
	)

	bind(consts.AliasesGroup)

	// [ Extensions ]
	bind(consts.ExtensionGroup)

	// Load Aliases
	aliasManifests := assets.GetInstalledAliasManifests()
	for _, manifest := range aliasManifests {
		_, err := alias.LoadAlias(manifest, con)
		if err != nil {
			console.Log.Errorf("Failed to load alias: %s", err)
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
			console.Log.Errorf("Failed to load extension: %s\n", err)
			continue
		}

		for _, ext := range mext.ExtCommand {
			extension.ExtensionRegisterCommand(ext, con)
		}
	}

	if con.ServerStatus == nil {
		login.LoginCmd(&grumble.Context{}, con)
	}
}
