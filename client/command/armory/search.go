package armory

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/extension"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/tui"
	"regexp"
)

// ArmorySearchCmd - Search for packages by name
func ArmorySearchCmd(ctx *grumble.Context, con *console.Console) {
	console.Log.Infof("Refreshing package cache ... ")
	clientConfig := parseArmoryHTTPConfig(ctx)
	refresh(clientConfig)
	tui.Clear()
	rawNameExpr := ctx.Args.String("name")
	if rawNameExpr == "" {
		console.Log.Errorf("Please specify a search term!\n")
		return
	}
	nameExpr, err := regexp.Compile(rawNameExpr)
	if err != nil {
		console.Log.Errorf("Invalid regular expression: %s\n", err)
		return
	}

	aliases, exts := packageManifestsInCache()
	matchedAliases := []*alias.AliasManifest{}
	for _, a := range aliases {
		if nameExpr.MatchString(a.CommandName) {
			matchedAliases = append(matchedAliases, a)
		}
	}
	matchedExts := []*extension.ExtensionManifest{}
	for _, extm := range exts {
		for _, ext := range extm.ExtCommand {
			if nameExpr.MatchString(ext.CommandName) {
				matchedExts = append(matchedExts, extm)
			}
		}
	}
	if len(matchedAliases) == 0 && len(matchedExts) == 0 {
		console.Log.Infof("No packages found matching '%s'\n", rawNameExpr)
		return
	}
	PrintArmoryPackages(matchedAliases, matchedExts, con, clientConfig)
}
