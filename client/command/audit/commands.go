package audit

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	auditCommand := &cobra.Command{
		Use:   consts.CommandAudit,
		Short: "audit func",
		Long:  "audit func .",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	sessionCommand := &cobra.Command{
		Use:   consts.CommandSession,
		Short: "download audit log",
		Long:  "Download specified session audit .",
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
