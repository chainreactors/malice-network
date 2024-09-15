package armory

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/extension"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/client/utils"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"os"
	"sort"
	"strconv"
	"strings"
)

type VersionInformation struct {
	OldVersion string
	NewVersion string
	ArmoryName string
}

type PackageType uint

const (
	AliasPackage PackageType = iota
	ExtensionPackage
)

type UpdateIdentifier struct {
	Type PackageType
	Name string
}

// ArmoryUpdateCmd - Update all installed extensions/aliases
func ArmoryUpdateCmd(cmd *cobra.Command, con *repl.Console) {
	var selectedUpdates []UpdateIdentifier
	var err error

	con.Log.Infof("Refreshing package cache ... ")
	clientConfig := parseArmoryHTTPConfig(cmd)
	refresh(clientConfig)
	tui.Clear()

	armoryName, _ := cmd.Flags().GetString("armory")

	// Find PK for the armory name
	armoryPK := getArmoryPublicKey(armoryName)

	// Check for updates
	if armoryName == "" {
		con.Log.Warnf("Could not find a configured armory named %q - searching all configured armories\n\n",
			armoryName)
	}

	// Check packages for updates
	aliasUpdates := checkForAliasUpdates(armoryPK)
	extUpdates := checkForExtensionUpdates(armoryPK)

	// Display a table of results
	if len(aliasUpdates) > 0 || len(extUpdates) > 0 {
		updateKeys := sortUpdateIdentifiers(aliasUpdates, extUpdates)
		displayAvailableUpdates(updateKeys, aliasUpdates, extUpdates)
		selectedUpdates, err = getUpdatesFromUser(updateKeys)
		if err != nil {
			con.Log.Errorf(err.Error() + "\n")
			return
		}
		if len(selectedUpdates) == 0 {
			return
		}
	} else {
		con.Log.Infof("All packages are up to date")
		return
	}

	for _, update := range selectedUpdates {
		switch update.Type {
		case AliasPackage:
			aliasVersionInfo, ok := aliasUpdates[update.Name]
			if !ok {
				continue
			}
			updatePackage, err := getPackageForCommand(update.Name, armoryPK, aliasVersionInfo.NewVersion)
			if err != nil {
				con.Log.Errorf("Could not get update package for alias %s: %s\n", update.Name, err)
				continue
			}
			err = installAliasPackage(updatePackage, false, clientConfig, con)
			if err != nil {
				con.Log.Errorf("Failed to update %s: %s\n", update.Name, err)
			}
		case ExtensionPackage:
			extVersionInfo, ok := extUpdates[update.Name]
			if !ok {
				continue
			}
			updatedPackage, err := getPackageForCommand(update.Name, armoryPK, extVersionInfo.NewVersion)
			if err != nil {
				con.Log.Errorf("Could not get update package for extension %s: %s\n", update.Name, err)
				continue
			}
			err = installExtensionPackage(updatedPackage, false, clientConfig, con)
			if err != nil {
				con.Log.Errorf("Failed to update %s: %s\n", update.Name, err)
			}
		default:
			continue
		}
	}
}

func checkForAliasUpdates(armoryPK string) map[string]VersionInformation {
	cachedAliases, _ := packageManifestsInCache()
	results := make(map[string]VersionInformation)
	for _, aliasManifestPath := range assets.GetInstalledAliasManifests() {
		data, err := os.ReadFile(aliasManifestPath)
		if err != nil {
			continue
		}
		localManifest, err := alias.ParseAliasManifest(data)
		if err != nil {
			continue
		}
		for _, latestAlias := range cachedAliases {
			if latestAlias.CommandName == localManifest.CommandName && latestAlias.Version > localManifest.Version {
				if latestAlias.ArmoryPK == armoryPK || armoryPK == "" {
					results[localManifest.CommandName] = VersionInformation{
						OldVersion: localManifest.Version,
						NewVersion: latestAlias.Version,
						ArmoryName: latestAlias.ArmoryName,
					}
				}
			}
		}
	}
	return results
}

func checkForExtensionUpdates(armoryPK string) map[string]VersionInformation {
	_, cachedExtensions := packageManifestsInCache()
	results := make(map[string]VersionInformation)
	for _, extManifestPath := range assets.GetInstalledExtensionManifests() {
		data, err := os.ReadFile(extManifestPath)
		if err != nil {
			continue
		}
		localManifest, err := extension.ParseExtensionManifest(data)
		if err != nil {
			continue
		}
		for _, latestExt := range cachedExtensions {
			if latestExt.Name == localManifest.Name && latestExt.Version > localManifest.Version {
				if latestExt.ArmoryPK == armoryPK || armoryPK == "" {
					results[localManifest.Name] = VersionInformation{
						OldVersion: localManifest.Version,
						NewVersion: latestExt.Version,
						ArmoryName: latestExt.ArmoryName,
					}
				}
			}
		}
	}
	return results
}

