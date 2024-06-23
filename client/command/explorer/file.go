package explorer

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/tui"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
	"os"
	"sync"
	"time"
)

func explorerCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}
	sid := con.ActiveTarget.GetInteractive().SessionId
	dirEntriesChan := make(chan []os.DirEntry, 1)
	var wg sync.WaitGroup
	var path = ""

	lsTask, err := con.Rpc.Ls(con.ActiveTarget.Context(), &implantpb.Request{
		Name:  consts.ModuleLs,
		Input: "./",
	})
	if err != nil {
		con.SessionLog(sid).Errorf("load directory error: %v", err)
		return
	}

	con.AddCallback(lsTask.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetLsResponse()
		var dirEntries []os.DirEntry
		for _, protoFile := range resp.GetFiles() {
			dirEntries = append(dirEntries, ProtobufDirEntry{FileInfo: protoFile})
		}
		path = resp.GetPath()
		path = path[4:]
		dirEntriesChan <- dirEntries
	})

	var dirEntries []os.DirEntry
	explorer := NewExplorer(dirEntries, con)
	explorer.FilePicker.CurrentDirectory = "./"
	explorer.FilePicker.Height = 50

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case newEntries := <-dirEntriesChan:
				if len(newEntries) > 0 {
					dirEntries = newEntries
					err := SetFiles(&explorer.FilePicker, dirEntries)
					if err != nil {
						con.SessionLog(sid).Errorf("Error setting files: %v", err)
						return
					}
					explorer.Files = dirEntries
					explorer.FilePicker.CurrentDirectory = path
					explorer.max = max(explorer.max, explorer.FilePicker.Height-1)
					err = tui.Run(explorer)
					if err != nil {
						con.SessionLog(sid).Errorf("Error running explorer: %v", err)
					}
					return
				}
			case <-time.After(1 * time.Second):
				continue
			}
		}
	}()

	wg.Wait()
}
