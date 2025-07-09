package website

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/mals"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	websiteCmd := &cobra.Command{
		Use:   consts.CommandWebsite,
		Short: "Register a new website",
		Args:  cobra.MaximumNArgs(1),
		Long:  `Register a new website with the specified listener. If **name** is not provided, it will be generated in the format **listenerID_web_port**.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return NewWebsiteCmd(cmd, con)
		},
		Example: `~~~
// Register a website with the default settings
website web_test --listener tcp_default --root /webtest

// Register a website with a custom name and port
website web_test --listener tcp_default --port 5003 --root /webtest

// Register a website with TLS enabled
website web_test --listener tcp_default --root /webtest --tls --cert /path/to/cert --key /path/to/key
~~~`,
	}

	common.BindFlag(websiteCmd, common.TlsCertFlagSet, common.PipelineFlagSet, func(f *pflag.FlagSet) {
		f.String("root", "/", "website root path")
	})

	common.BindFlagCompletions(websiteCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
		comp["port"] = carapace.ActionValues().Usage("website port")
		comp["root"] = carapace.ActionValues().Usage("website root path")
		comp["cert"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key"] = carapace.ActionFiles().Usage("path to the key file")
		comp["tls"] = carapace.ActionValues().Usage("enable tls")
		comp["cert-name"] = common.CertNameCompleter(con)
	})

	common.BindArgCompletions(websiteCmd, nil, carapace.ActionValues().Usage("website name"))

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
website start web_test 
~~~`,
	}

	common.BindArgCompletions(websiteStartCmd, nil, common.WebsiteCompleter(con))
	common.BindFlag(websiteStartCmd, func(f *pflag.FlagSet) {
		f.String("cert-name", "", "certificate name")
	})

	common.BindFlagCompletions(websiteStartCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
		comp["cert-name"] = common.CertNameCompleter(con)

	})

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
		common.WebsiteCompleter(con))

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

	common.BindFlag(websiteAddContentCmd, func(f *pflag.FlagSet) {
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
		common.WebContentCompleter(con),
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
		common.WebContentCompleter(con))

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

	websiteCmd.AddCommand(websiteListCmd, websiteStartCmd, websiteStopCmd,
		websiteAddContentCmd, websiteUpdateContentCmd, websiteRemoveContentCmd, websiteListContentCmd)

	return []*cobra.Command{websiteCmd}
}

func Register(con *repl.Console) {
	con.RegisterServerFunc("website_new", NewWebsite, &mals.Helper{Group: intermediate.ListenerGroup})
	con.RegisterServerFunc("website_start", StartWebsite, &mals.Helper{Group: intermediate.ListenerGroup})
	con.RegisterServerFunc("website_stop", StopWebsite, &mals.Helper{Group: intermediate.ListenerGroup})
	con.RegisterServerFunc("webcontent_add", AddWebContent, &mals.Helper{Group: intermediate.ListenerGroup})
	con.RegisterServerFunc("webcontent_update", UpdateWebContent, &mals.Helper{Group: intermediate.ListenerGroup})
	con.RegisterServerFunc("webcontent_remove", RemoveWebContent, &mals.Helper{Group: intermediate.ListenerGroup})
}
