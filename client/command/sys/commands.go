package sys

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	whoamiCmd := &cobra.Command{
		Use:   consts.ModuleWhoami,
		Short: "Print current user",
		RunE: func(cmd *cobra.Command, args []string) error {
			return WhoamiCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleWhoami,
			"ttp":    "T1033",
		},
	}

	killCmd := &cobra.Command{
		Use:   consts.ModuleKill + " [pid]",
		Short: "Kill the process by pid",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return KillCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleKill,
			"ttp":    "T1106",
		},
		Example: `kill the process which pid is 1234
~~~
kill 1234
~~~`,
	}

	common.BindArgCompletions(killCmd, nil,
		carapace.ActionValues().Usage("process pid"))

	psCmd := &cobra.Command{
		Use:   consts.ModulePs,
		Short: "List processes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return PsCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModulePs,
			"ttp":    "T1057",
		},
	}

	envCmd := &cobra.Command{
		Use:   consts.ModuleEnv,
		Short: "List environment variables",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return EnvCmd(cmd, con)
			} else {
				return fmt.Errorf("unknown cmd '%s'", args[0])
			}
		},
		Annotations: map[string]string{
			"depend": consts.ModuleEnv,
			"ttp":    "T1134",
		},
	}

	setEnvCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleSetEnv) + " [env-key] [env-value]",
		Short: "Set environment variable",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return SetEnvCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleSetEnv,
			"ttp":    "T1134",
		},
		Example: `~~~
	setenv key1 value1
	~~~`,
	}

	common.BindArgCompletions(setEnvCmd, nil,
		carapace.ActionValues().Usage("environment variable"),
		carapace.ActionValues().Usage("value"))

	unSetEnvCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleUnsetEnv) + " [env-key]",
		Short: "Unset environment variable",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return UnsetEnvCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleUnsetEnv,
			"ttp":    "T1134",
		},
		Example: `~~~
	unsetenv key1
	~~~`,
	}

	common.BindArgCompletions(unSetEnvCmd, nil,
		carapace.ActionValues().Usage("environment variable"))

	envCmd.AddCommand(unSetEnvCmd, setEnvCmd)

	netstatCmd := &cobra.Command{
		Use:   consts.ModuleNetstat,
		Short: "List network connections",
		RunE: func(cmd *cobra.Command, args []string) error {
			return NetstatCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleNetstat,
			"ttp":    "T1049",
		},
	}

	infoCmd := &cobra.Command{
		Use:   consts.ModuleSysInfo,
		Short: "Get basic sys info",
		RunE: func(cmd *cobra.Command, args []string) error {
			return InfoCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleSysInfo,
			"ttp":    "T1082",
		},
	}

	bypassCmd := &cobra.Command{
		Use:   consts.ModuleBypass,
		Short: "Bypass AMSI and ETW",
		RunE: func(cmd *cobra.Command, args []string) error {
			return BypassCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleBypass,
			"ttp":    "T1562.001",
		},
		Example: `
~~~
bypass --amsi --etw
~~~`,
	}

	common.BindFlag(bypassCmd, func(f *pflag.FlagSet) {
		f.Bool("amsi", false, "Bypass AMSI")
		f.Bool("etw", false, "Bypass ETW")
	})

	wmiQueryCmd := &cobra.Command{
		Use:   consts.ModuleWmiQuery,
		Short: "Perform a WMI query",
		Long:  "Executes a WMI query within the specified namespace to retrieve system information or perform administrative actions.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return WmiQueryCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleWmiQuery,
			"ttp":    "T1047",
		},
		Example: `Perform a WMI query in the root\\cimv2 namespace:
  ~~~
  wmiquery --namespace root\\cimv2 --args "SELECT * FROM Win32_Process"
  ~~~`,
	}
	wmiQueryCmd.Flags().String("namespace", "", "WMI namespace (e.g., root\\cimv2)")
	wmiQueryCmd.Flags().StringSlice("args", []string{}, "Arguments for the WMI query")

	wmiExecuteCmd := &cobra.Command{
		Use:   consts.ModuleWmiExec,
		Short: "Execute a WMI method",
		Long:  "Executes a specified method within a WMI class, allowing for more complex administrative actions via WMI.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return WmiExecuteCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleWmiExec,
			"ttp":    "T1047",
		},
		Example: `Execute a WMI method:
  ~~~
  wmiexecute --namespace root\\cimv2 --class_name Win32_Process --method_name Create --params CommandLine="notepad.exe"
  ~~~`,
	}

	common.BindFlag(wmiExecuteCmd, func(f *pflag.FlagSet) {
		f.String("namespace", "", "WMI namespace (e.g., root\\cimv2)")
		f.String("class_name", "", "WMI class name")
		f.String("method_name", "", "WMI method name")
		f.StringToString("params", map[string]string{}, "Parameters for the WMI method")
	})

	return []*cobra.Command{
		whoamiCmd,
		killCmd,
		psCmd,
		envCmd,
		netstatCmd,
		infoCmd,
		bypassCmd,
		wmiQueryCmd,
		wmiExecuteCmd,
	}
}

func Register(con *repl.Console) {
	RegisterEnvFunc(con)
	RegisterPsFunc(con)
	RegisterNetstatFunc(con)
	RegisterInfoFunc(con)
	RegisterBypassFunc(con)
	RegisterKillFunc(con)
	RegisterWhoamiFunc(con)
	RegisterWmiFunc(con)
}
