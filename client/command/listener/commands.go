package listener

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *core.Console) []*cobra.Command {
	listenerCmd := &cobra.Command{
		Use:   consts.CommandListener,
		Short: "List listeners on the server",
		Long:  "List listeners on the server in table form.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListenerCmd(cmd, con)
		},
		Annotations: map[string]string{
			"resource": "true",
		},
		Example: `~~~
listener
~~~`,
	}

	jobCmd := &cobra.Command{
		Use:   consts.CommandJob,
		Short: "List jobs on the server",
		Long:  "List jobs on the server in table form.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListJobsCmd(cmd, con)
		},
		Annotations: map[string]string{
			"resource": "true",
		},
		Example: `~~~
job
~~~`,
	}

	pipelineCmd := &cobra.Command{
		Use:   consts.CommandPipeline,
		Short: "Manage pipelines",
		Long:  "Start, stop, list, and delete server pipelines.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	startPipelineCmd := &cobra.Command{
		Use:   consts.CommandPipelineStart,
		Short: "Start a pipeline",
		Args:  cobra.ExactArgs(1),
		Long:  "Start the specified pipeline.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return StartPipelineCmd(cmd, con)
		},
		Example: `~~~
pipeline start tcp_test
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
		Short: "Stop a pipeline",
		Args:  cobra.ExactArgs(1),
		Long:  "Stop the specified pipeline.",
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
		Short: "List pipelines",
		Long:  "List pipelines for all listeners or for a specific listener.",
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
		Use:   consts.CommandPipelineDelete + " [pipeline]",
		Short: "Delete a pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return DeletePipelineCmd(cmd, con)
		},
	}

	common.BindArgCompletions(deletePipeCmd, nil, common.AllPipelineCompleter(con))

	pipelineCmd.AddCommand(startPipelineCmd, stopPipelineCmd, listPipelineCmd, deletePipeCmd)

	return []*cobra.Command{listenerCmd, jobCmd, pipelineCmd}
}
