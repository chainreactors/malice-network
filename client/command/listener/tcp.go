package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"os"
	"strconv"
)

func startTcpPipelineCmd(ctx *grumble.Context, con *console.Console) {
	certPath := ctx.Flags.String("cert_path")
	keyPath := ctx.Flags.String("key_path")
	host := ctx.Flags.String("host")
	port := uint32(ctx.Flags.Int("port"))
	name := ctx.Flags.String("name")
	listenerID := ctx.Flags.String("listener_id")
	var cert, key string
	var err error
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

func stopTcpPipelineCmd(ctx *grumble.Context, con *console.Console) {
	name := ctx.Args.String("name")
	listenerID := ctx.Args.String("listener_id")
	_, err := con.Rpc.StopTcpPipeline(context.Background(), &lispb.TCPPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		console.Log.Error(err.Error())
	}
}

func listTcpPipelines(ctx *grumble.Context, con *console.Console) {
	listenerID := ctx.Args.String("listener_id")
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
	fmt.Printf(tableModel.View(), os.Stdout)
}
