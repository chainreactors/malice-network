package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
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
	for _, pipeline := range Pipelines.GetPipelines() {
		switch pipeline.Body.(type) {
		case *clientpb.Pipeline_Tcp:
			tcp := pipeline.GetTcp()
			row = table.Row{
				pipeline.Name,
				pipeline.ListenerId,
				tcp.Host,
				strconv.Itoa(int(tcp.Port)),
				"TCP",
			}
		case *clientpb.Pipeline_Web:
			website := pipeline.GetWeb()
			row = table.Row{
				pipeline.Name,
				pipeline.ListenerId,
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

func listPipelineCmd(cmd *cobra.Command, con *repl.Console) {
	listenerID := cmd.Flags().Arg(0)
	if listenerID == "" {
		con.Log.Error("listener_id is required")
		return
	}
	Pipelines, err := con.LisRpc.ListPipelines(context.Background(), &clientpb.ListenerName{
		Name: listenerID,
	})
	if err != nil {
		con.Log.Error(err.Error())
		return
	}
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		{Title: "Name", Width: 20},
		{Title: "Type", Width: 10},
		{Title: "Host", Width: 10},
		{Title: "Port", Width: 7},
		{Title: "Enable", Width: 7},
	}, true)
	for _, pipeline := range Pipelines.GetPipelines() {
		switch body := pipeline.Body.(type) {
		case *clientpb.Pipeline_Tcp:
			row = table.Row{
				pipeline.Name,
				consts.TCPPipeline,
				body.Tcp.Host,
				strconv.Itoa(int(body.Tcp.Port)),
				strconv.FormatBool(pipeline.Enable),
			}
		case *clientpb.Pipeline_Bind:
			row = table.Row{
				pipeline.Name,
				consts.BindPipeline,
				"",
				"",
				strconv.FormatBool(pipeline.Enable),
			}
		}

		rowEntries = append(rowEntries, row)
	}
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
}

func startPipelineCmd(cmd *cobra.Command, con *repl.Console) {
	name := cmd.Flags().Arg(0)
	listenerID := cmd.Flags().Arg(1)
	_, err := con.LisRpc.StartPipeline(context.Background(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})

	if err != nil {
		con.Log.Error(err.Error())
	}
}

func stopPipelineCmd(cmd *cobra.Command, con *repl.Console) {
	name := cmd.Flags().Arg(1)
	listenerID := cmd.Flags().Arg(0)
	_, err := con.LisRpc.StopPipeline(context.Background(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		con.Log.Error(err.Error())
	}
}
