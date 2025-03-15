package extension

import (
	"encoding/json"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"

	"github.com/carapace-sh/carapace"
	"github.com/spf13/cobra"
	"io/ioutil"
	"strings"
)

// ExtensionsCmd - List information about installed extensions
func ExtensionsCmd(cmd *cobra.Command, con *repl.Console) {
	if 0 < len(getInstalledManifests()) {
		PrintExtensions(con)
	} else {
		con.Log.Infof("No extensions installed, use the 'armory' command to automatically install some\n")
	}
}

// PrintExtensions - Print a list of loaded extensions
func PrintExtensions(con *repl.Console) {
	var rowEntries []table.Row

	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("Name", "Name", 10),
		table.NewColumn("Command Name", "Command Name", 10),
		table.NewColumn("Platforms", "Platforms", 7),
		table.NewColumn("Version", "Version", 7),
		table.NewColumn("Installed", "Installed", 4),
		table.NewColumn("Extension Author", "Extension Author", 10),
		table.NewColumn("Original Author", "Original Author", 10),
		table.NewColumn("Repository", "Repository", 20),
	}, true)

	installedManifests := getInstalledManifests()
	for _, ext := range loadedExtensions {
		installed := ""
		if _, ok := installedManifests[ext.Manifest.CommandName]; ok {
			installed = "✅"
		}
		row := table.NewRow(
			table.RowData{
				"Name":             ext.Manifest.Manifest.Name,
				"Command Name":     ext.Manifest.CommandName,
				"Platforms":        strings.Join(extensionPlatforms(ext.Manifest), ",\n"),
				"Version":          ext.Manifest.Manifest.Version,
				"Installed":        installed,
				"Extension Author": ext.Manifest.Manifest.ExtensionAuthor,
				"Original Author":  ext.Manifest.Manifest.OriginalAuthor,
				"Repository":       ext.Manifest.Manifest.RepoURL,
			})
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	newTable := tui.NewModel(tableModel, nil, false, false)
	err := newTable.Run()
	if err != nil {
		con.Log.Errorf("Error running table: %s", err)
		return
	}
}

func extensionPlatforms(extension *ExtCommand) []string {
	platforms := map[string]string{}
	for _, entry := range extension.Files {
		platforms[fmt.Sprintf("%s/%s", entry.OS, entry.Arch)] = ""
	}
	keys := []string{}
	for key := range platforms {
		keys = append(keys, key)
	}
	return keys
}

func getInstalledManifests() map[string]*ExtensionManifest {
	manifestPaths := assets.GetInstalledExtensionManifests()
	installedManifests := map[string]*ExtensionManifest{}
	for _, manifestPath := range manifestPaths {
		data, err := ioutil.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		manifest := &ExtensionManifest{}
		err = json.Unmarshal(data, manifest)
		if err != nil {
			continue
		}
		installedManifests[manifest.Name] = manifest
	}
	return installedManifests
}

// ExtensionsCommandNameCompleter - Completer for installed extensions command names.
func ExtensionsCommandNameCompleter(con *repl.Console) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		results := []string{}
		for _, manifest := range loadedExtensions {
			results = append(results, manifest.Manifest.CommandName)
			results = append(results, manifest.Manifest.Help)
		}

		return carapace.ActionValuesDescribed(results...).Tag("extension commands")
	})
}
