package pivot

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"net/url"
)

func FormatRemCmdLine(con *repl.Console, pipe, mod string, remote, local *url.URL) ([]string, error) {
	remPipe, ok := con.Pipelines[pipe]
	if !(ok && remPipe.GetRem() != nil) {
		return nil, errs.ErrNotFoundPipeline
	}
	args := []string{"-c", remPipe.GetRem().Link}
	args = append(args, "-m", mod)
	if remote != nil {
		args = append(args, "-r", remote.String())
	}
	if local != nil {
		args = append(args, "-l", local.String())
	}
	return args, nil
}

func RemDial(rpc clientrpc.MaliceRPCClient, session *core.Session, pid string, args []string) (*clientpb.Task, error) {
	task, err := rpc.RemDial(session.Context(), &implantpb.Request{
		Name: consts.ModuleRem,
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
