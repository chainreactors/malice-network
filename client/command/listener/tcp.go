package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"log"
	"strconv"
)

func listTcpCmd(cmd *cobra.Command, con *repl.Console) {
	listenerID := cmd.Flags().Arg(0)
	if listenerID == "" {
		repl.Log.Error("listener_id is required")
		return
	}
	Pipelines, err := con.Rpc.ListTcpPipelines(context.Background(), &lispb.ListenerName{
		Name: listenerID,
	})
	if err != nil {
		repl.Log.Error(err.Error())
		return
	}
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		{Title: "Name", Width: 20},
		{Title: "Host", Width: 10},
		{Title: "Port", Width: 7},
		{Title: "Enable", Width: 7},
	}, true)
	for _, Pipeline := range Pipelines.GetPipelines() {
		tcp := Pipeline.GetTcp()
		row = table.Row{
			tcp.Name,
			tcp.Host,
			strconv.Itoa(int(tcp.Port)),
			strconv.FormatBool(tcp.Enable),
		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
}

func newTcpPipelineCmd(cmd *cobra.Command, con *repl.Console) {
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
	var tlsEnable = false
	if certPath != "" && keyPath != "" {
		cert, err = cryptography.ProcessPEM(certPath)
		if err != nil {
			repl.Log.Error(err.Error())
			return
		}
		key, err = cryptography.ProcessPEM(keyPath)
		tlsEnable = true
		if err != nil {
			repl.Log.Error(err.Error())
			return
		}
	}
	_, err = con.Rpc.NewPipeline(context.Background(), &lispb.Pipeline{
		Tls: &lispb.TLS{
			Cert:   cert,
			Key:    key,
			Enable: tlsEnable,
		},
		Body: &lispb.Pipeline_Tcp{
			Tcp: &lispb.TCPPipeline{
				Host:       host,
				Port:       port,
				Name:       name,
				ListenerId: listenerID,
				Enable:     false,
			},
		},
	})
	if err != nil {
		repl.Log.Error(err.Error())
	}
}

func startTcpPipelineCmd(cmd *cobra.Command, con *repl.Console) {
	name := cmd.Flags().Arg(0)
	listenerID := cmd.Flags().Arg(1)
	_, err := con.Rpc.StartTcpPipeline(context.Background(), &lispb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})

	if err != nil {
		repl.Log.Error(err.Error())
	}
}

func stopTcpPipelineCmd(cmd *cobra.Command, con *repl.Console) {
	name := cmd.Flags().Arg(0)
	listenerID := cmd.Flags().Arg(1)
	_, err := con.Rpc.StopTcpPipeline(context.Background(), &lispb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		repl.Log.Error(err.Error())
	}
}
