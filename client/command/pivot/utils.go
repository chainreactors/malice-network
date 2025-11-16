package pivot

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
	"net/url"
)

func RemDialCmd(cmd *cobra.Command, con *core.Console) error {
	pid := cmd.Flags().Arg(0)
	args := cmd.Flags().Args()[1:]
	task, err := RemDial(con.Rpc, con.GetInteractive(), pid, args)
	con.GetInteractive().Console(task, string(*con.App.Shell().Line()))
	if err != nil {
		return err
	}
	return nil
}

func GetRemLink(con *core.Console, pipe string) (string, error) {
	remPipe, ok := con.Pipelines[pipe]
	if !(ok && remPipe.GetRem() != nil) {
		return "", types.ErrNotFoundPipeline
	}
	return remPipe.GetRem().Link, nil
}

func FormatRemCmdLine(con *core.Console, pipe, mod string, remote, local *url.URL) ([]string, error) {
	remLink, err := GetRemLink(con, pipe)
	if err != nil {
		return nil, err
	}
	args := []string{"-c", remLink}
	args = append(args, "-m", mod)
	if remote != nil {
		args = append(args, "-r", remote.String())
	}
	if local != nil {
		args = append(args, "-l", local.String())
	}
	return args, nil
}

func RemDial(rpc clientrpc.MaliceRPCClient, session *client.Session, pid string, args []string) (*clientpb.Task, error) {
	task, err := rpc.RemDial(session.Context(), &implantpb.Request{
		Name: consts.ModuleRemDial,
		Args: args,
		Params: map[string]string{
			"pipeline_id": pid,
		},
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
