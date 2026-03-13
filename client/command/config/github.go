package config

import (
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

var githubConfig struct {
	Repo     string
	Owner    string
	Token    string
	Workflow string
}

func GetGithubConfigCmd(cmd *cobra.Command, con *core.Console) error {
	resp, err := con.Rpc.GetGithubConfig(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	githubConfig.Repo = resp.Repo
	githubConfig.Owner = resp.Owner
	githubConfig.Token = resp.Token
	githubConfig.Workflow = resp.WorkflowId
	con.Log.Console(tui.RendStructDefault(githubConfig) + "\n")
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
