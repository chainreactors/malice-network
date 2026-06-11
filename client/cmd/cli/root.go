package cli

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/command"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/generic"
	"github.com/chainreactors/malice-network/client/command/sessions"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func rootCmd(con *core.Console) (*cobra.Command, error) {
	var cmd = &cobra.Command{
		Use: "client",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Propagate mux-child flag to Console.
			if mc, _ := cmd.Flags().GetBool("mux-child"); mc {
				con.MuxChild = true
			}

			if err := common.ValidateExecutionModeFlags(cmd); err != nil {
				return err
			}

			// TUI multiplexer mode: login first (interactive auth selection
			// on the real terminal), then launch the tmux-like manager that
			// spawns child processes with the same auth config.
			if tuiMode, _ := cmd.Flags().GetBool("tui"); tuiMode {
				if err := generic.LoginCmd(cmd, con); err != nil {
					return err
				}
				return startMux(cmd, con)
			}

			if err := generic.LoginCmd(cmd, con); err != nil {
				return err
			}

			// --use flag: auto-switch to a session after login (used by mux child panes).
			if sid, _ := cmd.Flags().GetString("use"); sid != "" {
				if sess, err := con.GetOrUpdateSession(sid); err == nil {
					sessions.Use(con, sess)
				}
			}

			if common.ShouldStartRuntime(cmd) {
				restoreDaemon := con.WithDaemonExecution(common.ShouldStartDaemon(cmd))
				defer restoreDaemon()
				return con.Start(command.BindClientsCommands, command.BindImplantCommands)
			}
			return nil
		},
	}
	cmd.TraverseChildren = true

	// Add --tui flag for terminal multiplexer mode
	cmd.PersistentFlags().Bool("tui", false, "start in TUI multiplexer mode")
	// Hidden flags for mux child processes (set by the multiplexer, not by users)
	cmd.PersistentFlags().Bool("mux-child", false, "internal: run as multiplexed subprocess")
	cmd.PersistentFlags().Bool("quiet", false, "internal: suppress events and services")
	cmd.PersistentFlags().String("use", "", "internal: auto-use session after login")
	cmd.PersistentFlags().MarkHidden("mux-child")
	cmd.PersistentFlags().MarkHidden("quiet")
	cmd.PersistentFlags().MarkHidden("use")
	// Add --mcp flag
	cmd.PersistentFlags().String("mcp", "", "enable MCP server with address (e.g., 127.0.0.1:5005)")
	// Add --rpc flag
	cmd.PersistentFlags().String("rpc", "", "enable local gRPC server with address (e.g., 127.0.0.1:15004)")
	cmd.PersistentFlags().Bool("daemon", false, "keep background services alive without entering the interactive console")
	bind := command.MakeBind(cmd, con, "golang")
	command.BindCommonCommands(bind)
	// Setup console runner
	originalPre, originalPost := command.ConsoleRunnerCmd(con, cmd)
	cmd.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		if originalPre != nil {
			return originalPre(c, args)
		}
		return nil
	}
	cmd.PersistentPostRunE = func(c *cobra.Command, args []string) error {
		if originalPost != nil {
			return originalPost(c, args)
		}
		return nil
	}
	cmd.AddCommand(command.ImplantCmd(con))
	carapace.Gen(cmd)

	return cmd, nil
}
