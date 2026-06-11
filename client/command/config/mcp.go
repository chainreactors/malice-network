package config

import (
	"fmt"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

// MCPConfigCommand returns the mcp subcommand for use under `config`.
func MCPConfigCommand(con *core.Console) *cobra.Command {
	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "Show MCP server configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return MCPShowCmd(con)
		},
		Annotations: map[string]string{
			"static": "true",
		},
		Example: `~~~
// Show MCP status
config mcp

// Enable MCP server
config mcp enable

// Enable MCP on a custom address
config mcp enable --addr 127.0.0.1:6006

// Disable MCP server
config mcp disable
~~~`,
	}

	enableCmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable MCP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return MCPEnableCmd(cmd, con)
		},
		Annotations: map[string]string{
			"static": "true",
		},
	}
	enableCmd.Flags().String("addr", "", "MCP server address (host:port)")

	disableCmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable MCP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return MCPDisableCmd(con)
		},
		Annotations: map[string]string{
			"static": "true",
		},
	}

	mcpCmd.AddCommand(enableCmd, disableCmd)
	return mcpCmd
}

// MCPShowCmd displays MCP configuration.
func MCPShowCmd(con *core.Console) error {
	settings, err := assets.LoadSettings()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}
	printMCPStatus(con, settings)
	return nil
}

// MCPEnableCmd enables and starts the MCP server.
func MCPEnableCmd(cmd *cobra.Command, con *core.Console) error {
	settings, err := assets.LoadSettings()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	if addr, _ := cmd.Flags().GetString("addr"); addr != "" {
		settings.McpAddr = addr
	}

	// Stop existing server if running (addr change or re-enable)
	if con.MCP != nil {
		if err := con.MCP.Stop(); err != nil {
			return fmt.Errorf("failed to stop MCP server: %w", err)
		}
		con.MCP = nil
	}

	settings.McpEnable = true
	if err := assets.SaveSettings(settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	// InitMCPServer logs "MCP server started at ..." on success
	con.InitMCPServer()
	return nil
}

// MCPDisableCmd disables and stops the MCP server.
func MCPDisableCmd(con *core.Console) error {
	settings, err := assets.LoadSettings()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	settings.McpEnable = false
	if con.MCP != nil {
		if err := con.MCP.Stop(); err != nil {
			return fmt.Errorf("failed to stop MCP server: %w", err)
		}
		con.MCP = nil
	}

	if err := assets.SaveSettings(settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	logs.Log.Importantf("MCP server disabled\n")
	return nil
}

func printMCPStatus(con *core.Console, settings *assets.Settings) {
	running := con.MCP != nil
	status := tui.RedFg.Render("Stopped")
	if running {
		status = tui.GreenFg.Render("Running")
	}

	enabled := tui.RedFg.Render("No")
	if settings.McpEnable {
		enabled = tui.GreenFg.Render("Yes")
	}

	values := map[string]string{
		"Enabled": enabled,
		"Address": settings.McpAddr,
		"Status":  status,
	}
	keys := []string{"Enabled", "Address", "Status"}
	con.Log.Console(common.NewKVTable("MCP", keys, values).View() + "\n")
}
