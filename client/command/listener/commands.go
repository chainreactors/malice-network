package listener

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	listenerCmd := &cobra.Command{
		Use:   consts.CommandListener,
		Short: "List listeners in server",
		Long:  "Use a table to list listeners on the server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListenerCmd(cmd, con)
		},
		Example: `~~~
listener
~~~`,
	}

	jobCmd := &cobra.Command{
		Use:   consts.CommandJob,
		Short: "List jobs in server",
		Long:  "Use a table to list jobs on the server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListJobsCmd(cmd, con)
		},
		Example: `~~~
job
~~~`,
	}

	pipelineCmd := &cobra.Command{
		Use:   consts.CommandPipeline,
		Short: "manage pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	startPipelineCmd := &cobra.Command{
		Use:   consts.CommandPipelineStart,
		Short: "Start a TCP pipeline",
		Args:  cobra.ExactArgs(1),
		Long:  "Start a TCP pipeline with the specified name and listener ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			return StartPipelineCmd(cmd, con)
		},
		Example: `~~~
tcp start tcp_test
~~~`,
	}

	common.BindArgCompletions(startPipelineCmd, nil, common.AllPipelineCompleter(con))
	common.BindFlag(startPipelineCmd, func(f *pflag.FlagSet) {
		f.String("cert-name", "", "certificate name")
	})
	common.BindFlagCompletions(startPipelineCmd, func(comp carapace.ActionMap) {
		comp["cert-name"] = common.CertNameCompleter(con)
	})

	stopPipelineCmd := &cobra.Command{
		Use:   consts.CommandPipelineStop,
		Short: "Stop pipeline",
		Args:  cobra.ExactArgs(1),
		Long:  "Stop pipeline with the specified name and listener ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			return StopPipelineCmd(cmd, con)
		},
		Example: `~~~
pipeline stop tcp_test
~~~`,
	}

	common.BindArgCompletions(stopPipelineCmd, nil, common.AllPipelineCompleter(con))

	listPipelineCmd := &cobra.Command{
		Use:   consts.CommandPipelineList,
		Short: "List pipelines in listener",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListPipelineCmd(cmd, con)
		},
		Example: `
list all pipelines
~~~
pipeline list
~~~

list pipelines in listener
~~~
pipeline list listener_id
~~~`,
	}

	deletePipeCmd := &cobra.Command{
		Use:   consts.CommandPipelineDelete,
		Short: "Delete a pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			return DeletePipelineCmd(cmd, con)
		},
	}

	common.BindArgCompletions(deletePipeCmd, nil, common.AllPipelineCompleter(con))

	pipelineCmd.AddCommand(startPipelineCmd, stopPipelineCmd, listPipelineCmd, deletePipeCmd)

	return []*cobra.Command{listenerCmd, jobCmd, pipelineCmd}
}
