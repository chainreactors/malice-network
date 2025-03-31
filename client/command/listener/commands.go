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

	tcpCmd := &cobra.Command{
		Use:   consts.CommandPipelineTcp,
		Short: "Register a new TCP pipeline and start it",
		Long:  "Register a new TCP pipeline with the specified listener.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return NewTcpPipelineCmd(cmd, con)
		},
		Args: cobra.MaximumNArgs(1),
		Example: `~~~
// Register a TCP pipeline with the default settings
tcp --listener tcp_default

// Register a TCP pipeline with a custom name, host, and port
tcp --name tcp_test --listener tcp_default --host 192.168.0.43 --port 5003

// Register a TCP pipeline with TLS enabled and specify certificate and key paths
tcp --listener tcp_default --tls --cert_path /path/to/cert --key_path /path/to/key
~~~`,
	}
	common.BindFlag(tcpCmd, common.TlsCertFlagSet, common.PipelineFlagSet, common.EncryptionFlagSet)

	common.BindFlagCompletions(tcpCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
		comp["host"] = carapace.ActionValues().Usage("tcp host")
		comp["port"] = carapace.ActionValues().Usage("tcp port")
		comp["cert_path"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key_path"] = carapace.ActionFiles().Usage("path to the key file")
		comp["tls"] = carapace.ActionValues().Usage("enable tls")
	})
	tcpCmd.MarkFlagRequired("listener")

	// 添加HTTP命令
	httpCmd := &cobra.Command{
		Use:   consts.HTTPPipeline,
		Short: "Register a new HTTP pipeline and start it",
		Long:  "Register a new HTTP pipeline with the specified listener.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return NewHttpPipelineCmd(cmd, con)
		},
		Args: cobra.MaximumNArgs(1),
		Example: `~~~
// Register an HTTP pipeline with the default settings
http --listener http_default

// Register an HTTP pipeline with custom headers and error page
http --name http_test --listener http_default --host 192.168.0.43 --port 8080 --headers "Content-Type=text/html" --error-page /path/to/error.html

// Register an HTTP pipeline with TLS enabled
http --listener http_default --tls --cert_path /path/to/cert --key_path /path/to/key
~~~`,
	}

	// 绑定基本标志
	common.BindFlag(httpCmd, common.TlsCertFlagSet, common.PipelineFlagSet, common.EncryptionFlagSet, common.ArtifactFlagSet, func(f *pflag.FlagSet) {
		httpCmd.Flags().StringToString("headers", nil, "HTTP response headers (key=value)")
		httpCmd.Flags().String("error-page", "", "Path to custom error page file")
		//httpCmd.Flags().String("body-prefix", "", "Prefix to add to response body")
		//httpCmd.Flags().String("body-suffix", "", "Suffix to add to response body")
	})

	common.BindFlagCompletions(httpCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
		comp["host"] = carapace.ActionValues().Usage("http host")
		comp["port"] = carapace.ActionValues().Usage("http port")
		comp["cert_path"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key_path"] = carapace.ActionFiles().Usage("path to the key file")
		comp["tls"] = carapace.ActionValues().Usage("enable tls")
		comp["error-page"] = carapace.ActionFiles().Usage("path to error page file")
		comp["headers"] = carapace.ActionValues().Usage("http headers (key=value)")
		//comp["body-prefix"] = carapace.ActionValues().Usage("prefix for response body")
		//comp["body-suffix"] = carapace.ActionValues().Usage("suffix for response body")
	})
	httpCmd.MarkFlagRequired("listener")

	bindCmd := &cobra.Command{
		Use:   consts.CommandPipelineBind,
		Short: "Register a new bind pipeline and start it",
		RunE: func(cmd *cobra.Command, args []string) error {
			return NewBindPipelineCmd(cmd, con)
		},
		Example: `
new bind pipeline
~~~
bind listener
~~~
`,
	}

	common.BindFlag(bindCmd, func(f *pflag.FlagSet) {
		f.String("listener", "", "listener id")
	})

	common.BindFlagCompletions(bindCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
	})

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

	common.BindArgCompletions(startPipelineCmd, nil,
		carapace.ActionValues().Usage("tcp pipeline name"),
		common.ListenerIDCompleter(con))

	stopPipelineCmd := &cobra.Command{
		Use:   consts.CommandPipelineStop,
		Short: "Stop a TCP pipeline",
		Args:  cobra.ExactArgs(1),
		Long:  "Stop a TCP pipeline with the specified name and listener ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			return StopPipelineCmd(cmd, con)
		},
		Example: `~~~
pipeline stop tcp_test
~~~`,
	}

	common.BindArgCompletions(stopPipelineCmd, nil,
		common.ListenerIDCompleter(con),
		common.JobsCompleter(con, stopPipelineCmd, consts.CommandPipelineTcp),
	)

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

	common.BindArgCompletions(deletePipeCmd, nil,
		carapace.ActionValues().Usage("tcp pipeline name"),
		common.ListenerIDCompleter(con))

	pipelineCmd.AddCommand(startPipelineCmd, stopPipelineCmd, listPipelineCmd, deletePipeCmd)

	remCmd := &cobra.Command{
		Use: consts.CommandRem,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		Example: `~~~
rem
~~~`,
	}
	listremCmd := &cobra.Command{
		Use:   consts.CommandListRem + " [listener]",
		Short: "List REMs in listener",
		Long:  "Use a table to list REMs along with their corresponding listeners",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListRemCmd(cmd, con)
		},
		Example: `~~~
rem
~~~`,
	}
	common.BindArgCompletions(listremCmd, nil, common.ListenerIDCompleter(con))

	newRemCmd := &cobra.Command{
		Use:   consts.CommandRemNew + " [name]",
		Short: "Register a new REM and start it",
		Long:  "Register a new REM with the specified listener.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return NewRemCmd(cmd, con)
		},
		Example: `~~~
// Register a REM with the default settings
rem new --listener listener_id

// Register a REM with a custom name and console URL
rem new --name rem_test --listener listener_id -c tcp://127.0.0.1:19966
~~~`,
	}

	common.BindFlag(newRemCmd, func(f *pflag.FlagSet) {
		f.StringP("listener", "l", "", "listener id")
		f.StringP("console", "c", "tcp://0.0.0.0", "REM console URL")
	})

	common.BindFlagCompletions(newRemCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
		comp["console"] = carapace.ActionValues().Usage("REM console URL")
	})
	newRemCmd.MarkFlagRequired("listener")

	startRemCmd := &cobra.Command{
		Use:   consts.CommandRemStart,
		Short: "Start a REM",
		Args:  cobra.ExactArgs(1),
		Long:  "Start a REM with the specified name",
		RunE: func(cmd *cobra.Command, args []string) error {
			return StartRemCmd(cmd, con)
		},
		Example: `~~~
rem start rem_test
~~~`,
	}

	common.BindArgCompletions(startRemCmd, nil,
		common.RemPipelineCompleter(con))

	stopRemCmd := &cobra.Command{
		Use:   consts.CommandRemStop,
		Short: "Stop a REM",
		Args:  cobra.ExactArgs(1),
		Long:  "Stop a REM with the specified name",
		RunE: func(cmd *cobra.Command, args []string) error {
			return StopRemCmd(cmd, con)
		},
		Example: `~~~
rem stop rem_test
~~~`,
	}

	common.BindArgCompletions(stopRemCmd, nil,
		common.RemPipelineCompleter(con))

	deleteRemCmd := &cobra.Command{
		Use:   consts.CommandPipelineDelete,
		Short: "Delete a REM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return DeleteRemCmd(cmd, con)
		},
		Example: `~~~
rem delete rem_test
~~~`,
	}

	common.BindArgCompletions(deleteRemCmd, nil,
		common.RemPipelineCompleter(con))

	remCmd.AddCommand(listremCmd, newRemCmd, startRemCmd, stopRemCmd, deleteRemCmd)

	return []*cobra.Command{listenerCmd, jobCmd, pipelineCmd, tcpCmd, bindCmd, remCmd, httpCmd}
}
