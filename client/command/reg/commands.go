package reg

import (
	"strings"

	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func FormatRegPath(path string) (string, string) {
	path = fileutils.FormatWindowPath(path)
	i := strings.Index(path, "\\")
	if i == -1 {
		return path, ""
	} else {
		return path[:i], path[i+1:]
	}
}

func Commands(con *repl.Console) []*cobra.Command {
	regCmd := &cobra.Command{
		Use:   consts.CommandReg,
		Short: "Perform registry operations",
		Long:  "Manage Windows registry entries, including querying, adding, deleting, listing keys, and listing values.",
		Annotations: map[string]string{
			"depend": consts.ModuleRegQuery,
			"ttp":    "T1012",
		},
	}

	regQueryCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleRegQuery) + " --hive [hive] --path [path] --key [key]",
		Short: "Query a registry key",
		Long:  "Retrieve the value associated with a specific registry key.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RegQueryCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleRegQuery,
			"ttp":    "T1012",
		},
		Example: `Query a registry key:
  ~~~
  reg query HKEY_LOCAL_MACHINE\\SOFTWARE\\Example TestKey
  ~~~`,
	}

	regAddCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleRegAdd) + " [path] /v [value_name] /t [type] /d [data]",
		Short: "Add or modify a registry key",
		Long:  "Add or modify a registry key with specified values. Supported types: REG_SZ, REG_BINARY, REG_DWORD, REG_QWORD",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RegAddCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleRegAdd,
			"ttp":    "T1112",
		},
		Example: `Add or modify a registry key:
  ~~~
  reg add HKEY_LOCAL_MACHINE\\SOFTWARE\\Example /v TestValue /t REG_DWORD /d 1
  reg add HKEY_LOCAL_MACHINE\\SOFTWARE\\Example /v TestString /t REG_SZ /d "Hello World"
  reg add HKEY_LOCAL_MACHINE\\SOFTWARE\\Example /v TestBinary /t REG_BINARY /d 01020304
  ~~~`,
	}
	common.BindFlag(regAddCmd, func(f *pflag.FlagSet) {
		f.String("v", "", "Value name")
		f.String("t", "REG_SZ", "Value type (REG_SZ, REG_BINARY, REG_DWORD, REG_QWORD)")
		f.String("d", "", "Data to set")
	})

	regDeleteCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleRegDelete) + " --hive [hive] --path [path] --key [key]",
		Short: "Delete a registry key",
		Long:  "Remove a specific registry key.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RegDeleteCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleRegDelete,
			"ttp":    "T1112",
		},
		Example: `Delete a registry key:
  ~~~
  reg delete HKEY_LOCAL_MACHINE\\SOFTWARE\\Example TestKey
  ~~~`,
	}

	regListKeyCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleRegListKey) + " --hive [hive] --path [path]",
		Short: "List subkeys in a registry path",
		Long:  "Retrieve a list of all subkeys under a specified registry path.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RegListKeyCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleRegListKey,
			"ttp":    "T1012",
		},
		Example: `List subkeys in a registry path:
  ~~~
  reg list_key HKEY_LOCAL_MACHINE\\SOFTWARE\\Example
  ~~~`,
	}

	regListValueCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleRegListValue) + " --hive [hive] --path [path]",
		Short: "List values in a registry path",
		Long:  "Retrieve a list of all values under a specified registry path.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RegListValueCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleRegListValue,
			"ttp":    "T1012",
		},
		Example: `List values in a registry path:
  ~~~
  reg list_value HKEY_LOCAL_MACHINE\\SOFTWARE\\Example
  ~~~`,
	}

	// 将所有子命令添加到 regCmd
	regCmd.AddCommand(regQueryCmd, regAddCmd, regDeleteCmd, regListKeyCmd, regListValueCmd)

	return []*cobra.Command{regCmd}
}

func Register(con *repl.Console) {
	RegisterRegQueryFunc(con)
	RegisterRegAddFunc(con)
	RegisterRegDeleteFunc(con)
	RegisterRegListFunc(con)
}
