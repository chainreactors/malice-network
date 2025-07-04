package cert

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
)

func Commands(con *repl.Console) []*cobra.Command {
	certCmd := &cobra.Command{
		Use:   consts.CommandCert,
		Short: "Cert list",
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetCmd(cmd, con)
		},
	}

	addCmd := &cobra.Command{
		Use:   consts.CommandCertAdd,
		Short: "add a new cert",
		RunE: func(cmd *cobra.Command, args []string) error {
			return AddCmd(cmd, con)
		},
	}

	common.BindFlag(addCmd, common.TlsCertFlagSet)
	common.BindFlagCompletions(addCmd, func(comp carapace.ActionMap) {
		comp["cert"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key"] = carapace.ActionFiles().Usage("path to the key file")
		comp["cert-name"] = common.CertNameCompleter(con)
	})

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

	certCmd.AddCommand(addCmd, delCmd, updateCmd)
	return []*cobra.Command{
		certCmd,
	}
}
