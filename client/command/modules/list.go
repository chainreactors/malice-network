package modules

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/tui"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/charmbracelet/bubbles/list"
	"google.golang.org/protobuf/proto"
)

func listModules(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	sid := con.ActiveTarget.GetInteractive().SessionId
	listTask, err := con.Rpc.ListModules(con.ActiveTarget.Context(), &implantpb.Empty{})
	if err != nil {
		con.SessionLog(sid).Errorf("ListModules error: %v", err)
		return
	}
	con.AddCallback(listTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetModules()
		var modules = make([]list.Item, 0)
		for _, module := range resp.GetModules() {
			modules = append(modules, tui.Item{Ititle: module, Desc: ""})

		}
		listModel := tui.Newlist(modules)
		err := tui.Run(listModel)
		if err != nil {
			con.SessionLog(sid).Errorf("Error running list: %v", err)
		}
	})
}
