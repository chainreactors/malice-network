package generic

import (
	"errors"
	"fmt"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/tui"
	"github.com/gookit/config/v2"
	"github.com/spf13/cobra"
)

func LoginCmd(cmd *cobra.Command, con *core.Console) error {
	var err error

	// 处理 --mcp flag
	mcpAddr, _ := cmd.Flags().GetString("mcp")
	if mcpAddr != "" {
		con.Log.Infof("Enabling MCP server at %s", mcpAddr)
		err := enableMCPFromFlag(mcpAddr)
		if err != nil {
			con.Log.Errorf("Failed to enable MCP: %s", err)
		} else {
			con.Log.Importantf("MCP enabled, will start at %s after login", mcpAddr)
		}
	}

	// 处理 --rpc flag
	rpcAddr, _ := cmd.Flags().GetString("rpc")
	if rpcAddr != "" {
		con.Log.Infof("Enabling local gRPC server at %s", rpcAddr)
		err := enableLocalRPCFromFlag(rpcAddr)
		if err != nil {
			con.Log.Errorf("Failed to enable local RPC: %s", err)
		} else {
			con.Log.Importantf("Local RPC enabled, will start at %s after login", rpcAddr)
		}
	}

	if filename := cmd.Flags().Arg(0); filename != "" {
		return Login(con, filename)
	} else if filename, _ := cmd.Flags().GetString("auth"); filename != "" {
		return Login(con, filename)
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
		return Login(con, m.Selected)
	} else {
		return errors.New("no user selected")
	}
}

// enableMCPFromFlag 从命令行 flag 启用 MCP
func enableMCPFromFlag(addr string) error {
	// 使用 config.Set 来设置配置，会自动触发保存
	config.Set("settings.mcp_enable", true)
	config.Set("settings.mcp_addr", addr)
	return nil
}

// enableLocalRPCFromFlag 从命令行 flag 启用 Local RPC
func enableLocalRPCFromFlag(addr string) error {
	// 使用 config.Set 来设置配置，会自动触发保存
	config.Set("settings.localrpc_enable", true)
	config.Set("settings.localrpc_addr", addr)
	return nil
}

func Login(con *core.Console, authFile string) error {
	// 显示配置信息
	assets.PrintProfileSettings()

	config, err := assets.LoadConfig(authFile)
	if err != nil {
		return err
	}
	err = core.Login(con, config)
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