func sortUpdateIdentifiers(aliasUpdates, extensionUpdates map[string]VersionInformation) []UpdateIdentifier {
	/*
		This function helps us keep updates straight when the user chooses from them. Just in case
		an alias and an extension exist with the same name, we cannot simply combine the two maps.

		We will assume that no two aliases and no two extensions have the same name.
	*/

	result := []UpdateIdentifier{}

	aliasNames := utils.Keys(aliasUpdates)
	extensionNames := utils.Keys(extensionUpdates)
	for _, name := range aliasNames {
		result = append(result, UpdateIdentifier{
			Type: AliasPackage,
			Name: name,
		})
	}
	for _, name := range extensionNames {
		result = append(result, UpdateIdentifier{
			Type: ExtensionPackage,
			Name: name,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

func displayAvailableUpdates(updateKeys []UpdateIdentifier,
	aliasUpdates, extensionUpdates map[string]VersionInformation) {
	var (
		aliasSuffix     string
		extensionSuffix string
		title           = "Available Updates (%d alias%s, %d extension%s)"
		rowEntries      []table.Row
		row             table.Row
	)

	tableModel := tui.NewTable([]table.Column{
		{Title: "Package Name", Width: 20},
		{Title: "Package Type", Width: 15},
		{Title: "Installed Version", Width: 20},
		{Title: "Available Version", Width: 20},
	}, true)
	tableModel.Title = fmt.Sprintf(title, len(aliasUpdates), aliasSuffix, len(extensionUpdates), extensionSuffix)
	if len(aliasUpdates) != 1 {
		aliasSuffix = "es"
	}
	if len(extensionUpdates) != 1 {
		extensionSuffix = "s"
	}
	for _, key := range updateKeys {
		var (
			packageName    string
			packageType    string
			packageVersion VersionInformation
			ok             bool
		)
		switch key.Type {
		case AliasPackage:
			packageVersion, ok = aliasUpdates[key.Name]
			if !ok {
				continue
			}
			packageName = key.Name
			packageType = "Alias"
		case ExtensionPackage:
			packageVersion, ok = extensionUpdates[key.Name]
			if !ok {
				continue
			}
			packageName = key.Name
			packageType = "Extension"
		default:
			continue
		}
		row = table.Row{
			packageName,
			packageType,
			packageVersion.OldVersion,
			fmt.Sprintf("%s (Armory: %s)", packageVersion.NewVersion, packageVersion.ArmoryName),
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

func getUpdatesFromUser(updateKeys []UpdateIdentifier) (chosenUpdates []UpdateIdentifier, selectionError error) {
	chosenUpdates = []UpdateIdentifier{}

	var updateResponse string
	title := fmt.Sprintf("You can apply all, none, or some updates.\nTo apply some updates, " +
		"specify the number of a single update, a range (1-3), or a combination of the two (1, 3-5, 7)\n" +
		"Which updates would you like to apply? [A]ll, [N]one, or some:")
	inputModel := tui.NewInput(title)
	inputModel.SetHandler(func() {
		updateResponse = inputModel.TextInput.Value()
	})
	newInput := tui.NewModel(inputModel, nil, false, true)
	err := newInput.Run()
	if err != nil {
		core.Log.Errorf("failed to get user input: %s", err)
		return
	}
	updateResponse = strings.ToLower(updateResponse)
	updateResponse = strings.Replace(updateResponse, " ", "", -1)
	if updateResponse == "n" || updateResponse == "none" {
		return
	}

	if updateResponse == "a" || updateResponse == "all" {
		chosenUpdates = updateKeys
		return
	}

	selections := strings.Split(updateResponse, ",")

	for _, selection := range selections {
		if strings.Contains(selection, "-") {
			rangeParts := strings.Split(selection, "-")
			start, err := strconv.Atoi(rangeParts[0])
			if err != nil {
				selectionError = fmt.Errorf("%s is not a valid range", rangeParts[0])
				return
			}
			end, err := strconv.Atoi(rangeParts[1])
			if err != nil {
				selectionError = fmt.Errorf("%s is not a valid range", rangeParts[1])
				return
			}
			// Adjust for the 0 indexed slice we are working with
			start -= 1
			end -= 1
			if start < 0 {
				start = 0
			}
			if start > end {
				selectionError = fmt.Errorf("%s is not a valid range", selection)
				return
			}
			if end >= len(updateKeys) {
				end = len(updateKeys) - 1
			}

			for i := start; i <= end; i++ {
				chosenUpdates = append(chosenUpdates, updateKeys[i])
			}
		} else {
			// Single entry
			index, err := strconv.Atoi(selection)
			if err != nil {
				selectionError = fmt.Errorf("%s is not a valid range", selection)
				return
			}
			index -= 1
			if index >= 0 && index < len(updateKeys) {
				chosenUpdates = append(chosenUpdates, updateKeys[index])
			}
		}
	}
	return
}
