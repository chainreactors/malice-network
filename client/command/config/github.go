package config

import (
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func GetGithubConfigCmd(cmd *cobra.Command, con *core.Console) error {
	resp, err := con.Rpc.GetGithubConfig(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}

	token := "(not set)"
	if resp.Token != "" {
		if len(resp.Token) > 8 {
			token = resp.Token[:4] + "..." + resp.Token[len(resp.Token)-4:]
		} else {
			token = "****"
		}
	}

	values := map[string]string{
		"Owner":    resp.Owner,
		"Repo":     resp.Repo,
		"Token":    token,
		"Workflow": resp.WorkflowId,
	}
	keys := []string{"Owner", "Repo", "Token", "Workflow"}
	con.Log.Console(common.NewKVTable("Github", keys, values).View() + "\n")
	return nil
}

func UpdateGithubConfigCmd(cmd *cobra.Command, con *core.Console) error {
	current, err := con.Rpc.GetGithubConfig(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}

	githubConfig := mergeGithubUpdate(current, cmd)
	_, err = con.Rpc.UpdateGithubConfig(con.Context(), githubConfig)
	if err != nil {
		return err
	}
	con.Log.Console("Update github config success\n")
	return nil
}
