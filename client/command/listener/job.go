package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"strconv"
)

func listJobsCmd(cmd *cobra.Command, con *repl.Console) {
	Pipelines, err := con.Rpc.ListJobs(context.Background(), &clientpb.Empty{})
	if err != nil {
		con.Log.Error(err.Error())
		return
	}
	if len(Pipelines.GetPipelines()) == 0 {
		con.Log.Importantf("No jobs found")
		return
	}
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		{Title: "Name", Width: 20},
		{Title: "Listener_id", Width: 15},
		{Title: "Host", Width: 10},
		{Title: "Port", Width: 7},
		{Title: "Type", Width: 7},
	}, true)
	for _, Pipeline := range Pipelines.GetPipelines() {
		switch Pipeline.Body.(type) {
		case *clientpb.Pipeline_Tcp:
			tcp := Pipeline.GetTcp()
			row = table.Row{
				tcp.Name,
				tcp.ListenerId,
				tcp.Host,
				strconv.Itoa(int(tcp.Port)),
				"TCP",
			}
		case *clientpb.Pipeline_Web:
			website := Pipeline.GetWeb()
			row = table.Row{
				website.Name,
				website.ListenerId,
				"",
				strconv.Itoa(int(website.Port)),
				"Web",
			}

		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
}
