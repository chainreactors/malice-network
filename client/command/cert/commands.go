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
			return GetCmd(cmd, con)
		},
	}

	generateCmd := &cobra.Command{
		Use:   consts.CommandCertGenerate,
		Short: "generate a new cert",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	importCmd := &cobra.Command{
		Use:   consts.CommandCertImport,
		Short: "import a new cert",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ImportCmd(cmd, con)
		},
	}

	common.BindFlag(importCmd, common.ImportSet)
	common.BindFlagCompletions(importCmd, func(comp carapace.ActionMap) {
		comp["cert"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key"] = carapace.ActionFiles().Usage("path to the key file")
		comp["ca-cert"] = carapace.ActionFiles().Usage("path to the ca cert file")
	})

	selfSignCmd := &cobra.Command{
		Use:   consts.CommandCertSelfSigned,
		Short: "generate a self signed cert",
		RunE: func(cmd *cobra.Command, args []string) error {
			return SelfSignedCmd(cmd, con)
		},
	}
	common.BindFlag(selfSignCmd, common.SelfSignedFlagSet)

	acmeCmd := &cobra.Command{
		Use:   consts.CommandCertSelfSigned,
		Short: "generate a acme cert",
		RunE: func(cmd *cobra.Command, args []string) error {
			return AcmeCmd(cmd, con)
		},
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

	generateCmd.AddCommand(importCmd, selfSignCmd, acmeCmd)

	delCmd := &cobra.Command{
		Use:   consts.CommandCertDelete,
		Short: "del a cert",
		RunE: func(cmd *cobra.Command, args []string) error {
			return DeleteCmd(cmd, con)
		},
	}

	common.BindFlag(delCmd, common.TlsCertFlagSet)
	delCmd.MarkFlagRequired("cert-name")

	updateCmd := &cobra.Command{
		Use:   consts.CommandCertUpdate,
		Short: "update a cert",
		RunE: func(cmd *cobra.Command, args []string) error {
			return UpdateCmd(cmd, con)
		},
	}

	common.BindFlag(updateCmd, common.TlsCertFlagSet)
	updateCmd.MarkFlagRequired("cert-name")
	common.BindFlagCompletions(updateCmd, func(comp carapace.ActionMap) {
		comp["cert"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key"] = carapace.ActionFiles().Usage("path to the key file")
		comp["cert-name"] = common.CertNameCompleter(con)
	})

	certCmd.AddCommand(generateCmd, delCmd, updateCmd)
	return []*cobra.Command{
		certCmd,
	}
}
