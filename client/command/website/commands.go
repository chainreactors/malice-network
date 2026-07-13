package website

import (
	"fmt"

	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/wizard"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/mals"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *core.Console) []*cobra.Command {
	websiteCmd := &cobra.Command{
		Use:   consts.CommandWebsite,
		Short: "Register a new website",
		Args:  cobra.MaximumNArgs(1),
		Long:  `Register a new website with the specified listener. If **name** is not provided, it will be generated in the format **listenerID_web_port** .`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return NewWebsiteCmd(cmd, con)
		},
		Annotations: map[string]string{
			"resource": "true",
		},
		Example: `~~~
// Register a website with the default settings
website web_test --listener tcp_default --root /webtest

// Register a website with a custom name and port
website web_test --listener tcp_default --port 5003 --root /webtest

// Register a website with a temporary self-signed TLS certificate
website web_test --listener tcp_default --root /webtest --tls

// Register a website with TLS enabled using certificate files
website web_test --listener tcp_default --root /webtest --tls --cert /path/to/cert --key /path/to/key

// Register a website with TLS enabled and save the cert
website web_test --listener tcp_default --root /webtest --tls --cert /path/to/cert --key /path/to/key --save-cert --save-cert-name web_test_cert
~~~`,
	}

	common.BindFlag(websiteCmd, common.TlsCertFlagSet, common.PipelineFlagSet, func(f *pflag.FlagSet) {
		f.String("root", "/", "website root path")
		f.String("auth", "", "HTTP Basic Auth for all paths (user:pass)")
		f.Bool("save-cert", false, "save inline cert and key to certificate store")
		f.String("save-cert-name", "", "name for saved inline certificate")
		f.String("cert-comment", "", "comment for saved inline certificate")
	})

	common.BindFlagCompletions(websiteCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
		comp["port"] = carapace.ActionValues().Usage("website port")
		comp["root"] = carapace.ActionValues().Usage("website root path")
		comp["cert"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key"] = carapace.ActionFiles().Usage("path to the key file")
		comp["tls"] = carapace.ActionValues().Usage("enable tls")
		comp["cert-name"] = common.CertNameCompleter(con)
		comp["save-cert-name"] = carapace.ActionValues().Usage("saved certificate name")
	})

	common.BindArgCompletions(websiteCmd, nil, carapace.ActionValues().Usage("website name"))

	websiteListCmd := &cobra.Command{
		Use:   consts.CommandPipelineList,
		Short: "List websites",
		Long:  "List websites along with their corresponding listeners.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListWebsitesCmd(cmd, con)
		},
		Example: `~~~
website list [listener]
~~~`,
	}

	websiteInspectCmd := &cobra.Command{
		Use:   "inspect [website_name]",
		Short: "Inspect a website",
		Args:  cobra.ExactArgs(1),
		Long:  "Show listener, TLS, route, and runtime metadata for the specified website.",
		Example: `~~~
website inspect site-a
~~~`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return InspectWebsiteCmd(cmd, con)
		},
	}
	common.BindArgCompletions(websiteInspectCmd, nil, common.WebsiteCompleter(con))

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
		f.String("listener", "", "listener ID")
		f.String("cert-name", "", "certificate name")
	})

	common.BindFlagCompletions(websiteStartCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
		comp["cert-name"] = common.CertNameCompleter(con)

	})

	websiteRestartCmd := &cobra.Command{
		Use:   "restart [name]",
		Short: "Restart a website",
		Args:  cobra.ExactArgs(1),
		Long:  "Stop and then start a website with the specified name",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RestartWebsitePipelineCmd(cmd, con)
		},
		Example: `~~~
// Restart a website
website restart web_test --listener tcp_default
~~~`,
	}

	common.BindArgCompletions(websiteRestartCmd, nil, common.WebsiteCompleter(con))
	common.BindFlag(websiteRestartCmd, func(f *pflag.FlagSet) {
		f.String("listener", "", "listener ID")
	})

	common.BindFlagCompletions(websiteRestartCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
	})

	websiteTLSCmd := newWebsiteTLSCommand("tls", "Update website TLS settings", con, WebsiteTLSCmd)
	websiteCertCmd := newWebsiteTLSCommand("cert", "Bind or disable a website certificate", con, WebsiteCertCmd)

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
		Args:  validateWebsiteAddArgs,
		Long:  "Add new content to an existing website",
		RunE: func(cmd *cobra.Command, args []string) error {
			return AddWebContentCmd(cmd, con)
		},
		Example: `~~~
// Add content to a website with default web path (using filename)
website add /path/to/content.html --website web_test

// Add content to a website with custom web path and type
website add /path/to/content.html --website web_test --path /custom/path --type text/html

// Add an artifact to a website
website add --artifact beacon --website web_test --format shellcode --path /payload.bin
~~~`,
	}

	common.BindFlag(websiteAddContentCmd, func(f *pflag.FlagSet) {
		f.String("website", "", "website name (required)")
		f.String("path", "", "web path for the content (defaults to filename)")
		f.String("type", "raw", "content type of the file")
		f.String("auth", "", "HTTP Basic Auth for this path (user:pass), \"none\" to skip website default")
		f.String("name", "", "display name for the content")
		f.String("comment", "", "comment for the content")
		f.String("artifact", "", "artifact name to add instead of a local file")
		f.String("format", "", "artifact download format; shellcode is an alias for raw")
		f.String("RDI", "", "RDI conversion method")
	})
	websiteAddContentCmd.MarkFlagRequired("website")

	common.BindArgCompletions(websiteAddContentCmd, nil,
		carapace.ActionFiles().Usage("content file path"))
	common.BindFlagCompletions(websiteAddContentCmd, func(comp carapace.ActionMap) {
		comp["website"] = common.WebsiteCompleter(con)
		comp["path"] = carapace.ActionValues().Usage("web path for the content")
		comp["type"] = carapace.ActionValues().Usage("content type")
		comp["artifact"] = common.ArtifactCompleter(con)
		comp["format"] = artifactContentFormatCompleter()
	})

	websiteAddArtifactCmd := &cobra.Command{
		Use:    "add-artifact [artifact_name]",
		Short:  "Add an artifact to a website",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		Long:   "Download an artifact from the server and add it as website content",
		RunE: func(cmd *cobra.Command, args []string) error {
			return AddArtifactContentCmd(cmd, con)
		},
		Example: `~~~
// Add the native artifact binary to a website
website add-artifact beacon --website web_test --path /beacon.exe

// Add formatted shellcode bytes to a website
website add-artifact beacon --website web_test --format shellcode --path /payload.bin
~~~`,
	}

	common.BindFlag(websiteAddArtifactCmd, func(f *pflag.FlagSet) {
		f.String("website", "", "website name (required)")
		f.String("path", "", "web path for the content (defaults to artifact name and format)")
		f.String("format", "", "artifact download format; shellcode is an alias for raw")
		f.String("RDI", "", "RDI conversion method")
		f.String("type", "application/octet-stream", "content type of the artifact")
		f.String("auth", "", "HTTP Basic Auth for this path (user:pass), \"none\" to skip website default")
		f.String("name", "", "display name for the content")
		f.String("comment", "", "comment for the content")
	})
	websiteAddArtifactCmd.MarkFlagRequired("website")

	common.BindArgCompletions(websiteAddArtifactCmd, nil,
		common.ArtifactCompleter(con))
	common.BindFlagCompletions(websiteAddArtifactCmd, func(comp carapace.ActionMap) {
		comp["website"] = common.WebsiteCompleter(con)
		comp["format"] = artifactContentFormatCompleter()
		comp["path"] = carapace.ActionValues().Usage("web path for the content")
		comp["type"] = carapace.ActionValues().Usage("content type")
	})

	websiteUpdateContentCmd := &cobra.Command{
		Use:   "update [content_id] [file_path]",
		Short: "Update content in a website",
		Args:  cobra.RangeArgs(1, 2),
		Long:  "Update existing content in a website using content ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			return UpdateWebContentCmd(cmd, con)
		},
		Example: `~~~
// Update content in a website with content ID
website update 123e4567-e89b-12d3-a456-426614174000 /path/to/new_content.html --website web_test

// Update only content metadata
website update 123e4567-e89b-12d3-a456-426614174000 --name payload --comment "initial payload"
~~~`,
	}

	common.BindFlag(websiteUpdateContentCmd, func(f *pflag.FlagSet) {
		f.String("website", "", "website name (required)")
		f.String("type", "raw", "content type of the file")
		f.String("name", "", "display name for the content")
		f.String("comment", "", "comment for the content")
	})

	common.BindFlagCompletions(websiteUpdateContentCmd, func(comp carapace.ActionMap) {
		comp["website"] = common.WebsiteCompleter(con)
	})

	common.BindArgCompletions(websiteUpdateContentCmd, nil,
		common.WebContentCompleter(con),
		carapace.ActionFiles().Usage("content file path"))

	websiteUpdateContentMetadataCmd := &cobra.Command{
		Use:    "update-meta [content_id]",
		Short:  "Update website content metadata",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		Long:   "Update the display name and comment for an existing website content entry",
		RunE: func(cmd *cobra.Command, args []string) error {
			return UpdateWebContentMetadataCmd(cmd, con)
		},
		Example: `~~~
// Update content display name and comment
website update-meta 123e4567-e89b-12d3-a456-426614174000 --name beacon.exe --comment "initial payload"
~~~`,
	}

	common.BindFlag(websiteUpdateContentMetadataCmd, func(f *pflag.FlagSet) {
		f.String("name", "", "display name for the content")
		f.String("comment", "", "comment for the content")
	})

	common.BindArgCompletions(websiteUpdateContentMetadataCmd, nil,
		common.WebContentCompleter(con))

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

	websiteRouteCmd := &cobra.Command{
		Use:   "route",
		Short: "Manage website routes",
		Long:  "Add, remove, list, and inspect website content routes.",
		Example: `~~~
website route list payloads
website route inspect CONTENT_ID
website route remove CONTENT_ID
~~~`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	websiteRouteAddCmd := &cobra.Command{
		Use:   "add [file_path]",
		Short: "Add a website route",
		Args:  validateWebsiteAddArgs,
		Long:  "Add a local file_path as a website route, or use --artifact without file_path to publish a stored artifact.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return AddWebContentCmd(cmd, con)
		},
		Example: `~~~
website route add /path/to/index.html --website site-a --path /index.html

website route add --artifact beacon --website site-a --format shellcode --path /beacon.bin
~~~`,
	}
	common.BindFlag(websiteRouteAddCmd, func(f *pflag.FlagSet) {
		f.String("website", "", "website name (required)")
		f.String("path", "", "web path for the content (defaults to filename)")
		f.String("type", "raw", "content type of the file")
		f.String("auth", "", "HTTP Basic Auth for this path (user:pass), \"none\" to skip website default")
		f.String("name", "", "display name for the content")
		f.String("comment", "", "comment for the content")
		f.String("artifact", "", "artifact name to add instead of a local file")
		f.String("format", "", "artifact download format; shellcode is an alias for raw")
		f.String("RDI", "", "RDI conversion method")
	})
	websiteRouteAddCmd.MarkFlagRequired("website")
	common.BindArgCompletions(websiteRouteAddCmd, nil, carapace.ActionFiles().Usage("content file path"))
	common.BindFlagCompletions(websiteRouteAddCmd, func(comp carapace.ActionMap) {
		comp["website"] = common.WebsiteCompleter(con)
		comp["artifact"] = common.ArtifactCompleter(con)
		comp["format"] = artifactContentFormatCompleter()
	})
	websiteRouteRemoveCmd := &cobra.Command{
		Use:   "remove [content_id]",
		Short: "Remove a website route",
		Args:  cobra.ExactArgs(1),
		Long:  "Remove a website route by its content ID.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RemoveWebContentCmd(cmd, con)
		},
		Example: `~~~
website route remove 123e4567-e89b-12d3-a456-426614174000
~~~`,
	}
	common.BindArgCompletions(websiteRouteRemoveCmd, nil, common.WebContentCompleter(con))
	websiteRouteListCmd := &cobra.Command{
		Use:   "list [website_name]",
		Short: "List website routes",
		Args:  cobra.ExactArgs(1),
		Long:  "List all routes and content IDs for the specified website.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListWebContentCmd(cmd, con)
		},
		Example: `~~~
website route list site-a
~~~`,
	}
	common.BindArgCompletions(websiteRouteListCmd, nil, common.WebsiteCompleter(con))
	websiteRouteCmd.AddCommand(websiteRouteAddCmd, websiteRouteRemoveCmd, websiteRouteListCmd)

	// Enable wizard for website commands that need configuration
	common.EnableWizardForCommands(websiteCmd, websiteTLSCmd, websiteCertCmd, websiteAddContentCmd, websiteAddArtifactCmd, websiteUpdateContentCmd, websiteUpdateContentMetadataCmd)

	// Register wizard providers for dynamic options
	registerWizardProviders(websiteCmd, con)

	websiteCmd.AddCommand(websiteListCmd, websiteInspectCmd, websiteStartCmd, websiteStopCmd, websiteRestartCmd, websiteTLSCmd, websiteCertCmd,
		websiteAddContentCmd, websiteAddArtifactCmd, websiteUpdateContentCmd,
		websiteUpdateContentMetadataCmd, websiteRemoveContentCmd, websiteListContentCmd,
		websiteRouteCmd)

	return []*cobra.Command{websiteCmd}
}

