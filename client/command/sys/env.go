package sys

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func EnvCmd(cmd *cobra.Command, con *console.Console) {
	env(con)
}

func env(con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := session.SessionId
	envTask, err := con.Rpc.Env(con.ActiveTarget.Context(), &implantpb.Request{
		Name: consts.ModuleEnv,
	})
	if err != nil {
		console.Log.Errorf("Env error: %v", err)
		return
	}
	con.AddCallback(envTask.TaskId, func(msg proto.Message) {
		envSet := msg.(*implantpb.Spite).GetResponse().GetKv()
		for k, v := range envSet {
			con.SessionLog(sid).Consolef("export %s = %s\n", k, v)
		}
	})
}

func SetEnvCmd(cmd *cobra.Command, con *console.Console) {
	envName := cmd.Flags().Arg(0)
	value := cmd.Flags().Arg(1)
	if envName == "" || value == "" {
		console.Log.Errorf("required arguments missing")
		return
	}
	args := []string{envName, value}
	setEnv(args, con)
}

func setEnv(args []string, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := session.SessionId
	setEnvTask, err := con.Rpc.SetEnv(con.ActiveTarget.Context(), &implantpb.Request{
		Name: consts.ModuleSetEnv,
		Args: args,
	})
	if err != nil {
		console.Log.Errorf("SetEnv error: %v", err)
		return
	}
	con.AddCallback(setEnvTask.TaskId, func(msg proto.Message) {
		con.SessionLog(sid).Consolef("Set environment variable success\n")
	})
}

func UnsetEnvCmd(cmd *cobra.Command, con *console.Console) {
	envName := cmd.Flags().Arg(0)
	if envName == "" {
		console.Log.Errorf("required arguments missing")
		return
	}
	unSetEnv(envName, con)
}

func unSetEnv(env string, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := session.SessionId
	unsetEnvTask, err := con.Rpc.UnsetEnv(con.ActiveTarget.Context(), &implantpb.Request{
		Name:  consts.ModuleUnsetEnv,
		Input: env,
	})
	if err != nil {
		console.Log.Errorf("UnsetEnv error: %v", err)
		return
	}
	con.AddCallback(unsetEnvTask.TaskId, func(msg proto.Message) {
		con.SessionLog(sid).Consolef("Unset environment variable success\n")
	})
}
