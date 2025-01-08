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
		Use:   consts.CommandPipelineTcp,
		Short: "List tcp pipelines in listener",
		Long:  "Use a table to list TCP pipelines along with their corresponding listeners",
		Args:  cobra.MaximumNArgs(1),
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
If **name** is not provided, it will be generated in the format **listenerID_tcp_port**.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return NewTcpPipelineCmd(cmd, con)
		},
		Args: cobra.MaximumNArgs(1),
		Example: `~~~
// Register a TCP pipeline with the default settings
tcp register --listener tcp_default

// Register a TCP pipeline with a custom name, host, and port
tcp register --name tcp_test --listener tcp_default --host 192.168.0.43 --port 5003

// Register a TCP pipeline with TLS enabled and specify certificate and key paths
tcp register --listener tcp_default --tls --cert_path /path/to/cert --key_path /path/to/key
~~~`,
	}

	common.BindFlag(newTCPPipelineCmd, common.TlsCertFlagSet, common.PipelineFlagSet, common.EncryptionFlagSet)

	common.BindFlagCompletions(newTCPPipelineCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
		comp["host"] = carapace.ActionValues().Usage("tcp host")
		comp["port"] = carapace.ActionValues().Usage("tcp port")
		comp["cert_path"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key_path"] = carapace.ActionFiles().Usage("path to the key file")
		comp["tls"] = carapace.ActionValues().Usage("enable tls")
	})
	newTCPPipelineCmd.MarkFlagRequired("listener")
	tcpCmd.AddCommand(newTCPPipelineCmd)

	bindCmd := &cobra.Command{
		Use:   consts.CommandPipelineBind,
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

	websiteCmd := &cobra.Command{
		Use:   consts.CommandWebsite,
		Short: "website manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	common.BindArgCompletions(websiteCmd, nil, common.ListenerIDCompleter(con))

	websiteListCmd := &cobra.Command{
		Use:   consts.CommandPipelineList,
		Short: "List website in listener",
		Long:  "Use a table to list websites along with their corresponding listeners",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListWebsitesCmd(cmd, con)
		},
		Example: `~~~
website [listener]
~~~`,
	}

	websiteRegisterCmd := &cobra.Command{
		Use:   consts.CommandPipelineNew + " [name]",
		Short: "Register a new website",
		Args:  cobra.MaximumNArgs(1),
		Long:  `Register a new website with the specified listener. If **name** is not provided, it will be generated in the format **listenerID_web_port**.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return NewWebsiteCmd(cmd, con)
		},
		Example: `~~~
// Register a website with the default settings
website new --listener tcp_default --root /webtest

// Register a website with a custom name and port
website new web_test --listener tcp_default --port 5003 --root /webtest

// Register a website with TLS enabled
website new --listener tcp_default --root /webtest --tls --cert /path/to/cert --key /path/to/key
~~~`,
	}

	common.BindFlag(websiteRegisterCmd, common.TlsCertFlagSet, common.PipelineFlagSet, func(f *pflag.FlagSet) {
		f.String("root", "/", "website root path")
	})

	common.BindFlagCompletions(websiteRegisterCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
		comp["port"] = carapace.ActionValues().Usage("website port")
		comp["root"] = carapace.ActionValues().Usage("website root path")
		comp["cert"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key"] = carapace.ActionFiles().Usage("path to the key file")
		comp["tls"] = carapace.ActionValues().Usage("enable tls")
	})

	websiteStartCmd := &cobra.Command{
		Use:   consts.CommandPipelineStart + " [name]",
		Short: "Start a website",
		Args:  cobra.ExactArgs(1),
		Long:  "Start a website with the specified name",
		RunE: func(cmd *cobra.Command, args []string) error {
			return StartWebsitePipelineCmd(cmd, con)
		},
		Example: `~~~
// Start a website
website start web_test --listener tcp_default
~~~`,
	}

	common.BindFlag(websiteStartCmd, func(f *pflag.FlagSet) {
		f.String("listener", "", "listener ID")
	})

	common.BindFlagCompletions(websiteStartCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
	})

	common.BindArgCompletions(websiteStartCmd, nil,
		carapace.ActionValues().Usage("website name"))

	websiteStopCmd := &cobra.Command{
		Use:   consts.CommandPipelineStop + " [name]",
		Short: "Stop a website",
		Args:  cobra.ExactArgs(1),
		Long:  "Stop a website with the specified name",
		RunE: func(cmd *cobra.Command, args []string) error {
			return StopWebsitePipelineCmd(cmd, con)
		},
		Example: `~~~
// Stop a website
website stop web_test --listener tcp_default
~~~`,
	}

	common.BindFlag(websiteStopCmd, func(f *pflag.FlagSet) {
		f.String("listener", "", "listener ID")
	})

	common.BindFlagCompletions(websiteStopCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
	})

	common.BindArgCompletions(websiteStopCmd, nil,
		carapace.ActionValues().Usage("website name"))

	websiteAddContentCmd := &cobra.Command{
		Use:   "add [file_path]",
		Short: "Add content to a website",
		Args:  cobra.ExactArgs(1),
		Long:  "Add new content to an existing website",
		RunE: func(cmd *cobra.Command, args []string) error {
			return AddWebContentCmd(cmd, con)
		},
		Example: `~~~
// Add content to a website with default web path (using filename)
website add /path/to/content.html --website web_test

// Add content to a website with custom web path and type
website add /path/to/content.html --website web_test --path /custom/path --type text/html
~~~`,
	}

	common.BindFlag(websiteAddContentCmd, common.EncryptionFlagSet, func(f *pflag.FlagSet) {
		f.String("website", "", "website name (required)")
		f.String("path", "", "web path for the content (defaults to filename)")
		f.String("type", "raw", "content type of the file")
	})
	websiteAddContentCmd.MarkFlagRequired("website")

	common.BindArgCompletions(websiteAddContentCmd, nil,
		carapace.ActionFiles().Usage("content file path"))
	common.BindFlagCompletions(websiteAddContentCmd, func(comp carapace.ActionMap) {
		comp["website"] = common.WebsiteCompleter(con)
	})

	websiteUpdateContentCmd := &cobra.Command{
		Use:   "update [content_id] [file_path]",
		Short: "Update content in a website",
		Args:  cobra.ExactArgs(2),
		Long:  "Update existing content in a website using content ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			return UpdateWebContentCmd(cmd, con)
		},
		Example: `~~~
// Update content in a website with content ID
website update 123e4567-e89b-12d3-a456-426614174000 /path/to/new_content.html --website web_test
~~~`,
	}

	common.BindFlag(websiteUpdateContentCmd, func(f *pflag.FlagSet) {
		f.String("website", "", "website name (required)")
		f.String("type", "raw", "content type of the file")
	})

	common.BindFlagCompletions(websiteUpdateContentCmd, func(comp carapace.ActionMap) {
		comp["website"] = common.WebsiteCompleter(con)
	})

	common.BindArgCompletions(websiteUpdateContentCmd, nil,
		common.WebContentCompleter(con, ""),
		carapace.ActionFiles().Usage("content file path"))

	websiteRemoveContentCmd := &cobra.Command{
		Use:   "remove [content_id]",
		Short: "Remove content from a website",
		Args:  cobra.ExactArgs(1),
		Long:  "Remove content from an existing website using content ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RemoveWebContentCmd(cmd, con)
		},
		Example: `~~~
// Remove content from a website using content ID
website remove 123e4567-e89b-12d3-a456-426614174000
~~~`,
	}

	common.BindArgCompletions(websiteRemoveContentCmd, nil,
		common.WebContentCompleter(con, ""))

	websiteListContentCmd := &cobra.Command{
		Use:   "list-content [website_name]",
		Short: "List content in a website",
		Long:  "List all content in a website with detailed information",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListWebContentCmd(cmd, con)
		},
		Example: `~~~
// List all content in a website with detailed information
website list-content web_test
~~~`,
	}

	common.BindArgCompletions(websiteListContentCmd, nil,
		common.WebsiteCompleter(con))

	websiteCmd.AddCommand(websiteListCmd, websiteRegisterCmd, websiteStartCmd, websiteStopCmd,
		websiteAddContentCmd, websiteUpdateContentCmd, websiteRemoveContentCmd, websiteListContentCmd)

	return []*cobra.Command{listenerCmd, jobCmd, pipelineCmd, tcpCmd, bindCmd, websiteCmd}

}
