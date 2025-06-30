package privilege

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	runasCmd := &cobra.Command{
		Use:   "runas --username [username] --domain [domain] --password [password] --program [program] --args [args] --use-profile --use-env --netonly",
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
  sys runas --username admin --domain EXAMPLE --password admin123 --program /path/to/program --args "arg1 arg2" --use-profile --use-env
  ~~~`,
	}

	common.BindFlag(runasCmd, func(f *pflag.FlagSet) {
		f.String("username", "", "Username to run as")
		f.String("domain", "", "Domain of the user")
		f.String("password", "", "User password")
		f.String("path", "", "Path to the program to execute")
		f.String("args", "", "Arguments for the program")
		f.Bool("use-profile", false, "Load user profile")
		f.Bool("use-env", false, "Use user environment")
		f.Bool("netonly", false, "Use network credentials only")
	})

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

	rev2selfCmd := &cobra.Command{
		Use:   "rev2self",
		Short: "Revert to the original token",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Rev2selfCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleRev2Self,
			"ttp":    "T1134.002",
		},
		Example: `Revert to the original token:
  ~~~
  sys rev2self
  ~~~`,
	}

	return []*cobra.Command{runasCmd, privsCmd, getSystemCmd, rev2selfCmd}
}

func Register(con *repl.Console) {
	RegisterPrivsFunc(con)
	RegisterGetSystemFunc(con)
	RegisterRunasFunc(con)
	RegisterRev2selfFunc(con)
}
