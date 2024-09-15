package addon

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/helper"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	"math"
	"os"
	"path/filepath"
)

var loadedAddons = make(map[string]*loadedAddon)

type loadedAddon struct {
	Command *cobra.Command
	Func    *intermediate.InternalFunc
}

func LoadAddonCmd(cmd *cobra.Command, con *repl.Console) {
	path := cmd.Flags().Arg(0)
	module, _ := cmd.Flags().GetString("module")
	name, _ := cmd.Flags().GetString("name")
	//method, _ := cmd.Flags().GetString("method")
	if name == "" {
		name = filepath.Base(path)
	}
	if module == "" {
		module = helper.CheckExtModule(path)
	}

	session := con.GetInteractive()
	task, err := LoadAddon(con.Rpc, session, name, path, module)
	if err != nil {
		con.Log.Errorf("%s", err)
		return
	}

	session.Console(task, fmt.Sprintf("Load addon %s", name))
	con.AddCallback(task, func(msg *implantpb.Spite) {
		err = RegisterAddon(&implantpb.Addon{Name: name, Depend: module}, con)
		if err != nil {
			con.Log.Errorf("%s", err)
		}
	})
}

func LoadAddon(rpc clientrpc.MaliceRPCClient, sess *core.Session, name, path, depend string) (*clientpb.Task, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return rpc.LoadAddon(sess.Context(), &implantpb.LoadAddon{
		Name:   name,
		Depend: depend,
		Bin:    content,
	})
}

func RegisterAddon(addon *implantpb.Addon, con *repl.Console) error {
	cmd := con.ImplantMenu()
	addonCmd := &cobra.Command{
		Use:   addon.Name,
		Short: fmt.Sprintf("%s %s", addon.Depend, addon.Name),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteAddonCmd(cmd, con)
		},
		GroupID: consts.AddonGroup,
	}
	loadedAddons[addon.Name] = &loadedAddon{
		Command: addonCmd,
		Func: repl.WrapImplantFunc(con, func(rpc clientrpc.MaliceRPCClient, sess *core.Session, args string, sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
			cmdline, err := shellquote.Split(args)
			if err != nil {
				return nil, err
			}
			return ExecuteAddon(rpc, sess, addon.Name, cmdline, true, math.MaxUint32, sess.Os.Arch, "", sac)
		}, common.ParseAssembly),
	}
	cmd.AddCommand(addonCmd)
	return nil
}
