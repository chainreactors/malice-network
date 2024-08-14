package extension

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
)

// ExtensionsListCmd - List all extension loaded on the active session/beacon
func ExtensionsListCmd(ctx *grumble.Context, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}

	task, err := con.Rpc.ListExtensions(con.ActiveTarget.Context(), &implantpb.Request{
		Name: consts.ModuleListExtension,
	})
	if err != nil {
		console.Log.Errorf("%s\n", err)
		return
	}

	con.AddCallback(task.TaskId, func(msg proto.Message) {
		exts := msg.(*implantpb.Spite).GetExtensions()
		for _, ext := range exts.Extensions {
			con.SessionLog(session.SessionId).Consolef("%s\t%s\t%s", ext.Name, ext.Type, ext.Depend)
		}
	})
}
