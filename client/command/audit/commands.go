package audit

import (
	"fmt"

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
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return AuditSessionCmd(cmd, con)
			}
			return cmd.Help()
		},
		Example: `~~~
audit session 08d6c05a --output json
audit 08d6c05a --file ./session-audit.json
~~~`,
	}

	sessionCommand := &cobra.Command{
		Use:   consts.CommandSession + " [session_id]",
		Short: "Download a session audit log",
		Long:  "Download the audit log for the specified session.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("session id is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return AuditSessionCmd(cmd, con)
		},
		Example: `~~~
audit session 08d6c05a
audit session 08d6c05a --output html --file ./audit.html
~~~`,
	}

	common.BindArgCompletions(sessionCommand, nil, common.AllSessionIDCompleter(con))
	bindAuditFlags := func(f *pflag.FlagSet) {
		f.StringP("file", "f", "", "log save path")
		f.StringP("output", "o", "json", "log format(json/html)")
	}
	common.BindFlag(auditCommand, bindAuditFlags)
	common.BindFlag(sessionCommand, bindAuditFlags)

	auditCommand.AddCommand(sessionCommand)
	return []*cobra.Command{
		auditCommand,
	}
}
