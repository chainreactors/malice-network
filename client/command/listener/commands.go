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
		Run: func(cmd *cobra.Command, args []string) {
			ListenerCmd(cmd, con)
			return
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
			listJobsCmd(cmd, con)
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
		Run: func(cmd *cobra.Command, args []string) {
			listTcpCmd(cmd, con)
			return
		},
		GroupID: consts.ListenerGroup,
		Example: `~~~
tcp listener
~~~`,
	}

	common.BindArgCompletions(tcpCmd, nil, common.ListenerIDCompleter(con))

	tcpRegisterCmd := &cobra.Command{
		Use:   consts.CommandRegister + " [listener_id] ",
		Short: "Register a new TCP pipeline and start it",
		Args:  cobra.ExactArgs(1),
		Long: `Register a new TCP pipeline with the specified listener.
- If **name** is not provided, it will be generated in the format **listenerID_tcp_port**.
- If **host** is not specified, the default value will be **0.0.0.0**.
- If **port** is not specified, a random port will be selected from the range **10000-15000**.
- If TLS is enabled, you can provide file paths for the certificate and key.
- If no certificate or key paths are provided, the server will automatically generate a TLS certificate and key.`,
		Run: func(cmd *cobra.Command, args []string) {
			newTcpPipelineCmd(cmd, con)
			return
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

	common.BindArgCompletions(tcpRegisterCmd, nil,
		common.ListenerIDCompleter(con))

	common.BindFlag(tcpRegisterCmd, common.TlsCertFlagSet, common.PipelineFlagSet)

	common.BindFlagCompletions(tcpRegisterCmd, func(comp carapace.ActionMap) {
		comp["name"] = carapace.ActionValues().Usage("tcp name")
		comp["host"] = carapace.ActionValues().Usage("tcp host")
		comp["port"] = carapace.ActionValues().Usage("tcp port")
		comp["cert_path"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key_path"] = carapace.ActionFiles().Usage("path to the key file")
		comp["tls"] = carapace.ActionValues().Usage("enable tls")
	})

	tcpStartCmd := &cobra.Command{
		Use:   consts.CommandPipelineStart,
		Short: "Start a TCP pipeline",
		Args:  cobra.ExactArgs(2),
		Long:  "Start a TCP pipeline with the specified name and listener ID",
		Run: func(cmd *cobra.Command, args []string) {
			startTcpPipelineCmd(cmd, con)
			return
		},
		Example: `~~~
tcp start tcp_test listener
~~~`,
	}

	common.BindArgCompletions(tcpStartCmd, nil,
		carapace.ActionValues().Usage("tcp pipeline name"),
		common.ListenerIDCompleter(con))

	tcpStopCmd := &cobra.Command{
		Use:   consts.CommandPipelineStop,
		Short: "Stop a TCP pipeline",
		Args:  cobra.ExactArgs(2),
		Long:  "Stop a TCP pipeline with the specified name and listener ID",
		Run: func(cmd *cobra.Command, args []string) {
			stopTcpPipelineCmd(cmd, con)
			return
		},
		Example: `~~~
tcp stop tcp_test listener
~~~`,
	}

	common.BindArgCompletions(tcpStopCmd, nil,
		carapace.ActionValues().Usage("tcp pipeline name"),
		common.ListenerIDCompleter(con))

	tcpCmd.AddCommand(tcpRegisterCmd, tcpStartCmd, tcpStopCmd)

	websiteCmd := &cobra.Command{
		Use:   consts.CommandWebsite,
		Short: "List website in listener",
		Long:  "Use a table to list websites along with their corresponding listeners",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			listWebsitesCmd(cmd, con)
			return
		},
		Example: `~~~
website listener
~~~`,
	}

	common.BindArgCompletions(websiteCmd, nil, common.ListenerIDCompleter(con))

	websiteRegisterCmd := &cobra.Command{
		Use:   consts.CommandRegister + " [listener_id] [route_path] [content_path]",
		Short: "Register a new website and start it",
		Args:  cobra.ExactArgs(3),
		Long: `Register a new website with the specified listener.
- You must provide a web route path and the static file path. Currently, only one file can be registered.
- If **name** is not provided, it will be generated in the format **listenerID_web_port**.
- If **port** is not specified, a random port will be selected from the range **15001-20000**.
- If **content_type** is not specified, the default value will be **text/html**.
- If TLS is enabled, you can provide file paths for the certificate and key.
- If no certificate or key paths are provided, the server will automatically generate a TLS certificate and key.`,
		Run: func(cmd *cobra.Command, args []string) {
			newWebsiteCmd(cmd, con)
			return
		},
		Example: `~~~
// Register a website with the default settings
website register listener /webtest /path/to/file

// Register a website with a custom name, port, and content type
website register listener /webtest /path/to/file --name web_test --port 5003 --content_type text/html
			
// Register a website with TLS enabled and specify certificate and key paths
website register listener /webtest /path/to/file --tls --cert_path /path/to/cert --key_path /path/to/key
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
		comp["name"] = carapace.ActionValues().Usage("website name")
		comp["port"] = carapace.ActionValues().Usage("website port")
		comp["content_type"] = carapace.ActionFiles().Tag("website content type")
		comp["cert_path"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key_path"] = carapace.ActionFiles().Usage("path to the key file")
		comp["tls"] = carapace.ActionValues().Usage("enable tls")
	})

	websiteStartCmd := &cobra.Command{
		Use:   consts.CommandPipelineStart,
		Short: "Start a website",
		Args:  cobra.ExactArgs(2),
		Long:  "Start a website with the specified name and listener ID",
		Run: func(cmd *cobra.Command, args []string) {
			startWebsitePipelineCmd(cmd, con)
			return
		},
		Example: `~~~
website start web_test listener
~~~`,
	}

	common.BindArgCompletions(websiteStartCmd, nil,
		carapace.ActionValues().Usage("website name"),
		common.ListenerIDCompleter(con))

	websiteStopCmd := &cobra.Command{
		Use:   consts.CommandPipelineStop,
		Short: "Stop a website",
		Args:  cobra.ExactArgs(2),
		Long:  "Stop a website with the specified name and listener ID",
		Run: func(cmd *cobra.Command, args []string) {
			stopWebsitePipelineCmd(cmd, con)
			return
		},
		Example: `~~~
website stop web_test listener
~~~`,
	}

	common.BindArgCompletions(websiteStopCmd, nil,
		carapace.ActionValues().Usage("website name"),
		common.ListenerIDCompleter(con))

	websiteCmd.AddCommand(websiteRegisterCmd, websiteStartCmd, websiteStopCmd)

	return []*cobra.Command{listenerCmd, jobCmd, tcpCmd, websiteCmd}

}
