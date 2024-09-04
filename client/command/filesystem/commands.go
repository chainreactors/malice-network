package filesystem

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

	pwdCmd := &cobra.Command{
		Use:   consts.ModulePwd,
		Short: "Print working directory",
		Long:  help.GetHelpFor(consts.ModulePwd),
		Run: func(cmd *cobra.Command, args []string) {
			PwdCmd(cmd, con)
			return
		},
	}

	catCmd := &cobra.Command{
		Use:   consts.ModuleCat,
		Short: "Print file content",
		Long:  help.GetHelpFor(consts.ModuleCat),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			CatCmd(cmd, con)
			return
		},
	}

	carapace.Gen(catCmd).PositionalCompletion(
		carapace.ActionValues().Usage("cat file name"),
	)

	cdCmd := &cobra.Command{
		Use:   consts.ModuleCd,
		Short: "Change directory",
		Long:  help.GetHelpFor(consts.ModuleCd),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			CdCmd(cmd, con)
			return
		},
	}

	carapace.Gen(cdCmd).PositionalCompletion(
		carapace.ActionValues().Usage("cd path"),
	)

	chmodCmd := &cobra.Command{
		Use:   consts.ModuleChmod,
		Short: "Change file mode",
		Long:  help.GetHelpFor(consts.ModuleChmod),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			ChmodCmd(cmd, con)
			return
		},
	}

	carapace.Gen(chmodCmd).PositionalCompletion(
		carapace.ActionValues().Usage("chmod file mode"),
		carapace.ActionValues().Usage("chmod file path"),
	)

	chownCmd := &cobra.Command{
		Use:   consts.ModuleChown,
		Short: "Change file owner",
		Long:  help.GetHelpFor(consts.ModuleChown),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			ChownCmd(cmd, con)
			return
		},
	}

	carapace.Gen(chownCmd).PositionalCompletion(
		carapace.ActionValues().Usage("chown user"),
		carapace.ActionValues().Usage("chown file path"),
	)

	flags.Bind(consts.ModuleChown, false, chownCmd, func(f *pflag.FlagSet) {
		f.BoolP("recursive", "r", false, "recursive")
		f.StringP("gid", "g", "", "Group id")
	})

	cpCmd := &cobra.Command{
		Use:   consts.ModuleCp,
		Short: "Copy file",
		Long:  help.GetHelpFor(consts.ModuleCp),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			CpCmd(cmd, con)
			return
		},
	}

	carapace.Gen(cpCmd).PositionalCompletion(
		carapace.ActionValues().Usage("source file"),
		carapace.ActionValues().Usage("target file"),
	)

	lsCmd := &cobra.Command{
		Use:   consts.ModuleLs,
		Short: "List directory",
		Long:  help.GetHelpFor(consts.ModuleLs),
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			LsCmd(cmd, con)
			return
		},
	}

	carapace.Gen(lsCmd).PositionalCompletion(
		carapace.ActionValues().Usage("ls path"),
	)

	mkdirCmd := &cobra.Command{
		Use:   consts.ModuleMkdir,
		Short: "Make directory",
		Long:  help.GetHelpFor(consts.ModuleMkdir),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			MkdirCmd(cmd, con)
			return
		},
	}

	carapace.Gen(mkdirCmd).PositionalCompletion(
		carapace.ActionValues().Usage("mkdir path"),
	)

	mvCmd := &cobra.Command{
		Use:   consts.ModuleMv,
		Short: "Move file",
		Long:  help.GetHelpFor(consts.ModuleMv),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			MvCmd(cmd, con)
			return
		},
	}

	carapace.Gen(mvCmd).PositionalCompletion(
		carapace.ActionValues().Usage("source file"),
		carapace.ActionValues().Usage("target file"),
	)

	rmCmd := &cobra.Command{
		Use:   consts.ModuleRm,
		Short: "Remove file",
		Long:  help.GetHelpFor(consts.ModuleRm),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			RmCmd(cmd, con)
			return
		},
	}

	carapace.Gen(rmCmd).PositionalCompletion(
		carapace.ActionValues().Usage("rm file name"),
	)

	return []*cobra.Command{
		pwdCmd,
		catCmd,
		cdCmd,
		chmodCmd,
		chownCmd,
		cpCmd,
		lsCmd,
		mkdirCmd,
		mvCmd,
		rmCmd,
	}
}
