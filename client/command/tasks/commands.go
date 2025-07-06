package tasks

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	taskCmd := &cobra.Command{
		Use:   consts.CommandTasks,
		Short: "List tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetTasksCmd(cmd, con)
		},
	}

	common.Bind("tasks", true, taskCmd, func(f *pflag.FlagSet) {
		f.BoolP("all", "a", false, "show all tasks")
	})

	fetchTaskCmd := &cobra.Command{
		Use:   consts.CommandTaskFetch,
		Short: "Fetch the details of a task",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return TaskFetchCmd(cmd, con)
		},
	}

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

	common.BindArgCompletions(cancelTaskCmd, nil, common.SessionTaskCompleter(con))

	listTaskCmd := &cobra.Command{
		Use:   consts.ModuleListTask,
		Short: "List all tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListTaskCmd(cmd, con)
		},
		Example: `~~~
list_task
~~~
`}

	queryTaskCmd := &cobra.Command{
		Use:   consts.ModuleQueryTask + " [task_id]",
		Short: "Query a task by task_id",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return QueryTaskCmd(cmd, con)
		},
		Example: `~~~
query_task <task_id>
~~~
`}

	common.BindArgCompletions(queryTaskCmd, nil, common.SessionTaskCompleter(con))
	return []*cobra.Command{taskCmd, fetchTaskCmd, fileCmd, cancelTaskCmd, listTaskCmd, queryTaskCmd}
}

func Register(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleCancelTask,
		CancelTask,
		"",
		nil,
		output.ParseStatus,
		nil)

	con.AddCommandFuncHelper(
		consts.ModuleCancelTask,
		consts.ModuleCancelTask,
		"cancel_task <task_id>",
		[]string{
			"sess:special session",
			"task_id:task id",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		consts.ModuleListTask,
		ListTask,
		"",
		nil,
		func(content *clientpb.TaskContext) (interface{}, error) {
			logs.Log.Infof("list task\n")
			return tui.RendStructDefault(content.Spite.GetTaskList()), nil
		},
		nil)

	con.AddCommandFuncHelper(
		consts.ModuleListTask,
		consts.ModuleListTask,
		"list_task",
		[]string{
			"sess:special session",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		consts.ModuleQueryTask,
		QueryTask,
		"",
		nil,
		func(content *clientpb.TaskContext) (interface{}, error) {
			return tui.RendStructDefault(content.Spite.GetTaskInfo()), nil
		},
		nil)

	con.AddCommandFuncHelper(
		consts.ModuleQueryTask,
		consts.ModuleQueryTask,
		"query_task <task_id>",
		[]string{
			"sess:special session",
			"task_id:task id",
		},
		[]string{"task"})
}