func newWebsiteTLSCommand(name, short string, con *core.Console, run func(*cobra.Command, *core.Console) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name + " [website_name]",
		Short: short,
		Args:  cobra.ExactArgs(1),
		Long: `Update TLS for website_name. Select exactly one TLS mode:

- --cert-name selects an existing stored certificate.
- --cert/--key load an inline certificate pair and must be provided together.
- --generate creates a temporary self-signed certificate.
- --disable switches the website to HTTP.

Use --listener when website names are ambiguous. Saving an inline or generated certificate requires both --save-cert and --save-cert-name.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, con)
		},
		Example: fmt.Sprintf(`~~~
website %[1]s site-a --cert-name ZANY_PIN

website %[1]s site-a --cert /path/to/cert.pem --key /path/to/cert.key

website %[1]s site-a --generate

website %[1]s site-a --cert /path/to/cert.pem --key /path/to/cert.key --save-cert --save-cert-name ZANY_PIN

website %[1]s site-a --disable
~~~`, name),
	}
	common.BindArgCompletions(cmd, nil, common.WebsiteCompleter(con))
	common.BindFlag(cmd, func(f *pflag.FlagSet) {
		f.String("listener", "", "listener ID used to disambiguate the website")
		f.Bool("disable", false, "disable TLS for this website")
		f.String("cert-name", "", "existing certificate name")
		f.Bool("generate", false, "generate a temporary self-signed certificate")
		f.String("cert", "", "TLS certificate path; requires --key")
		f.String("key", "", "TLS private key path; requires --cert")
		f.Bool("save-cert", false, "save generated or inline certificate to the certificate store")
		f.String("save-cert-name", "", "name for the saved certificate; required with --save-cert")
		f.String("cert-comment", "", "comment for the saved certificate")
	})
	common.BindFlagCompletions(cmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
		comp["cert-name"] = common.CertNameCompleter(con)
		comp["generate"] = carapace.ActionValues().Usage("generate temporary self-signed certificate")
		comp["cert"] = carapace.ActionFiles().Usage("path to the certificate file")
		comp["key"] = carapace.ActionFiles().Usage("path to the private key file")
		comp["save-cert-name"] = carapace.ActionValues().Usage("saved certificate name")
	})
	return cmd
}

func validateWebsiteAddArgs(cmd *cobra.Command, args []string) error {
	artifactName, _ := cmd.Flags().GetString("artifact")
	if artifactName != "" {
		if len(args) != 0 {
			return fmt.Errorf("website add with --artifact does not accept file_path")
		}
		return nil
	}
	return cobra.ExactArgs(1)(cmd, args)
}

func artifactContentFormatCompleter() carapace.Action {
	formats := output.GetFormatsWithDescriptions()
	descriptions := make([]string, 0, len(formats)*2+2)
	for format, desc := range formats {
		descriptions = append(descriptions, format, desc)
	}
	descriptions = append(descriptions, "shellcode", "alias for raw artifact format")
	return carapace.ActionValuesDescribed(descriptions...).Tag("artifact format")
}

// registerWizardProviders registers dynamic option providers for wizard.
func registerWizardProviders(cmd *cobra.Command, con *core.Console) {
	// Listener options - fetch from cached listeners
	wizard.RegisterProviderForCommand(cmd, "listener", func() []string {
		listeners := con.SnapshotListeners()
		if len(listeners) == 0 {
			return nil
		}
		opts := make([]string, 0, len(listeners))
		for _, listener := range listeners {
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

func Register(con *core.Console) {
	con.RegisterServerFunc("website_new", NewWebsite, &mals.Helper{Group: intermediate.ListenerGroup})
	con.RegisterServerFunc("website_start", StartWebsite, &mals.Helper{Group: intermediate.ListenerGroup})
	con.RegisterServerFunc("website_stop", StopWebsite, &mals.Helper{Group: intermediate.ListenerGroup})
	con.RegisterServerFunc("website_restart", RestartWebsite, &mals.Helper{Group: intermediate.ListenerGroup})
	con.RegisterServerFunc("webcontent_add", AddWebContent, &mals.Helper{Group: intermediate.ListenerGroup})
	con.RegisterServerFunc("webcontent_add_artifact", AddArtifactContent, &mals.Helper{Group: intermediate.ListenerGroup})
	con.RegisterServerFunc("webcontent_update", UpdateWebContent, &mals.Helper{Group: intermediate.ListenerGroup})
	con.RegisterServerFunc("webcontent_update_metadata", UpdateWebContentMetadata, &mals.Helper{Group: intermediate.ListenerGroup})
	con.RegisterServerFunc("webcontent_remove", RemoveWebContent, &mals.Helper{Group: intermediate.ListenerGroup})
}
