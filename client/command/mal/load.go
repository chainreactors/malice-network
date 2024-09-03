package mal

import (
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/core/plugin"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

func MalLoadCmd(ctx *cobra.Command, con *console.Console) {
	dirPath := ctx.Flags().Arg(0)
	_, err := LoadMalManiFest(con, filepath.Join(assets.GetMalsDir(), dirPath, ManifestFileName))
	if err != nil {
		console.Log.Error(err)
	}
}

func LoadMalManiFest(con *console.Console, filename string) (*plugin.MalManiFest, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	manifest, err := ParseMalManifest(content)
	if err != nil {
		return nil, err
	}

	err = con.Plugins.LoadPlugin(manifest, con)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func ListMalManiFest(con *console.Console) {
	rows := []table.Row{}
	tableModel := tui.NewTable([]table.Column{
		{Title: "Name", Width: 10},
		{Title: "Type", Width: 10},
		{Title: "Version", Width: 7},
		{Title: "Author", Width: 4},
	}, true)

	for _, plug := range con.Plugins.Plugins {
		row := table.Row{
			plug.Name,
			plug.Type,
			plug.Version,
			plug.Author,
		}
		rows = append(rows, row)
	}
	tableModel.SetRows(rows)
	newTable := tui.NewModel(tableModel, nil, false, false)
	err := newTable.Run()
	if err != nil {
		console.Log.Errorf("Error running table: %s", err)
		return
	}
}
