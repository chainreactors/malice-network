package config

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

var githubConfig struct {
	Repo     string
	Owner    string
	Token    string
	Workflow string
}

func GetGithubConfigCmd(cmd *cobra.Command, con *repl.Console) error {
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

func UpdateGithubConfigCmd(cmd *cobra.Command, con *repl.Console) error {
	owner, repo, token, workflow := common.ParseGithubFlags(cmd)
	_, err := UpdateGithubConfig(con, owner, repo, token, workflow)
	if err != nil {
		return err
	}
	con.Log.Console("Update github config success\n")
	return nil
}

func UpdateGithubConfig(con *repl.Console, owner, repo, token, workflow string) (*clientpb.Empty, error) {
	return con.Rpc.UpdateGithubConfig(con.Context(), &clientpb.GithubWorkflowRequest{
		Owner:      owner,
		Repo:       repo,
		Token:      token,
		WorkflowId: workflow,
	})
}
