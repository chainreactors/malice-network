package privilege

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
)

func Commands(con *repl.Console) []*cobra.Command {
	runasCmd := &cobra.Command{
		Use:   "runas --username [username] --domain [domain] --password [password] --program [program] --args [args] --show [show] --netonly",
		Short: "Run a program as another user",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunasCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleRunas,
			"ttp":    "T1078.001",
		},
		Example: `Run a program as a different user:
  ~~~
  sys runas --username admin --domain EXAMPLE --password admin123 --program /path/to/program --args "arg1 arg2"
  ~~~`,
	}
	runasCmd.Flags().String("username", "", "Username to run as")
	runasCmd.Flags().String("domain", "", "Domain of the user")
	runasCmd.Flags().String("password", "", "User password")
	runasCmd.Flags().String("program", "", "Path to the program to execute")
	runasCmd.Flags().String("args", "", "Arguments for the program")
	runasCmd.Flags().Int32("show", 1, "Window display mode (1: default)")
	runasCmd.Flags().Bool("netonly", false, "Use network credentials only")

	privsCmd := &cobra.Command{
		Use:   "privs",
		Short: "List available privileges",
		RunE: func(cmd *cobra.Command, args []string) error {
			return PrivsCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModulePrivs,
			"ttp":    "T1134.001",
		},
		Example: `List available privileges:
  ~~~
  sys privs
  ~~~`,
	}

	getSystemCmd := &cobra.Command{
		Use:   "getsystem",
		Short: "Attempt to elevate privileges",
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetSystemCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleGetSystem,
			"ttp":    "T1134.001",
		},
		Example: `Attempt to elevate privileges:
  ~~~
  getsystem
  ~~~`,
	}

	return []*cobra.Command{runasCmd, privsCmd, getSystemCmd}
}

func Register(con *repl.Console) {
	RegisterPrivsFunc(con)
	RegisterGetSystemFunc(con)
	RegisterRunasFunc(con)
}
