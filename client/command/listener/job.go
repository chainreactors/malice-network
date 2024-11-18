package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
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
		table.NewColumn("Name", "Name", 20),
		table.NewColumn("Listener_id", "Listener_id", 15),
		table.NewColumn("Host", "Host", 10),
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
					"Host":        tcp.Host,
					"Port":        strconv.Itoa(int(tcp.Port)),
					"Type":        "TCP",
				})
		case *clientpb.Pipeline_Web:
			website := pipeline.GetWeb()
			row = table.NewRow(
				table.RowData{
					"Name":        pipeline.Name,
					"Listener_id": pipeline.ListenerId,
					"Host":        "",
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
		table.NewColumn("Name", "Name", 20),
		table.NewColumn("Type", "Type", 10),
		table.NewColumn("ListenerID", "ListenerID", 15),
		table.NewColumn("Host", "Host", 10),
		table.NewColumn("Port", "Port", 7),
		table.NewColumn("Enable", "Enable", 7),
	}, true)
	for _, pipeline := range pipelines.GetPipelines() {
		switch body := pipeline.Body.(type) {
		case *clientpb.Pipeline_Tcp:
			row = table.NewRow(
				table.RowData{
					"Name":       pipeline.Name,
					"Type":       consts.TCPPipeline,
					"ListenerID": pipeline.ListenerId,
					"Host":       body.Tcp.Host,
					"Port":       strconv.Itoa(int(body.Tcp.Port)),
					"Enable":     strconv.FormatBool(pipeline.Enable),
				})
		case *clientpb.Pipeline_Bind:
			row = table.NewRow(
				table.RowData{
					"Name":       pipeline.Name,
					"Type":       consts.BindPipeline,
					"ListenerID": pipeline.ListenerId,
					"Host":       "",
					"Port":       "",
					"Enable":     strconv.FormatBool(pipeline.Enable),
				})
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
