package alias

import (
	"encoding/json"
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/styles"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
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
	tw := table.NewWriter()
	tw.SetStyle(styles.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Name",
		"Command Name",
		"Platforms",
		"Version",
		"Installed",
		".NET Assembly",
		"Reflective",
		"Tool Author",
		"Repository",
	})
	tw.SortBy([]table.SortBy{
		{Name: "Name", Mode: table.Asc},
	})
	tw.SetColumnConfigs([]table.ColumnConfig{
		{Number: 5, Align: text.AlignCenter},
	})

	installedManifests := getInstalledManifests()
	for _, aliasPkg := range loadedAliases {
		installed := ""
		if _, ok := installedManifests[aliasPkg.Manifest.CommandName]; ok {
			installed = "✅"
		}
		tw.AppendRow(table.Row{
			aliasPkg.Manifest.Name,
			aliasPkg.Manifest.CommandName,
			strings.Join(aliasPlatforms(aliasPkg.Manifest), ",\n"),
			aliasPkg.Manifest.Version,
			installed,
			aliasPkg.Manifest.IsAssembly,
			aliasPkg.Manifest.IsReflective,
			aliasPkg.Manifest.OriginalAuthor,
			aliasPkg.Manifest.RepoURL,
		})
	}
	console.Log.Console(tw.Render())
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
