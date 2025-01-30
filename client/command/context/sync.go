package context

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
)

func SyncCmd(cmd *cobra.Command, con *repl.Console) error {
	tid := cmd.Flags().Arg(0)
	go func() {
		ctx, err := con.Rpc.Sync(con.ActiveTarget.Context(), &clientpb.Sync{
			ContextId: tid,
		})
		if err != nil {
			con.Log.Errorf("sync file error: %v\n", err)
			return
		}

		ictx, err := types.ParseContext(ctx.Type, ctx.Value)
		if err != nil {
			con.Log.Errorf("parse context error: %v\n", err)
			return
		}

		con.Log.Infof("Context: %s\n", ictx.String())

		switch c := ictx.(type) {
		case *types.ScreenShotContext, *types.DownloadContext, *types.KeyLoggerContext, *types.UploadContext:
			var filename string
			var content []byte
			switch t := c.(type) {
			case *types.ScreenShotContext:
				filename = t.Name
				content = t.Content
			case *types.DownloadContext:
				filename = t.Name
				content = t.Content
			case *types.KeyLoggerContext:
				filename = t.Name
				content = t.Content
			case *types.UploadContext:
				filename = t.Name
				content = t.Content
			}

			savePath := filepath.Join(assets.GetTempDir(), fmt.Sprintf("%s_%s", ctx.Id, filename))
			if err := os.WriteFile(savePath, content, 0644); err != nil {
				con.Log.Errorf("write file error: %v\n", err)
				return
			}
			con.Log.Infof("File saved to: %s\n", savePath)
		}
	}()
	return nil
}
