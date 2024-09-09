package filesystem

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"strconv"
)

func LsCmd(cmd *cobra.Command, con *repl.Console) {
	path := cmd.Flags().Arg(0)
	if path == "" {
		path = "./"
	}
	session := con.GetInteractive()
	task, err := Ls(con.Rpc, session, path)
	if err != nil {
		repl.Log.Errorf("Ls error: %v", err)
		return
	}
	con.AddCallback(task, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetLsResponse()
		var rowEntries []table.Row
		var row table.Row
		tableModel := tui.NewTable([]table.Column{
			{Title: "Name", Width: 20},
			{Title: "IsDir", Width: 5},
			{Title: "Size", Width: 7},
			{Title: "ModTime", Width: 10},
			{Title: "Link", Width: 15},
		}, true)
		for _, file := range resp.GetFiles() {
			row = table.Row{
				file.Name,
				strconv.FormatBool(file.IsDir),
				strconv.FormatUint(file.Size, 10),
				strconv.FormatInt(file.ModTime, 10),
				file.Link,
			}
			rowEntries = append(rowEntries, row)
		}
		tableModel.SetRows(rowEntries)
		fmt.Printf(tableModel.View())
	})
}

func Ls(rpc clientrpc.MaliceRPCClient, session *repl.Session, path string) (*clientpb.Task, error) {
	task, err := rpc.Ls(session.Context(), &implantpb.Request{
		Name:  consts.ModuleLs,
		Input: path,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}
