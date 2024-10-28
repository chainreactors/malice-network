package service

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	serviceCmd := &cobra.Command{
		Use:   consts.CommandService,
		Short: "Perform service operations",
		Long:  "Manage services, including listing, creating, starting, stopping, and querying service status.",
	}

	serviceListCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleServiceList),
		Short: "List all available services",
		Long:  "Retrieve and display a list of all services available on the system, including their configuration and current status.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ServiceListCmd(cmd, con)
		},
		Example: `List all services:
  ~~~
  service list
  ~~~`,
		Annotations: map[string]string{
			"depend": consts.ModuleServiceList,
			"ttp":    "T1007",
		},
	}

	serviceCreateCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleServiceCreate),
		Short: "Create a new service",
		Long: `Create a new service with specified name, display name, executable path, start type, error control, and account name.
		
Control the start type and error control by providing appropriate values.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ServiceCreateCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleServiceCreate,
			"ttp":    "T1543.003",
		},
		Example: `Create a new service named "example_service":
  ~~~
  service create --name example_service --display "Example Service" --path /path/to/executable --start_type 2 --error 1
  ~~~`,
	}
	common.BindFlag(serviceCreateCmd, func(f *pflag.FlagSet) {
		f.String("name", "", "Name of the service (required)")
		f.String("display", "", "Display name of the service")
		f.String("path", "", "Path to the executable (required)")
		f.Uint32("start_type", 3, "Service start type (e.g., 2 for auto-start)")
		f.Uint32("error", 1, "Error control level (e.g., 1 for normal)")
		f.String("account", "SYSTEM", "Account name for the service")
	})

	serviceStartCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleServiceStart) + " [service_name]",
		Short: "Start an existing service",
		Long:  "Start a service by specifying its name.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ServiceStartCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleServiceStart,
			"ttp":    "T1569.002",
		},
		Example: `Start a service named "example_service":
  ~~~
  service start example_service
  ~~~`,
	}

	serviceStopCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleServiceStop) + " [service_name]",
		Short: "Stop a running service",
		Long:  "Stop a service by specifying its name. This command will halt the service's operation.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ServiceStopCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleServiceStop,
			"ttp":    "T1569.002",
		},
		Example: `Stop a service named "example_service":
  ~~~
  service stop example_service
  ~~~`,
	}

	serviceQueryCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleServiceQuery) + " [service_name]",
		Short: "Query the status of a service",
		Long:  "Retrieve the current status and configuration of a specified service.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ServiceQueryCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleServiceQuery,
			"ttp":    "T1007",
		},
		Example: `Query the status of a service named "example_service":
  ~~~
  service query example_service
  ~~~`,
	}

	// 将所有子命令添加到 serviceCmd
	serviceCmd.AddCommand(serviceListCmd, serviceCreateCmd, serviceStartCmd, serviceStopCmd, serviceQueryCmd)

	return []*cobra.Command{serviceCmd}
}

func Register(con *repl.Console) {
	RegisterServiceListFunc(con)
	RegisterServiceCreateFunc(con)
	RegisterServiceStartFunc(con)
	RegisterServiceStopFunc(con)
	RegisterServiceQueryFunc(con)
}
