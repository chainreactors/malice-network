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

// LocalRPCConfigCommand returns the localrpc subcommand for use under `config`.
func LocalRPCConfigCommand(con *core.Console) *cobra.Command {
	localrpcCmd := &cobra.Command{
		Use:   "localrpc",
		Short: "Show Local RPC server configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return LocalRPCShowCmd(con)
		},
		Annotations: map[string]string{
			"static": "true",
		},
		Example: `~~~
// Show Local RPC status
config localrpc

// Enable Local RPC server
config localrpc enable

// Enable Local RPC on a custom address
config localrpc enable --addr 127.0.0.1:16004

// Disable Local RPC server
config localrpc disable
~~~`,
	}

	enableCmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable Local RPC server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return LocalRPCEnableCmd(cmd, con)
		},
		Annotations: map[string]string{
			"static": "true",
		},
	}
	enableCmd.Flags().String("addr", "", "Local RPC server address (host:port)")

	disableCmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable Local RPC server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return LocalRPCDisableCmd(con)
		},
		Annotations: map[string]string{
			"static": "true",
		},
	}

	localrpcCmd.AddCommand(enableCmd, disableCmd)
	return localrpcCmd
}

// LocalRPCShowCmd displays Local RPC configuration.
func LocalRPCShowCmd(con *core.Console) error {
	settings, err := assets.LoadSettings()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}
	printLocalRPCStatus(con, settings)
	return nil
}

// LocalRPCEnableCmd enables and starts the Local RPC server.
func LocalRPCEnableCmd(cmd *cobra.Command, con *core.Console) error {
	settings, err := assets.LoadSettings()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	if addr, _ := cmd.Flags().GetString("addr"); addr != "" {
		settings.LocalRPCAddr = addr
	}

	if con.LocalRPC != nil {
		if err := con.LocalRPC.Stop(); err != nil {
			return fmt.Errorf("failed to stop Local RPC server: %w", err)
		}
		con.LocalRPC = nil
	}

	settings.LocalRPCEnable = true
	if err := assets.SaveSettings(settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	con.InitLocalRPCServer()
	return nil
}

// LocalRPCDisableCmd disables and stops the Local RPC server.
func LocalRPCDisableCmd(con *core.Console) error {
	settings, err := assets.LoadSettings()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	settings.LocalRPCEnable = false
	if con.LocalRPC != nil {
		if err := con.LocalRPC.Stop(); err != nil {
			return fmt.Errorf("failed to stop Local RPC server: %w", err)
		}
		con.LocalRPC = nil
	}

	if err := assets.SaveSettings(settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	logs.Log.Importantf("Local RPC server disabled\n")
	return nil
}

func printLocalRPCStatus(con *core.Console, settings *assets.Settings) {
	running := con.LocalRPC != nil
	status := tui.RedFg.Render("Stopped")
	if running {
		status = tui.GreenFg.Render("Running")
	}

	enabled := tui.RedFg.Render("No")
	if settings.LocalRPCEnable {
		enabled = tui.GreenFg.Render("Yes")
	}

	values := map[string]string{
		"Enabled": enabled,
		"Address": settings.LocalRPCAddr,
		"Status":  status,
	}
	keys := []string{"Enabled", "Address", "Status"}
	con.Log.Console(common.NewKVTable("LocalRPC", keys, values).View() + "\n")
}
