package context

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
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

		ictx, err := output.ParseContext(ctx.Type, ctx.Value)
		if err != nil {
			con.Log.Errorf("parse context error: %v\n", err)
			return
		}

		con.Log.Infof("Context: \n%s\n", ictx.String())

		switch c := ictx.(type) {
		case *output.ScreenShotContext, *output.DownloadContext, *output.KeyLoggerContext, *output.UploadContext:
			var filename string
			var content []byte
			switch t := c.(type) {
			case *output.ScreenShotContext:
				filename = t.Name
				content = t.Content
			case *output.DownloadContext:
				filename = t.Name
				content = t.Content
			case *output.KeyLoggerContext:
				filename = t.Name
				content = t.Content
			case *output.UploadContext:
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
