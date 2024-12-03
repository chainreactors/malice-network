package action

import (
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	actionCmd := &cobra.Command{
		Use:   consts.CommandAction,
		Short: "Github action",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	runCmd := &cobra.Command{
		Use:   consts.CommandActionRun,
		Short: " run github workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunWorkFlowCmd(cmd, con)
		},
	}
	common.BindFlag(runCmd, common.GithubFlagSet, common.GenerateFlagSet, func(f *pflag.FlagSet) {
		f.String("type", "", "action run type")
		f.String("autorun", "", "autorun.yaml path")
		f.Uint32("artifact-id", 0, "load remote shellcode build-id")
	})
	common.BindFlagCompletions(runCmd, func(comp carapace.ActionMap) {
		comp["type"] = common.BuildTypeCompleter(con)
		comp["target"] = common.BuildTargetCompleter(con)
		comp["profile"] = common.ProfileCompleter(con)
		comp["autorun"] = carapace.ActionFiles()
	})
	runCmd.MarkFlagRequired("config")
	runCmd.MarkFlagRequired("type")
	runCmd.MarkFlagRequired("target")
	runCmd.MarkFlagRequired("profile")

	actionCmd.AddCommand(runCmd)
	return []*cobra.Command{actionCmd}
}

func Register(con *repl.Console) {
	settings := assets.GetProfile().Settings
	con.RegisterServerFunc(consts.CommandAction+"_"+consts.CommandActionRun, func(con *repl.Console, msg string) (*clientpb.Builder, error) {
		return RunWorkFlow(con, &clientpb.WorkflowRequest{
			Owner:      settings.GithubOwner,
			Repo:       settings.GithubRepo,
			Token:      settings.GithubToken,
			WorkflowId: settings.GithubWorkflowFile,
		})
	}, nil)
}
