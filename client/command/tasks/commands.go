package tasks

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	taskCmd := &cobra.Command{
		Use:   consts.CommandTasks,
		Short: "List tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListTasks(cmd, con)
		},
	}

	common.Bind("tasks", true, taskCmd, func(f *pflag.FlagSet) {
		f.BoolP("all", "a", false, "show all tasks")
	})

	fileCmd := &cobra.Command{
		Use:   consts.CommandFiles,
		Short: "List all downloaded files.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListFiles(cmd, con)
		},
	}

	cancelTaskCmd := &cobra.Command{
		Use:   consts.ModuleCancelTask + " [task_id]",
		Short: "Cancel a task by task_id",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return CancelTaskCmd(cmd, con)
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

	con.AddInternalFuncHelper(
		consts.ModuleCancelTask,
		consts.ModuleCancelTask,
		"cancel_task <task_id>",
		[]string{
			"sess:special session",
			"task_id:task id",
		},
		[]string{"task"})

}
