package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"math/rand"
	"strconv"
	"time"
)

func listTcpCmd(cmd *cobra.Command, con *repl.Console) {
	listenerID := cmd.Flags().Arg(0)
	if listenerID == "" {
		con.Log.Error("listener_id is required")
		return
	}
	Pipelines, err := con.LisRpc.ListTcpPipelines(context.Background(), &lispb.ListenerName{
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
	name, host, portUint, certPath, keyPath, tlsEnable := common.ParsePipelineSet(cmd)
	listenerID := cmd.Flags().Arg(0)
	var cert, key string
	var err error
	if portUint == 0 {
		rand.Seed(time.Now().UnixNano())
		portUint = uint(10000 + rand.Int31n(5001))
	}
	port := uint32(portUint)
	if host == "" {
		host = "0.0.0.0"
	}
	if name == "" {
		name = fmt.Sprintf("%s-tcp-%d", listenerID, port)
	}
	if certPath != "" && keyPath != "" {
		cert, err = cryptography.ProcessPEM(certPath)
		if err != nil {
			con.Log.Error(err.Error())
			return
		}
		key, err = cryptography.ProcessPEM(keyPath)
		tlsEnable = true
		if err != nil {
			con.Log.Error(err.Error())
			return
		}
	}
	_, err = con.LisRpc.RegisterPipeline(context.Background(), &lispb.Pipeline{
		Encryption: &lispb.Encryption{
			Enable: false,
			Type:   "",
			Key:    "",
		},
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
		con.Log.Error(err.Error())
	}

	_, err = con.LisRpc.StartTcpPipeline(context.Background(), &lispb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		con.Log.Error(err.Error())
	}
	con.Log.Importantf("TCP Pipeline %s added\n", name)
}

func startTcpPipelineCmd(cmd *cobra.Command, con *repl.Console) {
	name := cmd.Flags().Arg(0)
	listenerID := cmd.Flags().Arg(1)
	_, err := con.LisRpc.StartTcpPipeline(context.Background(), &lispb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})

	if err != nil {
		con.Log.Error(err.Error())
	}
}

func stopTcpPipelineCmd(cmd *cobra.Command, con *repl.Console) {
	name := cmd.Flags().Arg(0)
	listenerID := cmd.Flags().Arg(1)
	_, err := con.LisRpc.StopTcpPipeline(context.Background(), &lispb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		con.Log.Error(err.Error())
	}
}
