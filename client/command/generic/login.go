package generic

import (
	"errors"
	"fmt"
	"strings"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

func LoginCmd(cmd *cobra.Command, con *core.Console) error {
	var err error
	quiet := common.ShouldSuppressStartupOutput(cmd)

	// 处理 --mcp flag
	mcpAddr, _ := cmd.Flags().GetString("mcp")
	if mcpAddr != "" {
		if !quiet {
			con.Log.Importantf("MCP will start at %s after login", mcpAddr)
		}
		con.MCPAddr = mcpAddr
	}

	// 处理 --rpc flag
	rpcAddr, _ := cmd.Flags().GetString("rpc")
	if rpcAddr != "" {
		if !quiet {
			con.Log.Importantf("Local RPC will start at %s after login", rpcAddr)
		}
		con.RPCAddr = rpcAddr
	}

	// Prefer explicit --auth flag to avoid misinterpreting subcommand arguments
	// (e.g. `build beacon`) as an auth file.
	if filename, _ := cmd.Flags().GetString("auth"); filename != "" {
		return loginWithMode(con, filename, quiet)
	}

	// Only check Arg(0) as auth file for root command or login command
	// Avoid treating subcommand arguments (e.g., 'beacon' in 'build beacon') as auth file
	if cmd.Parent() == nil || cmd.Use == "client" || cmd.Use == "login" {
		if filename := cmd.Flags().Arg(0); strings.HasSuffix(filename, ".auth") {
			return loginWithMode(con, filename, quiet)
		}
	}
	files, err := assets.GetConfigs()
	if err != nil {
		return fmt.Errorf("error retrieving YAML files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no auth config found, maybe use `iom login [authfile.auth]` auto import")
	}
	// Create a model for the interactive list
	m := tui.NewSelect(files)
	m.Title = "Select User: "
	err = m.Run()
	if err != nil {
		con.Log.Errorf("Error running interactive list: %s", err)
		return err
	}

	// After the interactive list is completed, check the selected item
	if m.Selected != "" {
		tui.ClearLines(2)
		return loginWithMode(con, m.Selected, quiet)
	} else {
		return errors.New("no user selected")
	}
}

func loginWithMode(con *core.Console, authFile string, quiet bool) error {
	if !quiet {
		assets.PrintProfileSettings()
	}

	config, err := assets.LoadConfig(authFile)
	if err != nil {
		return err
	}
	err = core.LoginWithOptions(con, config, core.LoginOptions{
		SuppressStartupOutput: quiet,
	})
	if err != nil {
		return err
	}

	if fileutils.Exist(authFile) {
		err := assets.MvConfig(authFile)
		if err != nil {
			return err
		}
	}
	return nil
}
