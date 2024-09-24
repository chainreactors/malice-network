package alias

import (
	"encoding/json"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/rsteube/carapace"
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
		con.Log.Infof("No aliases installed, use the 'armory' command to automatically install some")
	}
}

// PrintAliases - Print a list of loaded aliases
func PrintAliases(con *repl.Console) {
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		{Title: "Name", Width: 10},
		{Title: "Command Name", Width: 15},
		{Title: "Platforms", Width: 10},
		{Title: "Version", Width: 10},
		{Title: "Installed", Width: 10},
		{Title: ".NET Assembly", Width: 15},
		{Title: "Reflective", Width: 10},
		{Title: "Tool Author", Width: 20},
		{Title: "Repository", Width: 20},
	}, true)

	installedManifests := getInstalledManifests()
	for _, aliasPkg := range loadedAliases {
		installed := ""
		if _, ok := installedManifests[aliasPkg.Manifest.CommandName]; ok {
			installed = "✅"
		}
		row = table.Row{
			aliasPkg.Manifest.Name,
			aliasPkg.Manifest.CommandName,
			strings.Join(aliasPlatforms(aliasPkg.Manifest), ",\n"),
			aliasPkg.Manifest.Version,
			installed,
			strconv.FormatBool(aliasPkg.Manifest.IsAssembly),
			strconv.FormatBool(aliasPkg.Manifest.IsReflective),
			aliasPkg.Manifest.OriginalAuthor,
			aliasPkg.Manifest.RepoURL,
		}
		rowEntries = append(rowEntries, row)
	}
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
