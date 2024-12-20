package rem

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	remCmd := &cobra.Command{
		Use: consts.CommandRem,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		Example: `~~~
rem
~~~`,
	}
	listremCmd := &cobra.Command{
		Use:   consts.CommandListRem + " [listener]",
		Short: "List REMs in listener",
		Long:  "Use a table to list REMs along with their corresponding listeners",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListRemCmd(cmd, con)
		},
		Example: `~~~
rem
~~~`,
	}
	common.BindArgCompletions(listremCmd, nil, common.ListenerIDCompleter(con))

	newRemCmd := &cobra.Command{
		Use:   consts.CommandRemNew + " [name]",
		Short: "Register a new REM and start it",
		Long:  "Register a new REM with the specified listener.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return NewRemCmd(cmd, con)
		},
		Example: `~~~
// Register a REM with the default settings
rem new --listener listener_id

// Register a REM with a custom name and console URL
rem new --name rem_test --listener listener_id -c tcp://127.0.0.1:19966
~~~`,
	}

	common.BindFlag(newRemCmd, func(f *pflag.FlagSet) {
		f.StringP("listener", "l", "", "listener id")
		f.StringP("console", "c", "tcp://0.0.0.0:19966", "REM console URL")
	})

	common.BindFlagCompletions(newRemCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
		comp["console"] = carapace.ActionValues().Usage("REM console URL")
	})
	newRemCmd.MarkFlagRequired("listener")

	startRemCmd := &cobra.Command{
		Use:   consts.CommandRemStart,
		Short: "Start a REM",
		Args:  cobra.ExactArgs(1),
		Long:  "Start a REM with the specified name",
		RunE: func(cmd *cobra.Command, args []string) error {
			return StartRemCmd(cmd, con)
		},
		Example: `~~~
rem start rem_test
~~~`,
	}

	common.BindArgCompletions(startRemCmd, nil,
		carapace.ActionValues().Usage("rem name"))

	stopRemCmd := &cobra.Command{
		Use:   consts.CommandRemStop,
		Short: "Stop a REM",
		Args:  cobra.ExactArgs(1),
		Long:  "Stop a REM with the specified name",
		RunE: func(cmd *cobra.Command, args []string) error {
			return StopRemCmd(cmd, con)
		},
		Example: `~~~
rem stop rem_test
~~~`,
	}

	common.BindArgCompletions(stopRemCmd, nil,
		carapace.ActionValues().Usage("rem name"))

	deleteRemCmd := &cobra.Command{
		Use:   consts.CommandPipelineDelete,
		Short: "Delete a REM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return DeleteRemCmd(cmd, con)
		},
		Example: `~~~
rem delete rem_test
~~~`,
	}

	common.BindArgCompletions(deleteRemCmd, nil,
		carapace.ActionValues().Usage("rem name"))

	remCmd.AddCommand(listremCmd, newRemCmd, startRemCmd, stopRemCmd, deleteRemCmd)

	return []*cobra.Command{remCmd}
}
