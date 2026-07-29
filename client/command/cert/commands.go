package cert

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *core.Console) []*cobra.Command {
	certCmd := &cobra.Command{
		Use:   consts.CommandCert,
		Short: "Cert list",
		Long:  "List certificates stored on the server and manage their lifecycle.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetCertCmd(cmd, con)
		},
		Example: `~~~
cert
~~~`,
	}

	importCmd := &cobra.Command{
		Use:   consts.CommandCertImport,
		Short: "import a new cert",
		Long:  "Import a certificate and private key into the server certificate store.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ImportCmd(cmd, con)
		},
		Example: `~~~
// generate a imported cert to server
cert import --cert cert_file_path --key key_file_path --ca-cert ca_cert_path
~~~`,
	}

	common.BindFlag(importCmd, common.ImportSet)
	common.BindFlag(importCmd, func(f *pflag.FlagSet) {
		f.String("name", "", "certificate name")
		f.String("comment", "", "certificate comment")
	})
	_ = importCmd.MarkFlagRequired("cert")
	_ = importCmd.MarkFlagRequired("key")
	common.BindFlagCompletions(importCmd, func(comp carapace.ActionMap) {
		comp["cert"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key"] = carapace.ActionFiles().Usage("path to the key file")
		comp["ca-cert"] = carapace.ActionFiles().Usage("path to the ca cert file")
		comp["name"] = carapace.ActionValues().Usage("certificate name")
	})

	selfSignCmd := &cobra.Command{
		Use:   consts.CommandCertSelfSigned,
		Short: "generate a self-signed cert",
		Long:  "Generate a self-signed certificate and store it on the server.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return SelfSignedCmd(cmd, con)
		},
		Example: `~~~
// generate a self-signed cert without using certificate information
cert self_signed

// generate a self-signed cert using certificate information
cert self_signed --CN commonName --O "Example Organization" --C US --L "San Francisco" --OU "IT Department" --ST California --validity 365
~~~`,
	}
	common.BindFlag(selfSignCmd, common.SelfSignedFlagSet)

	acmeCmd := &cobra.Command{
		Use:   consts.CommandCertAcme,
		Short: "obtain an ACME certificate via DNS-01 challenge",
		Long:  "Obtain and store an ACME certificate using a DNS-01 challenge.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return AcmeCmd(cmd, con)
		},
		Example: `~~~
// obtain cert using server config defaults
cert acme --domain *.example.com

// obtain cert with explicit provider
cert acme --domain example.com --provider cloudflare --cred api_token=xxx

// obtain cert using Let's Encrypt staging
cert acme --domain example.com --ca-url https://acme-staging-v02.api.letsencrypt.org/directory
~~~`,
	}
	common.BindFlag(acmeCmd, common.AcmeFlagSet)
	_ = acmeCmd.MarkFlagRequired("domain")

	acmeConfigCmd := &cobra.Command{
		Use:   consts.CommandCertAcmeConfig,
		Short: "view or update ACME configuration",
		Long:  "Show the current ACME defaults or update the supplied configuration fields.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return AcmeConfigCmd(cmd, con)
		},
		Example: `~~~
// view current ACME config
cert acme_config

// set default ACME config
cert acme_config --email admin@example.com --provider cloudflare --cred api_token=xxx

// update only email
cert acme_config --email new@example.com
~~~`,
	}
	common.BindFlag(acmeConfigCmd, common.AcmeConfigFlagSet)

	delCmd := &cobra.Command{
		Use:   consts.CommandCertDelete + " [cert_name]",
		Short: "Delete a certificate",
		Long:  "Delete the specified certificate from the server certificate store.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return DeleteCmd(cmd, con)
		},
		Example: `~~~
// delete a cert
cert delete cert-name
~~~`,
	}
	common.BindArgCompletions(delCmd, nil,
		common.CertNameCompleter(con),
	)

	updateCmd := &cobra.Command{
		Use:   consts.CommandCertUpdate + " [cert_name]",
		Short: "update a cert",
		Long:  "Update stored certificate fields and reload pipelines and websites that reference it. The positional certificate name remains supported for compatibility.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return UpdateCmd(cmd, con)
		},
		Example: `~~~
// update a cert
cert update --cert-name cert-name --cert cert_path --key key_path --type imported
~~~`,
	}

	common.BindFlag(updateCmd, common.ImportSet, func(f *pflag.FlagSet) {
		f.String("cert-name", "", "stored certificate name")
		f.String("type", "", "cert type")
		f.String("comment", "", "certificate comment")
		f.Bool("clear-ca", false, "clear the stored CA certificate")
		f.Bool("no-reload", false, "update the certificate without reloading references")
	})

	common.BindArgCompletions(updateCmd, nil,
		common.CertNameCompleter(con),
	)
	common.BindFlagCompletions(updateCmd, func(comp carapace.ActionMap) {
		comp["cert-name"] = common.CertNameCompleter(con)
		comp["cert"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key"] = carapace.ActionFiles().Usage("path to the key file")
		comp["type"] = common.CertTypeCompleter()
		comp["ca-cert"] = carapace.ActionFiles().Usage("path to the ca cert file")
	})

	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "Reload references to a stored certificate",
		Long:  "Reload every website and supported pipeline that references a stored certificate, optionally filtered by listener and pipeline name.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ApplyCmd(cmd, con)
		},
		Example: `~~~
cert apply --cert-name prod-cert

cert apply --cert-name prod-cert --listener edge-a --pipeline http-main
~~~`,
	}
	common.BindFlag(applyCmd, func(f *pflag.FlagSet) {
		f.String("cert-name", "", "stored certificate name")
		f.String("listener", "", "only reload references on this listener")
		f.String("pipeline", "", "only reload references with this pipeline or website name")
	})
	_ = applyCmd.MarkFlagRequired("cert-name")
	common.BindFlagCompletions(applyCmd, func(comp carapace.ActionMap) {
		comp["cert-name"] = common.CertNameCompleter(con)
		comp["listener"] = common.ListenerIDCompleter(con)
		comp["pipeline"] = common.PipelineNameFlagCompleter(con, applyCmd, consts.WebsitePipeline, consts.HTTPPipeline, consts.TCPPipeline)
	})

	downloadCmd := &cobra.Command{
		Use:   consts.CommandCertDownload + " [cert_name]",
		Short: "download a cert",
		Long:  "Download the specified certificate from the server certificate store.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return DownloadCmd(cmd, con)
		},
		Example: `~~~
// download a cert
cert download cert-name -o cert_path
~~~`,
	}

	common.BindArgCompletions(downloadCmd, nil,
		common.CertNameCompleter(con),
	)

	common.BindFlag(downloadCmd, func(f *pflag.FlagSet) {
		f.StringP("output", "o", "", "cert save path")
	})

	inspectCmd := &cobra.Command{
		Use:   "inspect [cert_name]",
		Short: "Inspect a certificate",
		Args:  cobra.ExactArgs(1),
		Long:  "Show certificate metadata, validity dates, fingerprints, and reference information.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return InspectCmd(cmd, con)
		},
		Example: `~~~
cert inspect ZANY_PIN
~~~`,
	}
	common.BindArgCompletions(inspectCmd, nil, common.CertNameCompleter(con))

	verifyCmd := &cobra.Command{
		Use:   "verify [cert_name]",
		Short: "Verify certificate validity and key pairing",
		Args:  cobra.ExactArgs(1),
		Long:  "Verify that a stored certificate is currently valid and matches its private key.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return VerifyCmd(cmd, con)
		},
		Example: `~~~
cert verify ZANY_PIN
~~~`,
	}
	common.BindArgCompletions(verifyCmd, nil, common.CertNameCompleter(con))

	renewCmd := &cobra.Command{
		Use:   "renew [cert_name]",
		Short: "Renew an ACME certificate",
		Args:  cobra.ExactArgs(1),
		Long:  "Renew a stored ACME certificate, optionally overriding its domain, provider, email, or CA directory.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RenewCmd(cmd, con)
		},
		Example: `~~~
cert renew example-com

cert renew example-com --domain example.com --provider cloudflare
~~~`,
	}
	common.BindFlag(renewCmd, func(f *pflag.FlagSet) {
		f.String("domain", "", "ACME domain override")
		f.String("provider", "", "ACME provider override")
		f.String("email", "", "ACME account email override")
		f.String("ca-url", "", "ACME CA directory URL override")
		f.Bool("no-reload", false, "renew the certificate without reloading references")
	})
	common.BindArgCompletions(renewCmd, nil, common.CertNameCompleter(con))

	listRefsCmd := &cobra.Command{
		Use:   "list-refs [cert_name]",
		Short: "List pipelines and websites referencing a certificate",
		Args:  cobra.ExactArgs(1),
		Long:  "List every pipeline and website that currently references the specified certificate.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListRefsCmd(cmd, con)
		},
		Example: `~~~
cert list-refs ZANY_PIN
~~~`,
	}
	common.BindArgCompletions(listRefsCmd, nil, common.CertNameCompleter(con))

	pruneExpiredCmd := &cobra.Command{
		Use:   "prune",
		Short: "Prune expired certificates",
		Long:  "Delete expired certificates from the certificate store. Pass --expired to confirm the selection criterion.",
		RunE: func(cmd *cobra.Command, args []string) error {
			expiredOnly, _ := cmd.Flags().GetBool("expired")
			if !expiredOnly {
				return cmd.Help()
			}
			return PruneExpiredCmd(cmd, con)
		},
		Example: `~~~
cert prune --expired
~~~`,
	}
	common.BindFlag(pruneExpiredCmd, func(f *pflag.FlagSet) {
		f.Bool("expired", false, "delete expired certificates")
	})
	// Enable wizard for cert commands that need configuration
	common.EnableWizardForCommands(importCmd, selfSignCmd, updateCmd)

	certCmd.AddCommand(importCmd, selfSignCmd, acmeCmd, acmeConfigCmd, delCmd, updateCmd, applyCmd, downloadCmd,
		inspectCmd, verifyCmd, renewCmd, listRefsCmd, pruneExpiredCmd)
	return []*cobra.Command{
		certCmd,
	}
}
