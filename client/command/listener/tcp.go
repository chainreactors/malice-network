package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"log"
	"strconv"
)

func startTcpPipelineCmd(cmd *cobra.Command, con *console.Console) {
	certPath, _ := cmd.Flags().GetString("cert_path")
	keyPath, _ := cmd.Flags().GetString("key_path")
	name := cmd.Flags().Arg(0)
	listenerID := cmd.Flags().Arg(1)
	host := cmd.Flags().Arg(2)
	portStr := cmd.Flags().Arg(3)
	portUint, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		log.Fatalf("Invalid port number: %v", err)
	}
	port := uint32(portUint)
	var cert, key string
	if certPath != "" && keyPath != "" {
		cert, err = cryptography.ProcessPEM(certPath)
		if err != nil {
			console.Log.Error(err.Error())
			return
		}
		key, err = cryptography.ProcessPEM(keyPath)
		if err != nil {
			console.Log.Error(err.Error())
			return
		}
	}
	_, err = con.Rpc.StartTcpPipeline(context.Background(), &lispb.Pipeline{
		Tls: &lispb.TLS{
			Cert: cert,
			Key:  key,
		},
		Body: &lispb.Pipeline_Tcp{
			Tcp: &lispb.TCPPipeline{
				Host:       host,
				Port:       port,
				Name:       name,
				ListenerId: listenerID,
			},
		},
	})

	if err != nil {
		console.Log.Error(err.Error())
	}
}

func stopTcpPipelineCmd(cmd *cobra.Command, con *console.Console) {
	name := cmd.Flags().Arg(0)
	listenerID := cmd.Flags().Arg(1)
	_, err := con.Rpc.StopTcpPipeline(context.Background(), &lispb.TCPPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		console.Log.Error(err.Error())
	}
}

func listTcpPipelines(cmd *cobra.Command, con *console.Console) {
	listenerID := cmd.Flags().Arg(0)
	if listenerID == "" {
		console.Log.Error("listener_id is required")
		return
	}
	Pipelines, err := con.Rpc.ListPipelines(context.Background(), &lispb.ListenerName{
		Name: listenerID,
	})
	if err != nil {
		console.Log.Error(err.Error())
		return
	}
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		{Title: "Name", Width: 10},
		{Title: "Host", Width: 10},
		{Title: "Port", Width: 7},
	}, true)
	for _, Pipeline := range Pipelines.GetPipelines() {
		tcp := Pipeline.GetTcp()
		row = table.Row{
			tcp.Name,
			tcp.Host,
			strconv.Itoa(int(tcp.Port)),
		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
}
