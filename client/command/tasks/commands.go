package tasks

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
)

func Commands(con *repl.Console) []*cobra.Command {
	taskCmd := &cobra.Command{
		Use:   consts.CommandTasks,
		Short: "List tasks",
		Long:  help.FormatLongHelp(consts.CommandTasks),
		Run: func(cmd *cobra.Command, args []string) {
			listTasks(cmd, con)
			return
		},
	}

	fileCmd := &cobra.Command{
		Use:   consts.CommandFiles,
		Short: "List files",
		Long:  help.FormatLongHelp(consts.CommandFiles),
		Run: func(cmd *cobra.Command, args []string) {
			listFiles(cmd, con)
			return
		},
	}

	cancelTaskCmd := &cobra.Command{
		Use:   consts.ModuleCancelTask + " [task_id]",
		Short: "Cancel a task",
		Long:  help.FormatLongHelp(consts.ModuleCancelTask),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			CancelTaskCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(cancelTaskCmd, nil, common.SessionTaskComplete(con))
	return []*cobra.Command{taskCmd, fileCmd, cancelTaskCmd}
}

func Register(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleCancelTask,
		CancelTaskCmd,
		"",
		nil,
		common.ParseStatus,
		nil)
}
