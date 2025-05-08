package mal

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core/plugin"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/mals/m"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"path/filepath"
)

var loadedMals = make(map[string]*LoadedMal)

type LoadedMal struct {
	Manifest *plugin.MalManiFest
	CMDs     []*cobra.Command
	Plugin   plugin.Plugin
}

func MalLoadCmd(ctx *cobra.Command, con *repl.Console) error {
	dirPath := ctx.Flags().Arg(0)
	mal, err := LoadMal(con, con.ImplantMenu(), filepath.Join(assets.GetMalsDir(), dirPath, m.ManifestFileName))
	if err != nil {
		return err
	}
	for _, cmd := range mal.CMDs {
		con.ImplantMenu().AddCommand(cmd)
		logs.Log.Debugf("add command: %s", cmd.Name())
	}
	return nil
}

func LoadMal(con *repl.Console, rootCmd *cobra.Command, filename string) (*LoadedMal, error) {
	manifest, err := plugin.LoadMalManiFest(filename)
	if err != nil {
		return nil, err
	}
	return LoadMalWithManifest(con, rootCmd, manifest)
}

func LoadMalWithManifest(con *repl.Console, rootCmd *cobra.Command, manifest *plugin.MalManiFest) (*LoadedMal, error) {
	plug, err := con.Plugins.LoadPlugin(manifest, con, rootCmd)
	if err != nil {
		return nil, err
	}
	for event, fn := range plug.GetEvents() {
		con.AddEventHook(event, fn)
	}
	profile, err := assets.GetProfile()
	if err != nil {
		return nil, err
	}
	profile.Add(manifest.Name)
	var cmdNames []string
	var cmds []*cobra.Command
	for _, cmd := range plug.Commands() {
		cmdNames = append(cmdNames, cmd.Command.Name())
		cmds = append(cmds, cmd.Command)
	}
	mal := &LoadedMal{
		Manifest: manifest,
		CMDs:     cmds,
		Plugin:   plug,
	}

	loadedMals[manifest.Name] = mal

	err = assets.SaveProfile(profile)
	if err != nil {
		return nil, err
	}
	con.Log.Importantf("load mal: %s successfully, register %v\n", manifest.Name, cmdNames)
	return mal, nil
}

func ListMalManifest(con *repl.Console) {
	if len(loadedMals) == 0 {
		con.Log.Infof("No mal loaded")
		return
	}
	rows := []table.Row{}
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("Name", "Name", 10),
		table.NewColumn("Type", "Type", 10),
		table.NewColumn("Version", "Version", 7),
		table.NewColumn("Author", "Author", 4),
		//{Title: "Name", Width: 10},
		//{Title: "Type", Width: 10},
		//{Title: "Version", Width: 7},
		//{Title: "Author", Width: 4},
	}, true)
	for _, m := range loadedMals {
		plug := m.Plugin.Manifest()
		row := table.NewRow(
			table.RowData{
				"Name":    plug.Name,
				"Type":    plug.Type,
				"Version": plug.Version,
				"Author":  plug.Author,
			},
		)
		//Row{
		//
		//	plug.Name,
		//	plug.Type,
		//	plug.Version,
		//	plug.Author,
		//}
		rows = append(rows, row)
	}
	tableModel.SetRows(rows)
	tableModel.SetMultiline()
	newTable := tui.NewModel(tableModel, nil, false, false)
	err := newTable.Run()
	if err != nil {
		con.Log.Errorf("Error running table: %s", err)
		return
	}
}
