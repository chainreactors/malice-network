package alias

import (
	"encoding/json"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"

	"github.com/carapace-sh/carapace"
	"github.com/spf13/cobra"
	"os"
	"strconv"
	"strings"
)

// AliasesCmd - The alias command
func AliasesCmd(cmd *cobra.Command, con *repl.Console) {
	if 0 < len(loadedAliases) {
		PrintAliases(con)
	} else {
		con.Log.Infof("No aliases installed, use the 'armory' command to automatically install some\n")
	}
}

// PrintAliases - Print a list of loaded aliases
func PrintAliases(con *repl.Console) {
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("Name", "Name", 10),
		table.NewColumn("Command Name", "Command Name", 15),
		table.NewColumn("Platforms", "Platforms", 10),
		table.NewColumn("Version", "Version", 10),
		table.NewColumn("Installed", "Installed", 10),
		table.NewColumn(".NET Assembly", ".NET Assembly", 15),
		table.NewColumn("Reflective", "Reflective", 10),
		table.NewColumn("Tool Author", "Tool Author", 20),
		table.NewColumn("Repository", "Repository", 20),
	}, true)

	installedManifests := getInstalledManifests()
	for _, aliasPkg := range loadedAliases {
		installed := ""
		if _, ok := installedManifests[aliasPkg.Manifest.CommandName]; ok {
			installed = "✅"
		}
		row = table.NewRow(
			table.RowData{
				"Name":          aliasPkg.Manifest.Name,
				"Command Name":  aliasPkg.Manifest.CommandName,
				"Platforms":     strings.Join(aliasPlatforms(aliasPkg.Manifest), ",\n"),
				"Version":       aliasPkg.Manifest.Version,
				"Installed":     installed,
				".NET Assembly": strconv.FormatBool(aliasPkg.Manifest.IsAssembly),
				"Reflective":    strconv.FormatBool(aliasPkg.Manifest.IsReflective),
				"Tool Author":   aliasPkg.Manifest.OriginalAuthor,
				"Repository":    aliasPkg.Manifest.RepoURL,
			})
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	newTable := tui.NewModel(tableModel, nil, false, false)
	err := newTable.Run()
	if err != nil {
		return
	}
}

// AliasCommandNameCompleter - Completer for installed extensions command names.
func AliasCompleter() carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		var results []string
		for name := range loadedAliases {
			results = append(results, name)
		}
		return carapace.ActionValues(results...).Tag("aliases")
	})
}

func aliasPlatforms(aliasPkg *AliasManifest) []string {
	platforms := map[string]string{}
	for _, entry := range aliasPkg.Files {
		platforms[fmt.Sprintf("%s/%s", entry.OS, entry.Arch)] = ""
	}
	keys := []string{}
	for key := range platforms {
		keys = append(keys, key)
	}
	return keys
}

func getInstalledManifests() map[string]*AliasManifest {
	manifestPaths := assets.GetInstalledAliasManifests()
	installedManifests := map[string]*AliasManifest{}
	for _, manifestPath := range manifestPaths {
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		manifest := &AliasManifest{}
		err = json.Unmarshal(data, manifest)
		if err != nil {
			continue
		}
		installedManifests[manifest.CommandName] = manifest
	}
	return installedManifests
}
