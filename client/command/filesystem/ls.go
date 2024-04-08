package filesystem

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/tui"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/charmbracelet/bubbles/table"
	"google.golang.org/protobuf/proto"
	"strconv"
)

func LsCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	sid := con.ActiveTarget.GetInteractive().SessionId
	path := ctx.Flags.String("path")
	if path == "" {
		path = "./"
	}
	lsTask, err := con.Rpc.Ls(con.ActiveTarget.Context(), &implantpb.Request{
		Name:  consts.ModuleLs,
		Input: path,
	})
	if err != nil {
		con.SessionLog(sid).Errorf("Ls error: %v", err)
		return
	}
	con.AddCallback(lsTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetLsResponse()
		var rowEntries []table.Row
		var row table.Row
		tableModel := tui.NewTable([]table.Column{
			{Title: "Name", Width: 10},
			{Title: "IsDir", Width: 5},
			{Title: "Size", Width: 7},
			{Title: "ModTime", Width: 10},
			{Title: "Link", Width: 15},
		})
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
		tableModel.Rows = rowEntries
		tableModel.SetRows()
		tableModel.SetHandle(func() {
		})
		err := tui.Run(tableModel)
		if err != nil {
			con.SessionLog(sid).Errorf("Error running table: %v", err)
		}
	})
}
