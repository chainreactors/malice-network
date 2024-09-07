package tasks

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
)

func Command(con *repl.Console) []*cobra.Command {
	taskCmd := &cobra.Command{
		Use:   consts.CommandTasks,
		Short: "List tasks",
		Long:  help.GetHelpFor(consts.CommandTasks),
		Run: func(cmd *cobra.Command, args []string) {
			listTasks(cmd, con)
			return
		},
	}

	cancelTaskCmd := &cobra.Command{
		Use:   consts.ModuleCancelTask,
		Short: "Cancel a task",
		Long:  help.GetHelpFor(consts.ModuleCancelTask),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			CancelTaskCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(cancelTaskCmd, nil, common.SessionTaskComplete(con))
	return []*cobra.Command{taskCmd, cancelTaskCmd}
}
