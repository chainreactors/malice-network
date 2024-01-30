package alias

import (
	"encoding/json"
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/styles"
	"github.com/charmbracelet/bubbles/table"
	"golang.org/x/term"
	"os"
	"strconv"
	"strings"
)

// AliasesCmd - The alias command
func AliasesCmd(ctx *grumble.Context, con *console.Console) error {
	if 0 < len(loadedAliases) {
		PrintAliases(con)
	} else {
		console.Log.Infof("No aliases installed, use the 'armory' command to automatically install some")
	}

	return nil
}

// PrintAliases - Print a list of loaded aliases
func PrintAliases(con *console.Console) {
	width, _, err := term.GetSize(0)
	var tableModel styles.TableModel
	var rowEntries []table.Row
	var row table.Row
	if err != nil {
		width = 99
	}
	if con.Settings.SmallTermWidth < width {
		tableModel = styles.TableModel{Columns: []table.Column{
			{Title: "Name", Width: 4},
			{Title: "Command Name", Width: 15},
			{Title: "Platforms", Width: 10},
			{Title: "Version", Width: 10},
			{Title: "Installed", Width: 10},
			{Title: ".NET Assembly", Width: 15},
			{Title: "Reflective", Width: 10},
			{Title: "Tool Author", Width: 15},
			{Title: "Repository", Width: 10},
		}}
	}

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
	tableModel.Rows = rowEntries
	err = tableModel.Run()
	if err != nil {
		console.Log.Errorf("Can't print aliases: %s", err)
	}
}

// AliasCommandNameCompleter - Completer for installed extensions command names
func AliasCommandNameCompleter(prefix string, args []string, con *console.Console) []string {
	results := []string{}
	for name := range loadedAliases {
		if strings.HasPrefix(name, prefix) {
			results = append(results, name)
		}
	}
	return results
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

// AliasCommandNameCompleter - Completer for installed extensions command names
//func AliasCompleter() carapace.Action {
//	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
//		results := []string{}
//		for name := range loadedAliases {
//			results = append(results, name)
//		}
//		return carapace.ActionValues(results...).Tag("aliases")
//	})
//}
