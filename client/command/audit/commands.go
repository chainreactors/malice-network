package audit

import (
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *core.Console) []*cobra.Command {
	auditCommand := &cobra.Command{
		Use:   consts.CommandAudit,
		Short: "Manage audit logs",
		Long:  "Download audit logs for server sessions.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	sessionCommand := &cobra.Command{
		Use:   consts.CommandSession,
		Short: "Download a session audit log",
		Long:  "Download the audit log for the specified session.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return AuditSessionCmd(cmd, con)
		},
	}

	common.BindArgCompletions(sessionCommand, nil, common.AllSessionIDCompleter(con))
	common.BindFlag(sessionCommand, func(f *pflag.FlagSet) {
		f.StringP("file", "f", "", "log save path")
		f.StringP("output", "o", "json", "log format(json/html)")
	})

	auditCommand.AddCommand(sessionCommand)
	return []*cobra.Command{
		auditCommand,
	}
}
