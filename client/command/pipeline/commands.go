package pipeline

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/wizard"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *core.Console) []*cobra.Command {
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
tcp --listener listener

// Register a TCP pipeline with a custom name, host, and port
tcp tcp_test --listener listener --host 192.168.0.43 --port 5003

// Register a TCP pipeline with TLS enabled and specify certificate and key paths
tcp --listener listener --tls --cert /path/to/cert --key /path/to/key
~~~`,
	}
	common.BindFlag(tcpCmd, common.PipelineFlagSet, common.TlsCertFlagSet, common.SecureFlagSet, common.EncryptionFlagSet)

	common.BindFlagCompletions(tcpCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
		comp["host"] = carapace.ActionValues().Usage("tcp host")
		comp["port"] = carapace.ActionValues().Usage("tcp port")
		comp["cert"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key"] = carapace.ActionFiles().Usage("path to the key file")
		comp["tls"] = carapace.ActionValues().Usage("enable tls")
		comp["cert-name"] = common.CertNameCompleter(con)
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
http --listener listener

// Register an HTTP pipeline with custom headers and error page
http http_test --listener listener --host 192.168.0.43 --port 8080 --headers "Content-Type=text/html" --error-page /path/to/error.html

// Register an HTTP pipeline with TLS enabled
http --listener listener --tls --cert /path/to/cert --key /path/to/key
~~~`,
	}

	// 绑定基本标志
	common.BindFlag(httpCmd, common.PipelineFlagSet, common.TlsCertFlagSet, common.SecureFlagSet, common.EncryptionFlagSet, func(f *pflag.FlagSet) {
		httpCmd.Flags().StringToString("headers", nil, "HTTP response headers (key=value)")
		httpCmd.Flags().String("error-page", "", "Path to custom error page file")
		//httpCmd.Flags().String("body-prefix", "", "Prefix to add to response body")
		//httpCmd.Flags().String("body-suffix", "", "Suffix to add to response body")
	})

	common.BindFlagCompletions(httpCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
		comp["host"] = carapace.ActionValues().Usage("http host")
		comp["port"] = carapace.ActionValues().Usage("http port")
		comp["cert"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key"] = carapace.ActionFiles().Usage("path to the key file")
		comp["tls"] = carapace.ActionValues().Usage("enable tls")
		comp["error-page"] = carapace.ActionFiles().Usage("path to error page file")
		comp["headers"] = carapace.ActionValues().Usage("http headers (key=value)")
		comp["cert-name"] = common.CertNameCompleter(con)
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
bind --listener listener
~~~
`,
	}

	common.BindFlag(bindCmd, func(f *pflag.FlagSet) {
		f.String("listener", "", "listener id")
	})

	common.BindFlagCompletions(bindCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
	})

	remCmd := &cobra.Command{
		Use:   consts.CommandRem,
		Short: "Manage REM pipelines",
		Long:  "List, create, start, stop, and delete REM pipelines.",
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
rem list [listener]
~~~`,
	}
	common.BindArgCompletions(listremCmd, nil, common.ListenerIDCompleter(con))

	newRemCmd := &cobra.Command{
		Use:   consts.CommandRemNew + " [name]",
		Short: "Register a new REM and start it",
		Long:  "Register a new REM with the specified listener.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return NewRemCmd(cmd, con)
		},
		Example: `~~~
// Register a REM with the default settings
rem new --listener listener_id

// Register a REM with a custom name and console URL
rem new rem_test --listener listener_id -c tcp://127.0.0.1:19966
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

	updateRemCmd := &cobra.Command{
		Use:   "update",
		Short: "Update REM agent configuration",
	}

	updateIntervalCmd := &cobra.Command{
		Use:   "interval [interval_ms]",
		Short: "Dynamically change REM agent polling interval",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RemUpdateIntervalCmd(cmd, con)
		},
		Example: `~~~
rem update interval --session-id 08d6c05a 5000
rem update interval --agent-id uDM0BgG6 5000
rem update interval --pipeline-id rem_graph_api_03 --agent-id uDM0BgG6 5000
~~~`,
	}
	common.BindFlag(updateIntervalCmd, func(f *pflag.FlagSet) {
		f.String("session-id", "", "Session ID to reconfigure (resolves pipeline and agent automatically)")
		f.String("pipeline-id", "", "Pipeline name (required only when agent exists on multiple pipelines)")
		f.String("agent-id", "", "REM agent ID (pipeline is auto-resolved if unique)")
	})
	common.BindFlagCompletions(updateIntervalCmd, func(comp carapace.ActionMap) {
		comp["pipeline-id"] = common.RemPipelineCompleter(con)
		comp["agent-id"] = common.RemAgentCompleter(con)
	})
	common.BindArgCompletions(updateIntervalCmd, nil,
		carapace.ActionValues("1000", "3000", "5000", "10000", "30000", "60000").Usage("polling interval in milliseconds"))

	updateRemCmd.AddCommand(updateIntervalCmd)

	remCmd.AddCommand(listremCmd, newRemCmd, startRemCmd, stopRemCmd, deleteRemCmd, updateRemCmd)

	// WebShell pipeline commands
	webshellCmd := &cobra.Command{
		Use:   "webshell",
		Short: "Manage WebShell pipelines",
		Long:  "List, create, start, stop, and delete WebShell bridge pipelines.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	listWebShellCmd := &cobra.Command{
		Use:   "list [listener]",
		Short: "List webshell pipelines",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListWebShellCmd(cmd, con)
		},
	}
	common.BindArgCompletions(listWebShellCmd, nil, common.ListenerIDCompleter(con))

	newWebShellCmd := &cobra.Command{
		Use:   "new [name]",
		Short: "Register a new webshell pipeline",
		Long:  "Register a CustomPipeline(type=webshell) for the webshell-bridge binary to connect to.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return NewWebShellCmd(cmd, con)
		},
		Example: `~~~
webshell new --listener my-listener
webshell new ws1 --listener my-listener
~~~`,
	}
	common.BindFlag(newWebShellCmd, func(f *pflag.FlagSet) {
		f.StringP("listener", "l", "", "listener id")
	})
	common.BindFlagCompletions(newWebShellCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
	})
	newWebShellCmd.MarkFlagRequired("listener")

	startWebShellCmd := &cobra.Command{
		Use:   "start <name>",
		Short: "Start a webshell pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return StartWebShellCmd(cmd, con)
		},
	}
	common.BindFlag(startWebShellCmd, func(f *pflag.FlagSet) {
		f.StringP("listener", "l", "", "listener id")
	})
	common.BindFlagCompletions(startWebShellCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
	})
	common.BindArgCompletions(startWebShellCmd, nil, common.PipelineCompleter(con, webshellPipelineType))

	stopWebShellCmd := &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop a webshell pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return StopWebShellCmd(cmd, con)
		},
	}
	common.BindFlag(stopWebShellCmd, func(f *pflag.FlagSet) {
		f.StringP("listener", "l", "", "listener id")
	})
	common.BindFlagCompletions(stopWebShellCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
	})
	common.BindArgCompletions(stopWebShellCmd, nil, common.PipelineCompleter(con, webshellPipelineType))

	deleteWebShellCmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a webshell pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return DeleteWebShellCmd(cmd, con)
		},
	}
	common.BindFlag(deleteWebShellCmd, func(f *pflag.FlagSet) {
		f.StringP("listener", "l", "", "listener id")
	})
	common.BindFlagCompletions(deleteWebShellCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
	})
	common.BindArgCompletions(deleteWebShellCmd, nil, common.PipelineCompleter(con, webshellPipelineType))

	webshellCmd.AddCommand(listWebShellCmd, newWebShellCmd, startWebShellCmd, stopWebShellCmd, deleteWebShellCmd)

	// Enable wizard for pipeline commands
	common.EnableWizardForCommands(tcpCmd, httpCmd, bindCmd, newRemCmd, newWebShellCmd)

	// Register wizard providers for dynamic options
	registerWizardProviders(tcpCmd, con)
	registerWizardProviders(httpCmd, con)
	registerWizardProviders(bindCmd, con)
	registerWizardProviders(newRemCmd, con)
	registerWizardProviders(newWebShellCmd, con)

	return []*cobra.Command{tcpCmd, httpCmd, bindCmd, remCmd, webshellCmd}
}

// registerWizardProviders registers dynamic option providers for wizard.
func registerWizardProviders(cmd *cobra.Command, con *core.Console) {
	// Listener options - fetch from cached listeners
	wizard.RegisterProviderForCommand(cmd, "listener", func() []string {
		if len(con.Listeners) == 0 {
			return nil
		}
		opts := make([]string, 0, len(con.Listeners))
		for _, listener := range con.Listeners {
			if listener.Id != "" {
				opts = append(opts, listener.Id)
			}
		}
		return opts
	})

	// Certificate name options - fetch from server
	wizard.RegisterProviderForCommand(cmd, "cert-name", func() []string {
		certificates, err := con.Rpc.GetAllCertificates(con.Context(), &clientpb.Empty{})
		if err != nil || len(certificates.Certs) == 0 {
			return nil
		}
		opts := make([]string, 0, len(certificates.Certs)+1)
		opts = append(opts, "") // Allow empty option
		for _, c := range certificates.Certs {
			if c.Cert.Name != "" {
				opts = append(opts, c.Cert.Name)
			}
		}
		return opts
	})
}
