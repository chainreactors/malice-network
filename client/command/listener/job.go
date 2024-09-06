package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"strconv"
)

func listJobsCmd(cmd *cobra.Command, con *repl.Console) {
	listenerID := cmd.Flags().Arg(0)
	if listenerID == "" {
		repl.Log.Error("listener_id is required")
		return
	}
	Pipelines, err := con.Rpc.ListJobs(context.Background(), &lispb.ListenerName{
		Name: listenerID,
	})
	if err != nil {
		repl.Log.Error(err.Error())
		return
	}
	if len(Pipelines.GetPipelines()) == 0 {
		repl.Log.Importantf("No jobs found")
		return
	}
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		{Title: "Name", Width: 20},
		{Title: "Host", Width: 10},
		{Title: "Port", Width: 7},
		{Title: "Type", Width: 7},
	}, true)
	for _, Pipeline := range Pipelines.GetPipelines() {
		switch Pipeline.Body.(type) {
		case *lispb.Pipeline_Tcp:
			tcp := Pipeline.GetTcp()
			row = table.Row{
				tcp.Name,
				tcp.Host,
				strconv.Itoa(int(tcp.Port)),
				"TCP",
			}
		case *lispb.Pipeline_Web:
			website := Pipeline.GetWeb()
			row = table.Row{
				website.Name,
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
