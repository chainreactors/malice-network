package extension

import (
	"encoding/json"
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/tui"
	"github.com/charmbracelet/bubbles/table"
	"io/ioutil"
	"strings"
)

// ExtensionsCmd - List information about installed extensions
func ExtensionsCmd(ctx *grumble.Context, con *console.Console) {
	if 0 < len(getInstalledManifests()) {
		PrintExtensions(con)
	} else {
		console.Log.Infof("No extensions installed, use the 'armory' command to automatically install some\n")
	}
}

// PrintExtensions - Print a list of loaded extensions
func PrintExtensions(con *console.Console) {
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		{Title: "Name", Width: 10},
		{Title: "Command Name", Width: 10},
		{Title: "Platforms", Width: 7},
		{Title: "Version", Width: 7},
		{Title: "Installed", Width: 4},
		{Title: "Extension Author", Width: 10},
		{Title: "Original Author", Width: 10},
		{Title: "Repository", Width: 20},
	})

	installedManifests := getInstalledManifests()
	for _, extension := range loadedExtensions {
		installed := ""
		if _, ok := installedManifests[extension.CommandName]; ok {
			installed = "✅"
		}
		row = table.Row{
			extension.Name,
			extension.CommandName,
			strings.Join(extensionPlatforms(extension), ",\n"),
			extension.Version,
			installed,
			extension.ExtensionAuthor,
			extension.OriginalAuthor,
			extension.RepoURL,
		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.Rows = rowEntries
	tableModel.SetRows()
	err := tui.Run(tableModel)
	if err != nil {
		return
	}
}

func extensionPlatforms(extension *ExtensionManifest) []string {
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
		installedManifests[manifest.CommandName] = manifest
	}
	return installedManifests
}

// ExtensionsCommandNameCompleter - Completer for installed extensions command names
func ExtensionsCommandNameCompleter(prefix string, args []string, con *console.Console) []string {
	installedManifests := getInstalledManifests()
	results := []string{}
	for _, manifest := range installedManifests {
		if strings.HasPrefix(manifest.CommandName, prefix) {
			results = append(results, manifest.CommandName)
		}
	}
	return results
}
