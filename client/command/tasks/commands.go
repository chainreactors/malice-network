package tasks

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
)

func Commands(con *repl.Console) []*cobra.Command {
	taskCmd := &cobra.Command{
		Use:   consts.CommandTasks,
		Short: "List tasks",
		Run: func(cmd *cobra.Command, args []string) {
			listTasks(cmd, con)
			return
		},
	}

	fileCmd := &cobra.Command{
		Use:   consts.CommandFiles,
		Short: "List all downloaded files.",
		Run: func(cmd *cobra.Command, args []string) {
			listFiles(cmd, con)
			return
		},
	}

	cancelTaskCmd := &cobra.Command{
		Use:   consts.ModuleCancelTask + " [task_id]",
		Short: "Cancel a task by task_id",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			CancelTaskCmd(cmd, con)
			return
		},
		Example: `~~~
cancel_task <task_id>
~~~
`}

	common.BindArgCompletions(cancelTaskCmd, nil, common.SessionTaskComplete(con))
	return []*cobra.Command{taskCmd, fileCmd, cancelTaskCmd}
}

func Register(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleCancelTask,
		CancelTask,
		"",
		nil,
		common.ParseStatus,
		nil)
}
