package exec

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
)

func Commands(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		&grumble.Command{
			Name: "execute",
			Help: "Execute command",
			Flags: func(f *grumble.Flags) {
				f.Bool("T", "token", false, "execute command with current token (windows only)")
				f.Bool("o", "output", false, "capture command output")
				f.Bool("s", "save", false, "save output to a file")
				f.Bool("X", "loot", false, "save output as loot")
				f.Bool("S", "ignore-stderr", false, "don't print STDERR output")
				f.String("O", "stdout", "", "remote path to redirect STDOUT to")
				f.String("E", "stderr", "", "remote path to redirect STDERR to")
				f.String("n", "name", "", "name to assign loot (optional)")
				f.Uint("P", "ppid", 0, "parent process id (optional, Windows only)")

				f.Int("t", "timeout", assets.DefaultSettings.DefaultTimeout, "command timeout in seconds")

			},
			Args: func(a *grumble.Args) {
				a.String("command", "command to execute")
				a.StringList("arguments", "arguments to the command")
			},
			Run: func(ctx *grumble.Context) error {
				ExecuteCmd(ctx, con)
				return nil
			},
		},
	}
}
