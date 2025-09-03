package pty

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Commands returns PTY-related cobra commands
func Commands(con *repl.Console) []*cobra.Command {
	shellCmd := &cobra.Command{
		Use:   consts.ModuleClientPty,
		Short: "Start an interactive PTY shell session",
		Long: `Start an interactive pseudo-terminal (PTY) shell session with the implant.
This provides a real terminal experience with:
- Real-time bidirectional communication
- Terminal resizing support
- Session persistence
- Multiple shell types (bash, cmd, powershell)

Use Ctrl+C to exit the shell.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ShellCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": "pty", // 依赖 PTY 模块
		},
		Example: `Start a bash shell (Linux/macOS):
~~~
pty
~~~

Start a PowerShell session (Windows):
~~~
pty --shell powershell
~~~

Start with custom session ID:
~~~
pty --session-id my_session --shell /bin/zsh
~~~`,
	}

	common.BindFlag(shellCmd, func(f *pflag.FlagSet) {
		f.StringP("shell", "s", "", "shell type (bash, cmd, powershell, zsh, etc.)")
		f.StringP("session-id", "i", "", "custom session ID")
		f.IntP("cols", "c", 80, "terminal columns")
		f.IntP("rows", "r", 24, "terminal rows")
		f.BoolP("background", "b", false, "run in background (non-interactive)")
	})

	common.BindFlagCompletions(shellCmd, func(comp carapace.ActionMap) {
		comp["shell"] = carapace.ActionValues(
			"bash", "sh", "zsh", "fish", // Unix shells
			"cmd", "powershell", "pwsh", // Windows shells
		).Usage("shell type")
	})

	return []*cobra.Command{
		shellCmd,
	}
}

// Register registers PTY-related functions with the console
func Register(con *repl.Console) {
	// 注册 PTY 相关的功能函数
	// 可以在这里添加自动补全、帮助信息等
}
