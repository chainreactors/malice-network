package file

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"path/filepath"
)

func Commands(con *repl.Console) []*cobra.Command {
	downloadCmd := &cobra.Command{
		Use:   consts.ModuleDownload + " [implant_file]",
		Short: "Download file",
		Long:  help.FormatLongHelp("download file in implant"),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			DownloadCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleDownload,
		},
		Example: `~~~
download ./file.txt
			~~~`,
	}

	common.BindArgCompletions(downloadCmd, nil,
		carapace.ActionValues().Usage("file name"),
		carapace.ActionValues().Usage("download file source path"))

	uploadCmd := &cobra.Command{
		Use:   consts.ModuleUpload + " [local] [remote]",
		Short: "Upload file",
		Long:  help.FormatLongHelp("upload local file to remote implant"),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			UploadCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleUpload,
		},
		Example: help.FormatLongHelp(`~~~
upload ./file.txt /tmp/file.txt
			~~~`)}

	common.BindArgCompletions(uploadCmd, nil,
		carapace.ActionFiles().Usage("file source path"),
		carapace.ActionValues().Usage("file target path"))

	common.BindFlag(uploadCmd, func(f *pflag.FlagSet) {
		f.IntP("priv", "", 0o644, "file privilege")
		f.BoolP("hidden", "", false, "hidden file")
	})

	syncCmd := &cobra.Command{
		Use:   consts.CommandSync + " [file_id]",
		Short: "Sync file",
		Long:  help.FormatLongHelp("sync download file in server"),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			SyncCmd(cmd, con)
			return
		},
		Example: `~~~
sync 1
			~~~`,
	}

	common.BindArgCompletions(syncCmd, nil,
		carapace.ActionValues().Usage("file ID"))

	return []*cobra.Command{
		downloadCmd,
		uploadCmd,
		syncCmd,
	}
}

func Register(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleDownload,
		Download,
		"bdownload",
		Download,
		common.ParseStatus,
		nil)

	intermediate.RegisterInternalDoneCallback(consts.ModuleDownload, func(content *clientpb.TaskContext) (string, error) {
		return fmt.Sprintf("download block %d/%d success", content.Task.Cur, content.Task.Total), nil
	})

	con.RegisterImplantFunc(
		consts.ModuleUpload,
		Upload,
		"bupload",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, path string) (*clientpb.Task, error) {
			return Upload(rpc, sess, path, filepath.Base(path), 0744, false)
		},
		common.ParseStatus,
		nil)

	intermediate.RegisterInternalDoneCallback(consts.ModuleUpload, func(content *clientpb.TaskContext) (string, error) {
		return fmt.Sprintf("upload block %d/%d success", content.Task.Cur, content.Task.Total), nil
	})
}
