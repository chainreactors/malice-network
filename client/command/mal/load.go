package mal

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core/plugin"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"os"
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
	mal, err := LoadMal(con, con.ImplantMenu(), filepath.Join(assets.GetMalsDir(), dirPath, ManifestFileName))
	if err != nil {
		return err
	}
	for _, cmd := range mal.CMDs {
		con.ImplantMenu().AddCommand(cmd)
		logs.Log.Debugf("add command: %s", cmd.Name())
	}
	return nil
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

func LoadMal(con *repl.Console, rootCmd *cobra.Command, filename string) (*LoadedMal, error) {
	manifest, err := LoadMalManiFest(con, filename)
	if err != nil {
		return nil, err
	}
	plug, err := con.Plugins.LoadPlugin(manifest, con, rootCmd)
	if err != nil {
		return nil, err
	}
	for event, fn := range plug.GetEvents() {
		con.AddEventHook(event, fn)
	}
	profile := assets.GetProfile()
	profile.AddMal(manifest.Name)
	var cmdNames []string
	var cmds []*cobra.Command
	for _, cmd := range plug.Commands() {
		cmdNames = append(cmdNames, cmd.CMD.Name())
		cmds = append(cmds, cmd.CMD)
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
	con.Log.Importantf("load mal: %s successfully, register %v", filename, cmdNames)
	return mal, nil
}

func ListMalManiFest(con *repl.Console) {
	if len(loadedMals) == 0 {
		con.Log.Infof("No mal loaded")
		return
	}
	rows := []table.Row{}
	tableModel := tui.NewTable([]table.Column{
		{Title: "Name", Width: 10},
		{Title: "Type", Width: 10},
		{Title: "Version", Width: 7},
		{Title: "Author", Width: 4},
	}, true)
	for _, m := range loadedMals {
		plug := m.Plugin.Manifest()
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
		con.Log.Errorf("Error running table: %s", err)
		return
	}
}
