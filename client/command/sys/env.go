package sys

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

func EnvCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	task, err := Env(con.Rpc, session)
	if err != nil {
		return err
	}
	session.Console(cmd, task, "env")
	return nil
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

func SetEnvCmd(cmd *cobra.Command, con *repl.Console) error {
	envName := cmd.Flags().Arg(0)
	value := cmd.Flags().Arg(1)
	if envName == "" || value == "" {
		return fmt.Errorf("required arguments missing")
	}
	session := con.GetInteractive()
	task, err := SetEnv(con.Rpc, session, envName, value)
	if err != nil {
		return err
	}

	session.Console(cmd, task, "setenv "+envName+" "+value)
	return nil
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

func UnsetEnvCmd(cmd *cobra.Command, con *repl.Console) error {
	envName := cmd.Flags().Arg(0)
	if envName == "" {
		return fmt.Errorf("required arguments missing")
	}
	session := con.GetInteractive()
	task, err := UnSetEnv(con.Rpc, session, envName)
	if err != nil {
		return err
	}
	session.Console(cmd, task, "unsetenv "+envName)
	return nil
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

func RegisterEnvFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleEnv,
		Env,
		"benv",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session) (*clientpb.Task, error) {
			return Env(rpc, sess)
		},
		output.ParseKVResponse, output.FormatKVResponse)

	con.AddCommandFuncHelper(
		consts.ModuleEnv,
		consts.ModuleEnv,
		"env(active())",
		[]string{
			"sess:special session",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		consts.ModuleSetEnv,
		SetEnv,
		"bsetenv",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, envName, value string) (*clientpb.Task, error) {
			return SetEnv(rpc, sess, envName, value)
		},
		output.ParseStatus,
		nil,
	)

	con.AddCommandFuncHelper(
		consts.ModuleSetEnv,
		consts.ModuleSetEnv,
		`env(active(), "name", "value")`,
		[]string{
			"sess:special session",
			"envName:env name",
			"value:env value",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		consts.ModuleUnsetEnv,
		UnSetEnv,
		"bunsetenv",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, envName string) (*clientpb.Task, error) {
			return UnSetEnv(rpc, sess, envName)
		},
		output.ParseStatus,
		nil)

	con.AddCommandFuncHelper(
		consts.ModuleUnsetEnv,
		consts.ModuleUnsetEnv,
		`unsetenv(active(), "envName")`,
		[]string{
			"sess:special session",
			"envName:env name",
		},
		[]string{"task"})

}
