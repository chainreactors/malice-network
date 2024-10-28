package taskschd

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
)

func Commands(con *repl.Console) []*cobra.Command {
	taskschdCmd := &cobra.Command{
		Use:   consts.CommandTaskSchd,
		Short: "Manage scheduled tasks",
		Long:  "Perform operations related to scheduled tasks, including listing, creating, starting, stopping, and deleting tasks.",
	}

	taskSchdListCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleTaskSchdList),
		Short: "List all scheduled tasks",
		Long:  "Retrieve a list of all scheduled tasks on the system.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return TaskSchdListCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleTaskSchdList,
			"ttp":    "T1053.005",
		},
		Example: `List all scheduled tasks:
  ~~~
  taskschd list
  ~~~`,
	}

	taskSchdCreateCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleTaskSchdCreate),
		Short: "Create a new scheduled task",
		Long:  "Create a new scheduled task with the specified name, executable path, trigger type, and start boundary.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return TaskSchdCreateCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleTaskSchdCreate,
			"ttp":    "T1053.005",
		},
		Example: `Create a scheduled task:
  ~~~
  taskschd create --name ExampleTask --path /path/to/executable --trigger_type 1 --start_boundary "2023-10-10T09:00:00"
  ~~~`,
	}
	taskSchdCreateCmd.Flags().String("name", "", "Name of the scheduled task (required)")
	taskSchdCreateCmd.Flags().String("path", "", "Path to the executable for the scheduled task (required)")
	taskSchdCreateCmd.Flags().Uint32("trigger_type", 1, "Trigger type for the task (e.g., 1 for daily, 2 for weekly)")
	taskSchdCreateCmd.Flags().String("start_boundary", "", "Start boundary for the scheduled task (e.g., 2023-10-10T09:00:00)")

	taskSchdStartCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleTaskSchdStart) + " [name]",
		Short: "Start a scheduled task",
		Long:  "Start a scheduled task by specifying its name.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return TaskSchdStartCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleTaskSchdStart,
			"ttp":    "T1053.005",
		},
		Example: `Start a scheduled task:
  ~~~
  taskschd start ExampleTask
  ~~~`,
	}

	taskSchdStopCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleTaskSchdStop) + " [name]",
		Short: "Stop a running scheduled task",
		Long:  "Stop a scheduled task by specifying its name.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return TaskSchdStopCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleTaskSchdStop,
			"ttp":    "T1053.005",
		},
		Example: `Stop a scheduled task:
  ~~~
  taskschd stop ExampleTask
  ~~~`,
	}

	taskSchdDeleteCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleTaskSchdDelete) + " [name]",
		Short: "Delete a scheduled task",
		Long:  "Delete a scheduled task by specifying its name.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return TaskSchdDeleteCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleTaskSchdDelete,
			"ttp":    "T1053.005",
		},
		Example: `Delete a scheduled task:
  ~~~
  taskschd delete ExampleTask
  ~~~`,
	}

	taskSchdQueryCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleTaskSchdQuery) + " [name]",
		Short: "Query the configuration of a scheduled task",
		Long:  "Retrieve the current configuration, status, and timing information of a specified scheduled task by name.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return TaskSchdQueryCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleTaskSchdQuery,
			"ttp":    "T1053.005",
		},
		Example: `Query the configuration of a scheduled task:
  ~~~
  taskschd query ExampleTask
  ~~~`,
	}

	taskSchdRunCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleTaskSchdRun) + " [name]",
		Short: "Run a scheduled task immediately",
		Long:  "Execute a scheduled task immediately by specifying its name.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return TaskSchdRunCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleTaskSchdRun,
			"ttp":    "T1053.005",
		},
		Example: `Run a scheduled task immediately:
  ~~~
  taskschd run ExampleTask
  ~~~`,
	}
	taskschdCmd.AddCommand(taskSchdListCmd, taskSchdCreateCmd, taskSchdStartCmd, taskSchdStopCmd, taskSchdDeleteCmd, taskSchdQueryCmd, taskSchdRunCmd)

	return []*cobra.Command{taskschdCmd}
}

func Register(con *repl.Console) {
	RegisterTaskSchdListFunc(con)
	RegisterTaskSchdCreateFunc(con)
	RegisterTaskSchdStartFunc(con)
	RegisterTaskSchdStopFunc(con)
	RegisterTaskSchdDeleteFunc(con)
	RegisterTaskSchdQueryFunc(con)
	RegisterTaskSchdRunFunc(con)
}
