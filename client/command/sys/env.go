package sys

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

func EnvCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	task, err := Env(con.Rpc, session)
	if err != nil {
		con.Log.Errorf("Env error: %v", err)
		return
	}
	session.Console(task, "env")
}

func Env(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
	task, err := rpc.Env(session.Context(), &implantpb.Request{
		Name: consts.ModuleEnv,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}

func SetEnvCmd(cmd *cobra.Command, con *repl.Console) {
	envName := cmd.Flags().Arg(0)
	value := cmd.Flags().Arg(1)
	if envName == "" || value == "" {
		con.Log.Errorf("required arguments missing")
		return
	}
	session := con.GetInteractive()
	task, err := SetEnv(con.Rpc, session, envName, value)
	if err != nil {
		return
	}

	session.Console(task, "setenv "+envName+" "+value)
}

func SetEnv(rpc clientrpc.MaliceRPCClient, session *core.Session, envName, value string) (*clientpb.Task, error) {
	task, err := rpc.SetEnv(session.Context(), &implantpb.Request{
		Name: consts.ModuleSetEnv,
		Args: []string{envName, value},
	})
	if err != nil {
		return nil, err
	}
	return task, err
}

func UnsetEnvCmd(cmd *cobra.Command, con *repl.Console) {
	envName := cmd.Flags().Arg(0)
	if envName == "" {
		con.Log.Errorf("required arguments missing")
		return
	}
	session := con.GetInteractive()
	task, err := UnSetEnv(con.Rpc, session, envName)
	if err != nil {
		con.Log.Errorf("UnsetEnv error: %v", err)
		return
	}
	session.Console(task, "unsetenv "+envName)
}

func UnSetEnv(rpc clientrpc.MaliceRPCClient, session *core.Session, envName string) (*clientpb.Task, error) {
	task, err := rpc.UnsetEnv(session.Context(), &implantpb.Request{
		Name:  consts.ModuleUnsetEnv,
		Input: envName,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
