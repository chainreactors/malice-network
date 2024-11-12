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

func ListJobsCmd(cmd *cobra.Command, con *repl.Console) error {
	Pipelines, err := con.Rpc.ListJobs(context.Background(), &clientpb.Empty{})
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
	return nil
}

func ListPipelineCmd(cmd *cobra.Command, con *repl.Console) error {
	listenerID, _ := cmd.Flags().GetString("listener")
	pipelines, err := con.Rpc.ListPipelines(context.Background(), &clientpb.Listener{
		Id: listenerID,
	})
	if err != nil {
		return err
	}
	if len(pipelines.Pipelines) == 0 {
		con.Log.Warnf("No pipelines found")
		return nil
	}
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		{Title: "Name", Width: 20},
		{Title: "Type", Width: 10},
		{Title: "ListenerID", Width: 15},
		{Title: "Host", Width: 10},
		{Title: "Port", Width: 7},
		{Title: "Enable", Width: 7},
	}, true)
	for _, pipeline := range pipelines.GetPipelines() {
		switch body := pipeline.Body.(type) {
		case *clientpb.Pipeline_Tcp:
			row = table.Row{
				pipeline.Name,
				consts.TCPPipeline,
				pipeline.ListenerId,
				body.Tcp.Host,
				strconv.Itoa(int(body.Tcp.Port)),
				strconv.FormatBool(pipeline.Enable),
			}
		case *clientpb.Pipeline_Bind:
			row = table.Row{
				pipeline.Name,
				consts.BindPipeline,
				pipeline.ListenerId,
				"",
				"",
				strconv.FormatBool(pipeline.Enable),
			}
		}

		rowEntries = append(rowEntries, row)
	}
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
	return nil
}

func StartPipelineCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	_, err := con.Rpc.StartPipeline(context.Background(), &clientpb.CtrlPipeline{
		Name: name,
	})
	if err != nil {
		return err
	}
	return nil
}

func StopPipelineCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	_, err := con.Rpc.StopPipeline(context.Background(), &clientpb.CtrlPipeline{
		Name: name,
	})
	if err != nil {
		return err
	}
	return nil
}
