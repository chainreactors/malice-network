package listener

import (
	"strconv"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func ForwardConnectCmd(cmd *cobra.Command, con *core.Console) error {
	host, err := cmd.Flags().GetString("host")
	if err != nil {
		return err
	}
	port, err := cmd.Flags().GetUint16("port")
	if err != nil {
		return err
	}
	timeout, err := cmd.Flags().GetUint32("timeout")
	if err != nil {
		return err
	}
	reply, err := con.Rpc.ConnectForwardListener(con.Context(), &clientpb.ForwardListenerConnect{
		ListenerId:     cmd.Flags().Arg(0),
		ConnectHost:    host,
		ConnectPort:    uint32(port),
		TimeoutSeconds: timeout,
	})
	if err != nil {
		return err
	}
	printForwardListenerStatuses(con, []*clientpb.ForwardListenerStatus{reply})
	return nil
}

func ForwardDisconnectCmd(cmd *cobra.Command, con *core.Console) error {
	reply, err := con.Rpc.DisconnectForwardListener(con.Context(), &clientpb.Listener{Id: cmd.Flags().Arg(0)})
	if err != nil {
		return err
	}
	printForwardListenerStatuses(con, []*clientpb.ForwardListenerStatus{reply})
	return nil
}

func ForwardStatusCmd(cmd *cobra.Command, con *core.Console) error {
	if cmd.Flags().NArg() == 0 {
		return ForwardListCmd(cmd, con)
	}
	reply, err := con.Rpc.GetForwardListenerStatus(con.Context(), &clientpb.Listener{Id: cmd.Flags().Arg(0)})
	if err != nil {
		return err
	}
	printForwardListenerStatuses(con, []*clientpb.ForwardListenerStatus{reply})
	return nil
}

func ForwardListCmd(cmd *cobra.Command, con *core.Console) error {
	reply, err := con.Rpc.ListForwardListeners(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	printForwardListenerStatuses(con, reply.GetListeners())
	return nil
}

func printForwardListenerStatuses(con *core.Console, statuses []*clientpb.ForwardListenerStatus) {
	if len(statuses) == 0 {
		con.Log.Importantf("No forward listeners found")
		return
	}
	rows := make([]table.Row, 0, len(statuses))
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("Listener", "Listener", 16),
		table.NewFlexColumn("Address", "Address", 1),
		table.NewColumn("Active", "Active", 7),
		table.NewColumn("Fingerprint", "Fingerprint", 16),
		table.NewFlexColumn("Error", "Error", 1),
	}, true)
	for _, status := range statuses {
		if status == nil {
			continue
		}
		fingerprint := status.GetFingerprint()
		if len(fingerprint) > 16 {
			fingerprint = fingerprint[:16]
		}
		rows = append(rows, table.NewRow(table.RowData{
			"Listener":    status.GetListenerId(),
			"Address":     forwardListenerAddress(status),
			"Active":      strconv.FormatBool(status.GetActive()),
			"Fingerprint": fingerprint,
			"Error":       status.GetError(),
		}))
	}
	tableModel.SetMultiline()
	tableModel.SetRows(rows)
	tableModel.Title = "forward listeners"
	con.Log.Console(tableModel.View())
}

func forwardListenerAddress(status *clientpb.ForwardListenerStatus) string {
	if status.GetAddress() != "" {
		return status.GetAddress()
	}
	if status.GetConnectHost() == "" || status.GetConnectPort() == 0 {
		return ""
	}
	return status.GetConnectHost() + ":" + strconv.Itoa(int(status.GetConnectPort()))
}
