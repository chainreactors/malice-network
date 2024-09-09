package file

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/help"
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
		Use:   consts.ModuleDownload,
		Short: "Download file",
		Long:  help.GetHelpFor(consts.ModuleDownload),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			DownloadCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleDownload,
		},
	}

	common.BindArgCompletions(downloadCmd, nil,
		carapace.ActionValues().Usage("file name"),
		carapace.ActionValues().Usage("download file source path"))

	uploadCmd := &cobra.Command{
		Use:   consts.ModuleUpload,
		Short: "Upload file",
		Long:  help.GetHelpFor(consts.ModuleUpload),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			UploadCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleUpload,
		},
	}

	common.BindArgCompletions(uploadCmd, nil,
		carapace.ActionFiles().Usage("file source path"),
		carapace.ActionValues().Usage("file target path"))

	common.BindFlag(uploadCmd, func(f *pflag.FlagSet) {
		f.IntP("priv", "", 0o644, "file privilege")
		f.BoolP("hidden", "", false, "hidden file")
	})

	syncCmd := &cobra.Command{
		Use:   consts.CommandSync,
		Short: "Sync file",
		Long:  help.GetHelpFor(consts.CommandSync),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			SyncCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(syncCmd, nil,
		carapace.ActionValues().Usage("task ID"))

	con.RegisterImplantFunc(
		consts.ModuleDownload,
		Download,
		"bdownload",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path string) (*clientpb.Task, error) {
			return Download(rpc, sess, path)
		},
		nil)

	con.RegisterImplantFunc(
		consts.ModuleUpload,
		Upload,
		"bupload",
		func(rpc clientrpc.MaliceRPCClient, sess *repl.Session, path string) (*clientpb.Task, error) {
			return Upload(rpc, sess, path, filepath.Base(path), 0744, false)
		},
		nil)

	return []*cobra.Command{
		downloadCmd,
		uploadCmd,
		syncCmd,
	}
}
