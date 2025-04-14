package armory

import (
	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/extension"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
	"regexp"
)

// ArmorySearchCmd - Search for packages by name
func ArmorySearchCmd(cmd *cobra.Command, con *repl.Console) {
	con.Log.Infof("Refreshing package cache ... \n")
	clientConfig := parseArmoryHTTPConfig(cmd)
	refresh(clientConfig)
	tui.Clear()
	rawNameExpr := cmd.Flags().Arg(0)
	if rawNameExpr == "" {
		con.Log.Errorf("Please specify a search term!\n")
		return
	}
	nameExpr, err := regexp.Compile(rawNameExpr)
	if err != nil {
		con.Log.Errorf("Invalid regular expression: %s\n", err)
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
		con.Log.Infof("No packages found matching '%s'\n", rawNameExpr)
		return
	}
	isStatic, err := cmd.Flags().GetBool("static")
	if err != nil {
		con.Log.Errorf("Error getting static flag: %v", err)
		return
	}
	PrintArmoryPackages(matchedAliases, matchedExts, con, clientConfig, isStatic)
}
