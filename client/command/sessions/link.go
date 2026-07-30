package sessions

import (
	"fmt"
	"time"

	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newSessionLinkCommand(con *core.Console) *cobra.Command {
	linkCommand := &cobra.Command{
		Use:   "link",
		Short: "Manage session parent-child relationships",
		Long:  "List and manage manually assigned parent-child relationships between sessions.",
		Example: `~~~
session link
session link --parent 08d6c05a21512a79a1dfeb9d2a8f262f
~~~`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return listSessionLinksCmd(cmd, con)
		},
	}
	bindSessionLinkFilterFlags(linkCommand, con)

	listCommand := &cobra.Command{
		Use:   "list",
		Short: "List session parent-child relationships",
		Long:  "List all session links, or filter them by parent and child session IDs.",
		Example: `~~~
session link list
session link list --child 08d6c05a21512a79a1dfeb9d2a8f262f
~~~`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return listSessionLinksCmd(cmd, con)
		},
	}
	bindSessionLinkFilterFlags(listCommand, con)

	setCommand := &cobra.Command{
		Use:     "set",
		Aliases: []string{"reparent"},
		Short:   "Set or replace a session parent",
		Long:    "Assign a parent to a session. If the child already has a parent, replace the existing relationship.",
		Example: `~~~
session link set --parent 08d6c05a21512a79a1dfeb9d2a8f262f --child b2bc23d0325a476ea01308d93f15f9da
session link reparent --parent c459870f1f854653a81407d7018ee756 --child b2bc23d0325a476ea01308d93f15f9da
~~~`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return setSessionLinkCmd(cmd, con)
		},
	}
	common.BindFlag(setCommand, func(flags *pflag.FlagSet) {
		flags.String("parent", "", "parent session id")
		flags.String("child", "", "child session id")
	})
	_ = setCommand.MarkFlagRequired("parent")
	_ = setCommand.MarkFlagRequired("child")
	bindSessionLinkFlagCompletions(setCommand, con)

	unlinkCommand := &cobra.Command{
		Use:   "unlink",
		Short: "Remove a session parent relationship",
		Long:  "Detach a child session from its current parent while preserving the child's own descendants.",
		Example: `~~~
session link unlink --child b2bc23d0325a476ea01308d93f15f9da
~~~`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return removeSessionLinkCmd(cmd, con)
		},
	}
	common.BindFlag(unlinkCommand, func(flags *pflag.FlagSet) {
		flags.String("child", "", "child session id")
	})
	_ = unlinkCommand.MarkFlagRequired("child")
	common.BindFlagCompletions(unlinkCommand, func(comp carapace.ActionMap) {
		comp["child"] = common.AllSessionIDCompleter(con)
	})

	linkCommand.AddCommand(listCommand, setCommand, unlinkCommand)
	return linkCommand
}

func bindSessionLinkFilterFlags(cmd *cobra.Command, con *core.Console) {
	common.BindFlag(cmd, func(flags *pflag.FlagSet) {
		flags.String("parent", "", "filter by parent session id")
		flags.String("child", "", "filter by child session id")
	})
	bindSessionLinkFlagCompletions(cmd, con)
}

func bindSessionLinkFlagCompletions(cmd *cobra.Command, con *core.Console) {
	common.BindFlagCompletions(cmd, func(comp carapace.ActionMap) {
		comp["parent"] = common.AllSessionIDCompleter(con)
		comp["child"] = common.AllSessionIDCompleter(con)
	})
}

func listSessionLinksCmd(cmd *cobra.Command, con *core.Console) error {
	parentSessionID, err := resolveSessionLinkFlag(con, cmd, "parent")
	if err != nil {
		return err
	}
	childSessionID, err := resolveSessionLinkFlag(con, cmd, "child")
	if err != nil {
		return err
	}

	links, err := con.Rpc.ListSessionLinks(con.Context(), &clientpb.SessionLinkRequest{
		ParentSessionId: parentSessionID,
		ChildSessionId:  childSessionID,
	})
	if err != nil {
		return err
	}
	if len(links.GetLinks()) == 0 {
		con.Log.Console("No session links found\n")
		return nil
	}
	printSessionLinks(con, links.GetLinks())
	return nil
}

func setSessionLinkCmd(cmd *cobra.Command, con *core.Console) error {
	parentSessionID, err := resolveSessionLinkFlag(con, cmd, "parent")
	if err != nil {
		return err
	}
	childSessionID, err := resolveSessionLinkFlag(con, cmd, "child")
	if err != nil {
		return err
	}

	link, err := con.Rpc.SetSessionLink(con.Context(), &clientpb.SessionLinkRequest{
		ParentSessionId: parentSessionID,
		ChildSessionId:  childSessionID,
	})
	if err != nil {
		return err
	}
	con.Log.Console(fmt.Sprintf("Session %s now uses %s as its parent\n", link.GetChildSessionId(), link.GetParentSessionId()))
	return nil
}

func removeSessionLinkCmd(cmd *cobra.Command, con *core.Console) error {
	childSessionID, err := resolveSessionLinkFlag(con, cmd, "child")
	if err != nil {
		return err
	}
	if _, err := con.Rpc.RemoveSessionLink(con.Context(), &clientpb.SessionLinkRequest{ChildSessionId: childSessionID}); err != nil {
		return err
	}
	con.Log.Console(fmt.Sprintf("Session %s detached from its parent\n", childSessionID))
	return nil
}

func resolveSessionLinkFlag(con *core.Console, cmd *cobra.Command, name string) (string, error) {
	sessionID, err := cmd.Flags().GetString(name)
	if err != nil || sessionID == "" {
		return sessionID, err
	}
	return resolveSessionID(con, sessionID)
}

func printSessionLinks(con *core.Console, links []*clientpb.SessionLink) {
	rows := make([]table.Row, 0, len(links))
	for _, link := range links {
		if link == nil {
			continue
		}
		updatedAt := "-"
		if link.GetUpdatedAt() > 0 {
			updatedAt = time.Unix(link.GetUpdatedAt(), 0).Format("2006-01-02 15:04:05")
		}
		rows = append(rows, table.NewRow(table.RowData{
			"Parent":  link.GetParentSessionId(),
			"Child":   link.GetChildSessionId(),
			"Source":  link.GetSource(),
			"Updated": updatedAt,
		}))
	}

	tableModel := tui.NewTable([]table.Column{
		table.NewFlexColumn("Parent", "Parent", 1),
		table.NewFlexColumn("Child", "Child", 1),
		table.NewColumn("Source", "Source", 10),
		table.NewColumn("Updated", "Updated", 19),
	}, true)
	tableModel.SetMultiline()
	tableModel.SetRows(rows)
	tableModel.Title = "session links"
	con.Log.Console(tableModel.View())
}
