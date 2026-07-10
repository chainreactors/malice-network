package mal

import (
	"fmt"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/plugin"
	"github.com/spf13/cobra"
)

func registerMalPlugin(con *core.Console, root *cobra.Command, plug plugin.Plugin) error {
	if con == nil || root == nil || plug == nil || plug.Manifest() == nil || plug.Manifest().Name == "" {
		return fmt.Errorf("invalid mal plugin registration")
	}

	for _, command := range plug.Commands() {
		if command == nil || command.Command == nil {
			continue
		}
		common.RegisterCommand(root, con, plug.Manifest().Name, "mal", command.Command)
	}
	common.RegisterPluginEventHooks(con, plug)
	refreshMalSearchIndex(con)
	return nil
}

func unregisterMalPlugin(con *core.Console, root *cobra.Command, plug plugin.Plugin) {
	if con == nil || root == nil || plug == nil {
		return
	}
	common.UnregisterPluginEventHooks(con, plug)
	for _, command := range plug.Commands() {
		if command == nil || command.Command == nil {
			continue
		}
		common.UnregisterCommand(root, con, command.Command)
	}
	refreshMalSearchIndex(con)
}

func refreshMalSearchIndex(con *core.Console) {
	if con.Server != nil && con.Server.ServerState != nil && con.ActiveTarget != nil {
		if session := con.ActiveTarget.Get(); session != nil {
			con.RefreshCmd(session)
		}
	}
	if con.SearchIndex == nil || con.App == nil {
		return
	}
	clientMenu := con.App.Menu(consts.ClientMenu)
	implantMenu := con.App.Menu(consts.ImplantMenu)
	if err := con.SearchIndex.Rebuild(clientMenu.Commands, implantMenu.Commands); err != nil {
		con.Log.Warnf("search index rebuild after mal change failed: %v\n", err)
	}
}
