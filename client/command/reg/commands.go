package reg

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	regCmd := &cobra.Command{
		Use:   consts.CommandReg,
		Short: "Perform registry operations",
		Long:  "Manage Windows registry entries, including querying, adding, deleting, listing keys, and listing values.",
	}

	regQueryCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleRegQuery) + " --hive [hive] --path [path] --key [key]",
		Short: "Query a registry key",
		Long:  "Retrieve the value associated with a specific registry key.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RegQueryCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleRegQuery,
			"ttp":    "T1012",
		},
		Example: `Query a registry key:
  ~~~
  reg query --hive HKEY_LOCAL_MACHINE --path SOFTWARE\\Example --key TestKey
  ~~~`,
	}
	common.BindFlag(regQueryCmd, common.RegistryFlagSet)

	regAddCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleRegAdd) + " --hive [hive] --path [path] --key [key]",
		Short: "Add or modify a registry key",
		Long:  "Add or modify a registry key with specified values such as string, byte, DWORD, or QWORD.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RegAddCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleRegAdd,
			"ttp":    "T1112",
		},
		Example: `Add or modify a registry key:
  ~~~
  reg add --hive HKEY_LOCAL_MACHINE --path SOFTWARE\\Example --key TestKey --string_value "example" --dword_value 1
  ~~~`,
	}
	common.BindFlag(regQueryCmd, common.RegistryFlagSet, func(f *pflag.FlagSet) {
		f.String("string_value", "", "String value to write")
		f.BytesBase64("byte_value", []byte{}, "Byte array value to write")
		f.Uint32("dword_value", 0, "DWORD value to write")
		f.Uint64("qword_value", 0, "QWORD value to write")
		f.Uint32("regtype", 1, "Registry data type (e.g., 1 for REG_SZ)")
	})

	regDeleteCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleRegDelete) + " --hive [hive] --path [path] --key [key]",
		Short: "Delete a registry key",
		Long:  "Remove a specific registry key.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RegDeleteCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleRegDelete,
			"ttp":    "T1112",
		},
		Example: `Delete a registry key:
  ~~~
  reg delete --hive HKEY_LOCAL_MACHINE --path SOFTWARE\\Example --key TestKey
  ~~~`,
	}
	common.BindFlag(regDeleteCmd, common.RegistryFlagSet)

	regListKeyCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleRegListKey) + " --hive [hive] --path [path]",
		Short: "List subkeys in a registry path",
		Long:  "Retrieve a list of all subkeys under a specified registry path.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RegListKeyCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleRegListKey,
			"ttp":    "T1012",
		},
		Example: `List subkeys in a registry path:
  ~~~
  reg listkey --hive HKEY_LOCAL_MACHINE --path SOFTWARE\\Example
  ~~~`,
	}
	common.BindFlag(regListKeyCmd, common.RegistryFlagSet)

	regListValueCmd := &cobra.Command{
		Use:   consts.SubCommandName(consts.ModuleRegListValue) + " --hive [hive] --path [path]",
		Short: "List values in a registry path",
		Long:  "Retrieve a list of all values under a specified registry path.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RegListValueCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleRegListValue,
			"ttp":    "T1012",
		},
		Example: `List values in a registry path:
  ~~~
  reg listvalue --hive HKEY_LOCAL_MACHINE --path SOFTWARE\\Example
  ~~~`,
	}
	common.BindFlag(regListValueCmd, common.RegistryFlagSet)

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
