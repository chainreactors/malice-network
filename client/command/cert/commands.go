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
		RunE: func(cmd *cobra.Command, args []string) error {
			return ImportCmd(cmd, con)
		},
		Example: `~~~
// generate a imported cert to server
cert import --cert cert_file_path --key key_file_path --ca-cert ca_cert_path
~~~`,
	}

	common.BindFlag(importCmd, common.ImportSet)
	_ = importCmd.MarkFlagRequired("cert")
	_ = importCmd.MarkFlagRequired("key")
	common.BindFlagCompletions(importCmd, func(comp carapace.ActionMap) {
		comp["cert"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key"] = carapace.ActionFiles().Usage("path to the key file")
		comp["ca-cert"] = carapace.ActionFiles().Usage("path to the ca cert file")
	})

	selfSignCmd := &cobra.Command{
		Use:   consts.CommandCertSelfSigned,
		Short: "generate a self-signed cert",
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
		Use:  consts.CommandCertDelete,
		Args: cobra.ExactArgs(1),
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
		Use:   consts.CommandCertUpdate,
		Short: "update a cert",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return UpdateCmd(cmd, con)
		},
		Example: `~~~
// update a cert
cert update cert-name --cert cert_path --key key_path --type imported
~~~`,
	}

	common.BindFlag(updateCmd, common.ImportSet, func(f *pflag.FlagSet) {
		f.String("type", "", "cert type")
	})

	common.BindArgCompletions(updateCmd, nil,
		common.CertNameCompleter(con),
	)
	common.BindFlagCompletions(updateCmd, func(comp carapace.ActionMap) {
		comp["cert"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key"] = carapace.ActionFiles().Usage("path to the key file")
		comp["type"] = common.CertTypeCompleter()
		comp["ca-cert"] = carapace.ActionFiles().Usage("path to the ca cert file")
	})

	downloadCmd := &cobra.Command{
		Use:   consts.CommandCertDownload,
		Short: "download a cert",
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
	// Enable wizard for cert commands that need configuration
	common.EnableWizardForCommands(importCmd, selfSignCmd, updateCmd)

	certCmd.AddCommand(importCmd, selfSignCmd, acmeCmd, acmeConfigCmd, delCmd, updateCmd, downloadCmd)
	return []*cobra.Command{
		certCmd,
	}
}
