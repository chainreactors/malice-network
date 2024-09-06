package extension

import (
	"encoding/json"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"io/ioutil"
	"strings"
)

// ExtensionsCmd - List information about installed extensions
func ExtensionsCmd(cmd *cobra.Command, con *repl.Console) {
	if 0 < len(getInstalledManifests()) {
		PrintExtensions(con)
	} else {
		repl.Log.Infof("No extensions installed, use the 'armory' command to automatically install some\n")
	}
}

// PrintExtensions - Print a list of loaded extensions
func PrintExtensions(con *repl.Console) {
	var rowEntries []table.Row

	tableModel := tui.NewTable([]table.Column{
		{Title: "Name", Width: 10},
		{Title: "Command Name", Width: 10},
		{Title: "Platforms", Width: 7},
		{Title: "Version", Width: 7},
		{Title: "Installed", Width: 4},
		{Title: "Extension Author", Width: 10},
		{Title: "Original Author", Width: 10},
		{Title: "Repository", Width: 20},
	}, true)

	installedManifests := getInstalledManifests()
	for _, ext := range loadedExtensions {
		installed := ""
		if _, ok := installedManifests[ext.CommandName]; ok {
			installed = "✅"
		}
		row := table.Row{
			ext.Manifest.Name,
			ext.CommandName,
			strings.Join(extensionPlatforms(ext), ",\n"),
			ext.Manifest.Version,
			installed,
			ext.Manifest.ExtensionAuthor,
			ext.Manifest.OriginalAuthor,
			ext.Manifest.RepoURL,
		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetRows(rowEntries)
	newTable := tui.NewModel(tableModel, nil, false, false)
	err := newTable.Run()
	if err != nil {
		repl.Log.Errorf("Error running table: %s", err)
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
			results = append(results, manifest.CommandName)
			results = append(results, manifest.Help)
		}

		return carapace.ActionValuesDescribed(results...).Tag("extension commands")
	})
}
