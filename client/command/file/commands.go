package file

import (
	"github.com/chainreactors/malice-network/client/command/flags"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *console.Console) []*cobra.Command {
	downloadCmd := &cobra.Command{
		Use:   consts.ModuleDownload,
		Short: "Download file",
		Long:  help.GetHelpFor(consts.ModuleDownload),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			DownloadCmd(cmd, con)
			return
		},
		GroupID: consts.ImplantGroup,
	}

	carapace.Gen(downloadCmd).PositionalCompletion(
		carapace.ActionValues().Usage("file name"),
		carapace.ActionValues().Usage("download file source path"),
	)

	uploadCmd := &cobra.Command{
		Use:   consts.ModuleUpload,
		Short: "Upload file",
		Long:  help.GetHelpFor(consts.ModuleUpload),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			UploadCmd(cmd, con)
			return
		},
		GroupID: consts.ImplantGroup,
	}

	carapace.Gen(uploadCmd).PositionalCompletion(
		carapace.ActionFiles().Usage("file source path"),
		carapace.ActionValues().Usage("file target path"),
	)

	flags.Bind(consts.ModuleUpload, false, uploadCmd, func(f *pflag.FlagSet) {
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
		GroupID: consts.ImplantGroup,
	}

	carapace.Gen(syncCmd).PositionalCompletion(
		carapace.ActionValues().Usage("task ID"),
	)

	return []*cobra.Command{
		downloadCmd,
		uploadCmd,
		syncCmd,
	}
}
