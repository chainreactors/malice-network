package filesystem

import (
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"google.golang.org/protobuf/proto"
	"os"
	"strconv"
)

func LsCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	path := ctx.Flags.String("path")
	if path == "" {
		path = "./"
	}
	resultChan := make(chan *implantpb.LsResponse)
	lsTask, err := con.Rpc.Ls(con.ActiveTarget.Context(), &implantpb.Request{
		Name:  consts.ModuleLs,
		Input: path,
	})
	if err != nil {
		console.Log.Errorf("Ls error: %v", err)
		return
	}
	con.AddCallback(lsTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetLsResponse()
		resultChan <- resp
	})
	files := <-resultChan
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		{Title: "Name", Width: 20},
		{Title: "IsDir", Width: 5},
		{Title: "Size", Width: 7},
		{Title: "ModTime", Width: 10},
		{Title: "Link", Width: 15},
	}, true)
	for _, file := range files.GetFiles() {
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
	fmt.Printf(tableModel.View(), os.Stdout)
	//newTable := tui.NewModel(tableModel, nil, false, false)
	//err = newTable.Run()
	//if err != nil {
	//	con.SessionLog(sid).Errorf("Error running table: %v", err)
	//}
}
