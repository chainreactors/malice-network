package context

import (
	"fmt"

	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/assets"
)

func SyncCmd(cmd *cobra.Command, con *core.Console) error {
	tid := cmd.Flags().Arg(0)
	if tid == "" {
		return fmt.Errorf("context_id is required")
	}

	ctx, err := con.Rpc.Sync(con.Context(), &clientpb.Sync{
		ContextId: tid,
	})
	if err != nil {
		return fmt.Errorf("sync context failed: %w", err)
	}

	ictx, err := output.ParseContext(ctx.Type, ctx.Value)
	if err != nil {
		return fmt.Errorf("parse context failed: %w", err)
	}

	con.Log.Infof("Context: \n%s\n", ictx.String())

	switch c := ictx.(type) {
	case *output.ScreenShotContext, *output.DownloadContext, *output.KeyLoggerContext, *output.UploadContext, *output.MediaContext:
		var filename string
		var content []byte
		switch t := c.(type) {
		case *output.ScreenShotContext:
			filename = t.Name
			content = ctx.Content
		case *output.DownloadContext:
			filename = t.Name
			content = ctx.Content
		case *output.KeyLoggerContext:
			filename = t.Name
			content = ctx.Content
		case *output.UploadContext:
			filename = t.Name
			content = ctx.Content
		case *output.MediaContext:
			filename = t.Name
			content = ctx.Content
		}

		savePath := filepath.Join(assets.GetTempDir(), fmt.Sprintf("%s_%s", ctx.Id, filename))
		if err := os.WriteFile(savePath, content, 0o644); err != nil {
			return fmt.Errorf("write file failed: %w", err)
		}
		con.Log.Infof("File saved to: %s\n", savePath)
	}

	return nil
}
