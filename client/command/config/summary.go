package config

import (
	"fmt"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
)

// ConfigSummaryCmd displays a summary table of all config modules.
func ConfigSummaryCmd(con *core.Console) error {
	settings, err := assets.LoadSettings()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	var rowEntries []table.Row

	// MCP
	rowEntries = append(rowEntries, table.NewRow(table.RowData{
		"Module": "MCP",
		"Status": renderEnabled(settings.McpEnable),
		"Detail": mcpDetail(con, settings),
	}))

	// LocalRPC
	rowEntries = append(rowEntries, table.NewRow(table.RowData{
		"Module": "LocalRPC",
		"Status": renderEnabled(settings.LocalRPCEnable),
		"Detail": localrpcDetail(con, settings),
	}))

	// AI
	aiStatus, aiDetail := aiSummary(settings)
	rowEntries = append(rowEntries, table.NewRow(table.RowData{
		"Module": "AI",
		"Status": aiStatus,
		"Detail": aiDetail,
	}))

	// Github (RPC-based, may fail if not connected)
	githubStatus, githubDetail := githubSummary(con)
	rowEntries = append(rowEntries, table.NewRow(table.RowData{
		"Module": "Github",
		"Status": githubStatus,
		"Detail": githubDetail,
	}))

	// Notify (RPC-based)
	notifyStatus, notifyDetail := notifySummary(con)
	rowEntries = append(rowEntries, table.NewRow(table.RowData{
		"Module": "Notify",
		"Status": notifyStatus,
		"Detail": notifyDetail,
	}))

	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("Module", "Module", 12),
		table.NewColumn("Status", "Status", 14),
		table.NewFlexColumn("Detail", "Detail", 1),
	}, true)

	tableModel.SetRows(rowEntries)
	con.Log.Console(tableModel.View() + "\n")
	return nil
}

func renderEnabled(enabled bool) string {
	if enabled {
		return tui.GreenFg.Render("Enabled")
	}
	return tui.RedFg.Render("Disabled")
}

func mcpDetail(con *core.Console, settings *assets.Settings) string {
	detail := settings.McpAddr
	if con.MCP != nil {
		detail += " " + tui.GreenFg.Render("(Running)")
	}
	return detail
}

func localrpcDetail(con *core.Console, settings *assets.Settings) string {
	detail := settings.LocalRPCAddr
	if con.LocalRPC != nil {
		detail += " " + tui.GreenFg.Render("(Running)")
	}
	return detail
}

func aiSummary(settings *assets.Settings) (string, string) {
	if settings.AI == nil || !settings.AI.Enable {
		return tui.RedFg.Render("Disabled"), ""
	}
	return tui.GreenFg.Render("Enabled"), fmt.Sprintf("%s / %s", settings.AI.Provider, settings.AI.Model)
}

func githubSummary(con *core.Console) (string, string) {
	if con.Rpc == nil {
		return tui.DarkGrayFg.Render("N/A"), "not connected"
	}
	resp, err := con.Rpc.GetGithubConfig(con.Context(), &clientpb.Empty{})
	if err != nil || resp.Owner == "" {
		return tui.DarkGrayFg.Render("Not Set"), ""
	}
	return tui.GreenFg.Render("Configured"), fmt.Sprintf("%s/%s", resp.Owner, resp.Repo)
}

func notifySummary(con *core.Console) (string, string) {
	if con.Rpc == nil {
		return tui.DarkGrayFg.Render("N/A"), "not connected"
	}
	notify, err := con.Rpc.GetNotifyConfig(con.Context(), &clientpb.Empty{})
	if err != nil {
		return tui.DarkGrayFg.Render("N/A"), ""
	}
	providers := notifyEnabledProviders(notify)
	if providers == "None" {
		return tui.DarkGrayFg.Render("Not Set"), ""
	}
	return tui.GreenFg.Render("Active"), providers
}
