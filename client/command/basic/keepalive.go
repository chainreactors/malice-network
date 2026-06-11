package basic

import (
	"fmt"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
	"strings"
)

func KeepaliveCmd(cmd *cobra.Command, con *core.Console) error {
	arg := cmd.Flags().Arg(0)
	session := con.GetInteractive()

	enable, err := parseBoolArg(arg)
	if err != nil {
		return err
	}

	task, err := Keepalive(con.Rpc, session, enable)
	if err != nil {
		return err
	}

	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func Keepalive(rpc clientrpc.MaliceRPCClient, session *client.Session, enable bool) (*clientpb.Task, error) {
	return rpc.Keepalive(session.Context(), &implantpb.CommonBody{
		BoolArray: []bool{enable},
	})
}

func parseBoolArg(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "on", "enable", "yes":
		return true, nil
	case "false", "0", "off", "disable", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid argument %q: use true/false, on/off, enable/disable", s)
	}
}
