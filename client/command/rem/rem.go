package rem

import (
	"fmt"
	"strconv"

	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func ListRemCmd(cmd *cobra.Command, con *repl.Console) error {
	listenerID := cmd.Flags().Arg(0)
	rems, err := con.Rpc.ListRems(con.Context(), &clientpb.Listener{
		Id: listenerID,
	})
	if err != nil {
		return err
	}
	if len(rems.Pipelines) == 0 {
		con.Log.Warnf("No REMs found")
		return nil
	}

	var rowEntries []table.Row
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("Name", "Name", 20),
		table.NewColumn("Enable", "Enable", 7),
		table.NewColumn("ListenerID", "ListenerID", 15),
		table.NewColumn("Console", "Console", 30),
	}, true)

	for _, rem := range rems.GetPipelines() {
		newRow := table.RowData{}
		newRow["Name"] = rem.Name
		if rem.Enable {
			newRow["Enable"] = tui.GreenFg.Render(strconv.FormatBool(rem.Enable))
		} else {
			newRow["Enable"] = tui.RedFg.Render(strconv.FormatBool(rem.Enable))
		}
		newRow["ListenerID"] = rem.ListenerId
		newRow["Console"] = rem.GetRem().Console

		rowEntries = append(rowEntries, table.NewRow(newRow))
	}
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
	return nil
}

func NewRemCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	listenerID, _ := cmd.Flags().GetString("listener")
	console, _ := cmd.Flags().GetString("console")

	if name == "" {
		name = fmt.Sprintf("%s_rem_%s", listenerID, console)
	}

	pipeline := &clientpb.Pipeline{
		Name:       name,
		ListenerId: listenerID,
		Enable:     true,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Console: console,
			},
		},
	}

	_, err := con.Rpc.RegisterRem(con.Context(), pipeline)
	if err != nil {
		return err
	}

	_, err = con.Rpc.StartRem(con.Context(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		return err
	}

	return nil
}

func StartRemCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	_, err := con.Rpc.StartRem(con.Context(), &clientpb.CtrlPipeline{
		Name: name,
	})
	if err != nil {
		return err
	}
	return nil
}

func StopRemCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	_, err := con.Rpc.StopRem(con.Context(), &clientpb.CtrlPipeline{
		Name: name,
	})
	if err != nil {
		return err
	}
	return nil
}

func DeleteRemCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	_, err := con.Rpc.DeleteRem(con.Context(), &clientpb.CtrlPipeline{
		Name: name,
	})
	if err != nil {
		return err
	}
	return nil
}
