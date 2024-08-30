package file

import (
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/utils"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"path/filepath"

	"google.golang.org/protobuf/proto"
	"os"
)

func uploadCmd(ctx *grumble.Context, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	path := ctx.Args.String("source")
	target := ctx.Args.String("destination")
	priv := ctx.Flags.Int("priv")
	//hidden := ctx.Flags.Bool("hidden")
	task, err := upload(con, path, target, priv)
	if err != nil {
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {

	})
}

func upload(con *console.Console, params ...interface{}) (*clientpb.Task, error) {
	// local, target, priv
	if len(params) < 3 {
		return nil, fmt.Errorf("%w, need 3 params: local, target, priv, bug get %d", utils.ErrFuncHasNotEnoughParams, len(params))
	}
	local := utils.MustGetParam[string](params[0])
	data, err := os.ReadFile(local)
	if err != nil {
		console.Log.Errorf("Can't open file: %s", err)
		return nil, err
	}
	task, err := con.Rpc.Upload(con.ActiveTarget.Context(), &implantpb.UploadRequest{
		Name:   filepath.Base(local),
		Target: utils.MustGetParam[string](params[1]),
		Priv:   utils.MustGetParam[uint32](params[2]),
		Data:   data,
		//Hidden: hidden,
	})
	if err != nil {
		console.Log.Errorf("Download error: %v", err)
		return nil, err
	}

	return task, nil
}
