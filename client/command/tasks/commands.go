package tasks

import (
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *core.Console) []*cobra.Command {
	taskCmd := &cobra.Command{
		Use:   consts.CommandTasks,
		Short: "List tasks",
		Long:  "List tasks",
		Example: `~~~
tasks
tasks --all
tasks info 59
~~~`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetTasksCmd(cmd, con)
		},
		Annotations: map[string]string{
			"resource": "true",
		},
	}

	common.Bind("tasks", true, taskCmd, func(f *pflag.FlagSet) {
		f.BoolP("all", "a", false, "show all tasks")
	})

	taskInfoCmd := &cobra.Command{
		Use:   "info [task_id]",
		Short: "Show task request metadata",
		Long:  "Show request metadata for a task, with optional raw request data and results.",
		Example: `~~~
tasks info 59
tasks info 59 --results --json
~~~`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return TaskInfoCmd(cmd, con)
		},
	}
	taskInfoCmd.Flags().Bool("raw", false, "include raw task request")
	taskInfoCmd.Flags().Bool("results", false, "include task results")
	taskInfoCmd.Flags().Bool("json", false, "output as JSON")
	common.BindArgCompletions(taskInfoCmd, nil, common.SessionTaskCompleter(con))
	taskCmd.AddCommand(taskInfoCmd)

	fetchTaskCmd := &cobra.Command{
		Use:   consts.CommandTaskFetch + " [task_id]",
		Short: "Fetch the details of a task",
		Long:  "Fetch task results for the active session and optionally save them to a file.",
		Example: `~~~
fetch_task 59
fetch_task 59 --file --output ./task-59.bin
~~~`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return TaskFetchCmd(cmd, con)
		},
	}

	common.Bind("task_fetch", false, fetchTaskCmd, func(f *pflag.FlagSet) {
		f.BoolP("file", "f", false, "output to file")
		f.StringP("output", "o", "", "output file path")
	})
	common.BindArgCompletions(fetchTaskCmd, nil, common.SessionTaskCompleter(con))

	fileCmd := &cobra.Command{
		Use:   consts.CommandFiles,
		Short: "List all downloaded files.",
		Long:  "List files downloaded from the active implant and stored by the client.",
		Example: `~~~
files
~~~`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListFiles(cmd, con)
		},
	}

	cancelTaskCmd := &cobra.Command{
		Use:   consts.ModuleCancelTask + " [task_id]",
		Short: "Cancel a task by task_id",
		Long:  "Request cancellation of a queued or running implant task by ID.",
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
		Long:  "Ask the active implant to list its task queue.",
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
		Long:  "Ask the active implant for the current state of a task by ID.",
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

func Register(con *core.Console) {
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
