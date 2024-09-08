package addon

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/helper"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
)

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
		repl.Log.Errorf("%s", err)
		return
	}

	con.AddCallback(task, func(msg proto.Message) {
		session.Log.Infof("addon %s loaded", name)
		err = RegisterAddon(&implantpb.Addon{Name: name, Depend: module}, con, con.ImplantMenu())
		if err != nil {
			session.Log.Errorf("%s", err)
			return
		}
		con.UpdateSession(session.SessionId)
	})
}

func LoadAddon(rpc clientrpc.MaliceRPCClient, sess *repl.Session, name, path, depend string) (*clientpb.Task, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return rpc.LoadAddon(repl.Context(sess), &implantpb.LoadAddon{
		Name:   name,
		Depend: depend,
		Bin:    content,
	})
}

func RegisterAddon(addon *implantpb.Addon, con *repl.Console, cmd *cobra.Command) error {
	addonCmd := &cobra.Command{
		Use:   addon.Name,
		Short: fmt.Sprintf("%s %s", addon.Depend, addon.Name),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteAddonCmd(cmd, con)
		},
		GroupID: consts.AddonGroup,
	}

	cmd.AddCommand(addonCmd)
	return nil
}
