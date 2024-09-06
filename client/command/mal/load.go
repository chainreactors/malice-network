package mal

import (
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core/plugin"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

func MalLoadCmd(ctx *cobra.Command, con *repl.Console) {
	dirPath := ctx.Flags().Arg(0)
	_, err := LoadMal(con, filepath.Join(assets.GetMalsDir(), dirPath, ManifestFileName))
	if err != nil {
		repl.Log.Error(err)
		return
	}
}

func LoadMalManiFest(con *repl.Console, filename string) (*plugin.MalManiFest, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	manifest, err := ParseMalManifest(content)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}

func LoadMal(con *repl.Console, filename string) (*plugin.MalManiFest, error) {
	manifest, err := LoadMalManiFest(con, filename)
	plug, err := con.Plugins.LoadPlugin(manifest, con)
	if err != nil {
		return nil, err
	}

	err = plug.ReverseRegisterLuaFunctions(con.App.Menu(consts.ImplantMenu).Command)
	if err != nil {
		return nil, err
	}
	var cmds []string
	for _, cmd := range plug.CMDs {
		cmds = append(cmds, cmd.Name())
	}
	repl.Log.Importantf("load mal: %s successfully, register %v", filename, cmds)
	return manifest, nil
}

func ListMalManiFest(con *repl.Console) {
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
		repl.Log.Errorf("Error running table: %s", err)
		return
	}
}
