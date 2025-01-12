package listener

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"strconv"
)

func ListJobsCmd(cmd *cobra.Command, con *repl.Console) error {
	Pipelines, err := con.Rpc.ListJobs(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	if len(Pipelines.GetPipelines()) == 0 {
		con.Log.Importantf("No jobs found")
		return nil
	}
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("Name", "Name", 20),
		table.NewColumn("Listener_id", "Listener_id", 15),
		table.NewColumn("IP", "IP", 10),
		table.NewColumn("Port", "Port", 7),
		table.NewColumn("Type", "Type", 7),
	}, true)
	for _, pipeline := range Pipelines.GetPipelines() {
		switch pipeline.Body.(type) {
		case *clientpb.Pipeline_Tcp:
			tcp := pipeline.GetTcp()
			row = table.NewRow(
				table.RowData{
					"Name":        pipeline.Name,
					"Listener_id": pipeline.ListenerId,
					"IP":          pipeline.Ip,
					"Port":        strconv.Itoa(int(tcp.Port)),
					"Type":        "TCP",
				})
		case *clientpb.Pipeline_Bind:
			//bind := pipeline.GetBind()
			row = table.NewRow(
				table.RowData{
					"Name":        pipeline.Name,
					"Listener_id": pipeline.ListenerId,
					"IP":          "",
					"Port":        "",
					"Type":        "Bind",
				})

		case *clientpb.Pipeline_Web:
			website := pipeline.GetWeb()
			row = table.NewRow(
				table.RowData{
					"Name":        pipeline.Name,
					"Listener_id": pipeline.ListenerId,
					"IP":          pipeline.Ip,
					"Port":        strconv.Itoa(int(website.Port)),
					"Type":        "Web",
				})

		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
	return nil
}
