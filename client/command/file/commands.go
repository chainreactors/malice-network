package file

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"path/filepath"

	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	downloadCmd := &cobra.Command{
		Use:   consts.ModuleDownload + " [implant_file]",
		Short: "Download file",
		Long:  "download file in implant",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return DownloadCmd(cmd, con)
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
		Long:  "upload local file to remote implant",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return UploadCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleUpload,
		},
		Example: `~~~
upload ./file.txt /tmp/file.txt
~~~`,
	}

	common.BindArgCompletions(uploadCmd, nil,
		carapace.ActionFiles().Usage("file source path"),
		carapace.ActionValues().Usage("file target path"))

	common.BindFlag(uploadCmd, func(f *pflag.FlagSet) {
		f.String("priv", "0644", "file privilege")
		f.BoolP("hidden", "", false, "hidden file")
	})

	syncCmd := &cobra.Command{
		Use:   consts.CommandSync + " [file_id]",
		Short: "Sync file",
		Long:  "sync download file in server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return SyncCmd(cmd, con)
		},
		Example: `~~~
sync 1
~~~`,
	}

	common.BindArgCompletions(syncCmd, nil,
		common.SyncFileCompleter(con))

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

	con.AddCommandFuncHelper(
		consts.ModuleDownload,
		consts.ModuleDownload,
		consts.ModuleDownload+"(active(),`file.txt`)",
		[]string{
			"session: special session",
			"path: file path",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		consts.ModuleUpload,
		Upload,
		"bupload",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, path string) (*clientpb.Task, error) {
			return Upload(rpc, sess, path, filepath.Base(path), "0644", false)
		},
		common.ParseStatus,
		nil)

	con.RegisterImplantFunc(
		"uploadraw",
		UploadRaw,
		"buploadraw",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, data, target_path string) (*clientpb.Task, error) {
			return UploadRaw(rpc, sess, data, target_path, "0644", false)
		},
		common.ParseStatus,
		nil)

	intermediate.RegisterInternalDoneCallback(consts.ModuleUpload, func(content *clientpb.TaskContext) (string, error) {
		return fmt.Sprintf("upload block %d/%d success", content.Task.Cur, content.Task.Total), nil
	})

	con.AddCommandFuncHelper(
		consts.ModuleUpload,
		consts.ModuleUpload,
		consts.ModuleUpload+`(active(),"/source/path","/target/path",parse_octal("644"),false)`,
		[]string{
			"session: special session",
			"path: source path",
			"target: target path",
			"priv",
			"hidden",
		},
		[]string{"task"})

	con.AddCommandFuncHelper(
		"bupload",
		"bupload",
		`bupload(active(),"/source/path")`,
		[]string{
			"session: special session",
			"path: source path",
		},
		[]string{"task"})

}
