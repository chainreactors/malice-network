package generic

import (
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func ListPivotCmd(cmd *cobra.Command, con *core.Console) error {
	all, _ := cmd.Flags().GetBool("all")
	pivots, err := con.Rpc.GetContexts(con.Context(), &clientpb.Context{
		Type: consts.ContextPivoting,
	})
	if err != nil {
		return err
	}

	if len(pivots.Contexts) == 0 {
		logs.Log.Info("No pivots\n")
		return nil
	}

	PrintPivots(pivots.Contexts, con, all)
	return nil
}

func ListPivot(con *core.Console) ([]*output.PivotingContext, error) {
	pivots, err := con.Rpc.GetContexts(con.Context(), &clientpb.Context{
		Type: consts.ContextPivoting,
	})
	if err != nil {
		return nil, err
	}
	ctxs, err := output.ToContexts[*output.PivotingContext](pivots.Contexts)
	return ctxs, nil
}

func PrintPivots(contexts []*clientpb.Context, con *core.Console, all bool) {
	var rowEntries []table.Row
	for _, ctx := range contexts {
		pivot, err := output.ToContext[*output.PivotingContext](ctx)
		if err != nil {
			continue
		}

		sessionID := ""
		if ctx.Session != nil {
			sessionID = ctx.Session.SessionId
		}

		row := table.NewRow(
			table.RowData{
				"Session":    sessionID,
				"Enable":     fmt.Sprintf("%t", pivot.Enable),
				"Listener":   pivot.Listener,
				"Pipeline":   pivot.Pipeline,
				"RemAgentID": pivot.RemAgentID,
				"LocalURL":   pivot.LocalURL,
				"RemoteURL":  pivot.RemoteURL,
				"Mod":        pivot.Mod,
			})
		if all || pivot.Enable {
			rowEntries = append(rowEntries, row)
		}
	}

	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("Session", "Session", 10),
		table.NewColumn("Enable", "Enable", 6),
		table.NewColumn("Listener", "Listener", 10),
		table.NewColumn("Pipeline", "Pipeline", 10),
		table.NewColumn("RemAgentID", "Rem Agent ID", 10),
		table.NewFlexColumn("LocalURL", "Local URL", 1),
		table.NewFlexColumn("RemoteURL", "Remote URL", 1),
		table.NewColumn("Mod", "Mod", 10),
	}, true)

	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	con.Log.Console(tableModel.View())
}
