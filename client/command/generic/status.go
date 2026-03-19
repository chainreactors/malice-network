package generic

import (
	"fmt"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/plugin"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func StatusCommand(con *core.Console) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show runtime status overview",
		RunE: func(cmd *cobra.Command, args []string) error {
			return StatusCmd(con)
		},
		Annotations: map[string]string{
			"static": "true",
		},
	}
}

func StatusCmd(con *core.Console) error {
	if con.Server == nil {
		con.Log.Console("Not connected to server\n")
		return nil
	}

	settings, _ := assets.LoadSettings()
	if settings == nil {
		settings = &assets.Settings{}
	}

	// Server info
	serverValues := map[string]string{
		"Client":  con.Client.Name,
		"Version": con.Info.Version,
		"Auth":    con.ConfigPath,
	}
	serverKeys := []string{"Client", "Version", "Auth"}
	con.Log.Console(common.NewKVTable("Server", serverKeys, serverValues).View() + "\n")

	// Resources
	var alive int
	for _, s := range con.Sessions {
		if s.IsAlive {
			alive++
		}
	}
	total := len(con.Sessions)

	var pipelineCount int
	for _, l := range con.Listeners {
		pipelineCount += len(l.Pipelines.GetPipelines())
	}

	mm := plugin.GetGlobalMalManager()
	embeddedCount := len(mm.GetAllEmbeddedPlugins())
	externalCount := len(mm.GetAllExternalPlugins())

	var rowEntries []table.Row
	rowEntries = append(rowEntries, table.NewRow(table.RowData{
		"Resource": "Sessions",
		"Count":    fmt.Sprintf("%d", total),
		"Detail":   fmt.Sprintf("%s alive, %s dead", tui.GreenFg.Render(fmt.Sprintf("%d", alive)), tui.RedFg.Render(fmt.Sprintf("%d", total-alive))),
	}))
	rowEntries = append(rowEntries, table.NewRow(table.RowData{
		"Resource": "Listeners",
		"Count":    fmt.Sprintf("%d", len(con.Listeners)),
		"Detail":   fmt.Sprintf("%d pipelines", pipelineCount),
	}))
	rowEntries = append(rowEntries, table.NewRow(table.RowData{
		"Resource": "Clients",
		"Count":    fmt.Sprintf("%d", len(con.Clients)),
		"Detail":   "",
	}))
	rowEntries = append(rowEntries, table.NewRow(table.RowData{
		"Resource": "Mals",
		"Count":    fmt.Sprintf("%d", embeddedCount+externalCount),
		"Detail":   fmt.Sprintf("%d embedded, %d external", embeddedCount, externalCount),
	}))

	resourceTable := tui.NewTable([]table.Column{
		table.NewColumn("Resource", "Resource", 12),
		table.NewColumn("Count", "Count", 8),
		table.NewFlexColumn("Detail", "Detail", 1),
	}, true)
	resourceTable.SetRows(rowEntries)
	con.Log.Console(resourceTable.View() + "\n")

	// Services
	serviceValues := map[string]string{
		"MCP":      serviceStatus(con.MCP != nil, settings.McpEnable, settings.McpAddr),
		"LocalRPC": serviceStatus(con.LocalRPC != nil, settings.LocalRPCEnable, settings.LocalRPCAddr),
	}
	serviceKeys := []string{"MCP", "LocalRPC"}
	con.Log.Console(common.NewKVTable("Services", serviceKeys, serviceValues).View() + "\n")

	return nil
}

func serviceStatus(running, enabled bool, addr string) string {
	if running {
		return tui.GreenFg.Render("Running") + " (" + addr + ")"
	}
	if enabled {
		return tui.YellowFg.Render("Enabled") + " (" + addr + ")"
	}
	return tui.RedFg.Render("Disabled")
}
