package listener

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
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
		Run: func(cmd *cobra.Command, args []string) {
			ListJobsCmd(cmd, con)
			return
		},
		Example: `~~~
job
~~~`,
	}

	tcpCmd := &cobra.Command{
		Use:   consts.CommandTcp,
		Short: "List tcp pipelines in listener",
		Long:  "Use a table to list TCP pipelines along with their corresponding listeners",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		Example: `~~~
tcp listener
~~~`,
	}

	newTCPPipelineCmd := &cobra.Command{
		Use:   consts.CommandPipelineNew + " [name] ",
		Short: "Register a new TCP pipeline and start it",
		Long: `Register a new TCP pipeline with the specified listener.
- If **name** is not provided, it will be generated in the format **listenerID_tcp_port**.
- If **host** is not specified, the default value will be **0.0.0.0**.
- If **port** is not specified, a random port will be selected from the range **10000-15000**.
- If TLS is enabled, you can provide file paths for the certificate and key.
- If no certificate or key paths are provided, the server will automatically generate a TLS certificate and key.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return NewTcpPipelineCmd(cmd, con)
		},
		Example: `~~~
// Register a TCP pipeline with the default settings
tcp register listener

// Register a TCP pipeline with a custom name, host, and port
tcp register listener --name tcp_test --host 192.168.0.43 --port 5003

// Register a TCP pipeline with TLS enabled and specify certificate and key paths
tcp register listener --tls --cert_path /path/to/cert --key_path /path/to/key
~~~`,
	}

	common.BindFlag(newTCPPipelineCmd, common.TlsCertFlagSet, common.PipelineFlagSet)

	common.BindFlagCompletions(newTCPPipelineCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
		comp["host"] = carapace.ActionValues().Usage("tcp host")
		comp["port"] = carapace.ActionValues().Usage("tcp port")
		comp["cert_path"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key_path"] = carapace.ActionFiles().Usage("path to the key file")
		comp["tls"] = carapace.ActionValues().Usage("enable tls")
	})

	tcpCmd.AddCommand(newTCPPipelineCmd)

	bindCmd := &cobra.Command{
		Use:   consts.CommandBind,
		Short: "manage bind pipeline to a listener",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	newBindCmd := &cobra.Command{
		Use:   consts.CommandPipelineNew + " [name]",
		Short: "Register a new bind pipeline and start it",
		RunE: func(cmd *cobra.Command, args []string) error {
			return NewBindPipelineCmd(cmd, con)
		},
		Example: `
new bind pipeline
~~~
bind new listener
~~~
`,
	}

	common.BindFlag(newBindCmd, func(f *pflag.FlagSet) {
		f.String("listener", "", "listener id")
	})

	common.BindFlagCompletions(newBindCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
	})

	bindCmd.AddCommand(newBindCmd)
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
		common.JobsComplete(con, stopPipelineCmd, consts.CommandTcp),
	)

	listPipelineCmd := &cobra.Command{
		Use:   consts.CommandPipelineList,
		Short: "List pipelines in listener",
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

	pipelineCmd.AddCommand(startPipelineCmd, stopPipelineCmd, listPipelineCmd)

	websiteCmd := &cobra.Command{
		Use:   consts.CommandWebsite,
		Short: "List website in listener",
		Long:  "Use a table to list websites along with their corresponding listeners",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListWebsitesCmd(cmd, con)
		},
		Example: `~~~
website [listener]
~~~`,
	}

	common.BindArgCompletions(websiteCmd, nil, common.ListenerIDCompleter(con))

	websiteRegisterCmd := &cobra.Command{
		Use:   consts.CommandPipelineNew + " [listener_id] [route_path] [content_path]",
		Short: "Register a new website and start it",
		Args:  cobra.ExactArgs(3),
		Long: `Register a new website with the specified listener.
- You must provide a web route path and the static file path. Currently, only one file can be registered.
- If **name** is not provided, it will be generated in the format **listenerID_web_port**.
- If **port** is not specified, a random port will be selected from the range **15001-20000**.
- If **content_type** is not specified, the default value will be **text/html**.
- If TLS is enabled, you can provide file paths for the certificate and key.
- If no certificate or key paths are provided, the server will automatically generate a TLS certificate and key.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return NewWebsiteCmd(cmd, con)
		},
		Example: `~~~
// Register a website with the default settings
website register name /webtest /path/to/file

// Register a website with a custom name, port, and content type
website register name /webtest /path/to/file --name web_test --port 5003 --content_type text/html
			
// Register a website with TLS enabled and specify certificate and key paths
website register name /webtest /path/to/file --tls --cert /path/to/cert --key /path/to/key
~~~`,
	}

	common.BindArgCompletions(websiteRegisterCmd, nil,
		common.ListenerIDCompleter(con),
		carapace.ActionValues().Usage("website router root path"),
		carapace.ActionFiles().Usage("website content path"))

	common.BindFlag(websiteRegisterCmd, common.TlsCertFlagSet, common.PipelineFlagSet, func(f *pflag.FlagSet) {
		f.String("content_type", "", "website content type")
	})

	common.BindFlagCompletions(websiteRegisterCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
		comp["port"] = carapace.ActionValues().Usage("website port")
		comp["content_type"] = carapace.ActionFiles().Tag("website content type")
		comp["cert"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key"] = carapace.ActionFiles().Usage("path to the key file")
		comp["tls"] = carapace.ActionValues().Usage("enable tls")
	})

	websiteStartCmd := &cobra.Command{
		Use:   consts.CommandPipelineStart,
		Short: "Start a website",
		Args:  cobra.ExactArgs(1),
		Long:  "Start a website with the specified name and listener ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			return StartWebsitePipelineCmd(cmd, con)
		},
		Example: `~~~
website start web_test 
~~~`,
	}

	common.BindArgCompletions(websiteStartCmd, nil,
		carapace.ActionValues().Usage("website name"),
		common.ListenerIDCompleter(con))

	websiteStopCmd := &cobra.Command{
		Use:   consts.CommandPipelineStop,
		Short: "Stop a website",
		Args:  cobra.ExactArgs(1),
		Long:  "Stop a website with the specified name and listener ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			return StopWebsitePipelineCmd(cmd, con)
		},
		Example: `~~~
website stop web_test listener
~~~`,
	}

	common.BindArgCompletions(websiteStopCmd, nil,
		common.ListenerIDCompleter(con),
		common.JobsComplete(con, websiteStopCmd, consts.CommandWebsite),
	)

	websiteCmd.AddCommand(websiteRegisterCmd, websiteStartCmd, websiteStopCmd)

	return []*cobra.Command{listenerCmd, jobCmd, pipelineCmd, tcpCmd, bindCmd, websiteCmd}

}
