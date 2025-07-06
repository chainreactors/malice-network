package cert

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
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
cert selfSign

// generate a self-signed cert using certificate information
cert selfSign --CN commonName --O "Example Organization" --C US --L "San Francisco" --OU "IT Department" --ST California --validity 365
~~~`,
	}
	common.BindFlag(selfSignCmd, common.SelfSignedFlagSet)

	acmeCmd := &cobra.Command{
		Use:   consts.CommandCertAcme,
		Short: "generate a acme cert",
		RunE: func(cmd *cobra.Command, args []string) error {
			return AcmeCmd(cmd, con)
		},
		Example: `~~~
// generate a acme cert
cert acme --domain *.example.com --pipeline http
~~~`,
	}
	common.BindFlag(acmeCmd, func(f *pflag.FlagSet) {
		f.String("domain", "", "acme domain")
		f.String("pipeline", "", "pipeline name")
	})

	acmeCmd.MarkFlagRequired("domain")
	acmeCmd.MarkFlagRequired("pipeline")

	common.BindFlagCompletions(acmeCmd, func(comp carapace.ActionMap) {
		comp["pipeline"] = common.HttpPipelineCompleter(con)
	})

	delCmd := &cobra.Command{
		Use:  consts.CommandCertDelete,
		Args: cobra.MaximumNArgs(1),
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
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return UpdateCmd(cmd, con)
		},
		Example: `~~~
// update a cert
cert update cert-name --cert cert_path --key key_path --type imported
~~~`,
	}

	common.BindFlag(updateCmd, func(f *pflag.FlagSet) {
		f.String("cert", "", "tls cert path")
		f.String("key", "", "tls key path")
		f.String("type", "", "cert type")
	})

	common.BindArgCompletions(updateCmd, nil,
		common.CertNameCompleter(con),
	)
	common.BindFlagCompletions(updateCmd, func(comp carapace.ActionMap) {
		comp["cert"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key"] = carapace.ActionFiles().Usage("path to the key file")
		comp["type"] = common.CertTypeCompleter()
	})

	downloadCmd := &cobra.Command{
		Use:   consts.CommandCertDownload,
		Short: "download a cert",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return DownloadCmd(cmd, con)
		},
		Example: `~~~
// download a cert
cert download cert-name -o cert_path 
~~~`,
	}

	common.BindArgCompletions(updateCmd, nil,
		common.CertNameCompleter(con),
	)

	common.BindFlag(downloadCmd, func(f *pflag.FlagSet) {
		f.StringP("output", "o", "", "cert save path")
	})

	certCmd.AddCommand(importCmd, selfSignCmd, acmeCmd, delCmd, updateCmd)
	return []*cobra.Command{
		certCmd,
	}
}
