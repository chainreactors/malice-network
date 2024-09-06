package sys

import (
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func EnvCmd(cmd *cobra.Command, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := session.SessionId
	task, err := Env(con.Rpc, session)
	if err != nil {
		console.Log.Errorf("Env error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		envSet := msg.(*implantpb.Spite).GetResponse().GetKv()
		for k, v := range envSet {
			con.SessionLog(sid).Consolef("export %s = %s\n", k, v)
		}
	})
}

func Env(rpc clientrpc.MaliceRPCClient, session *clientpb.Session) (*clientpb.Task, error) {
	task, err := rpc.Env(console.Context(session), &implantpb.Request{
		Name: consts.ModuleEnv,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}

func SetEnvCmd(cmd *cobra.Command, con *console.Console) {
	envName := cmd.Flags().Arg(0)
	value := cmd.Flags().Arg(1)
	if envName == "" || value == "" {
		console.Log.Errorf("required arguments missing")
		return
	}
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := session.SessionId
	task, err := SetEnv(con.Rpc, session, envName, value)
	if err != nil {
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		con.SessionLog(sid).Consolef("Set environment variable success\n")
	})
}

func SetEnv(rpc clientrpc.MaliceRPCClient, session *clientpb.Session, envName, value string) (*clientpb.Task, error) {
	task, err := rpc.SetEnv(console.Context(session), &implantpb.Request{
		Name: consts.ModuleSetEnv,
		Args: []string{envName, value},
	})
	if err != nil {
		return nil, err
	}
	return task, err
}

func UnsetEnvCmd(cmd *cobra.Command, con *console.Console) {
	envName := cmd.Flags().Arg(0)
	if envName == "" {
		console.Log.Errorf("required arguments missing")
		return
	}
	session := con.GetInteractive()
	if session == nil {
		return
	}
	sid := session.SessionId
	task, err := UnSetEnv(con.Rpc, session, envName)
	if err != nil {
		console.Log.Errorf("UnsetEnv error: %v", err)
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		con.SessionLog(sid).Consolef("Unset environment variable success\n")
	})
}

func UnSetEnv(rpc clientrpc.MaliceRPCClient, session *clientpb.Session, envName string) (*clientpb.Task, error) {
	task, err := rpc.UnsetEnv(console.Context(session), &implantpb.Request{
		Name:  consts.ModuleUnsetEnv,
		Input: envName,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
