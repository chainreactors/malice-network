package build

import (
	"errors"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"strings"
)

// BeaconFlagSet 定义所有构建相关的flag
func ModuleFlagSet(f *pflag.FlagSet) {
	f.String("modules", "", "Override modules (comma-separated, e.g., 'full,execute_exe')")
	f.String("3rd", "", "Override 3rd party modules")
	common.SetFlagSetGroup(f, "module")
}

func ModulesCmd(cmd *cobra.Command, con *core.Console) error {
	var err error
	//buildConfig, err := prepareBuildConfig(cmd, con, consts.CommandBuildModules)
	buildConfig, err := parseBasicConfig(cmd, con)
	buildConfig.BuildType = consts.CommandBuildModules
	if err != nil {
		return err
	}
	// config and check source
	buildConfig, err = parseSourceConfig(cmd, con, buildConfig)
	if err != nil {
		return err
	}
	if err := parseLibFlag(cmd, buildConfig); err != nil {
		return err
	}
	//
	modules, _ := cmd.Flags().GetString("modules")
	thirdModules, _ := cmd.Flags().GetString("3rd")
	if modules == "" && thirdModules == "" {
		return errors.New("--module and --3rd options are mutually exclusive. please specify only one of them")
	}
	// set profile about modules
	mainProfile := implanttypes.ProfileConfig{}
	mainProfile.SetDefaults()
	if thirdModules != "" {
		mainProfile.Implant.ThirdModules = strings.Split(thirdModules, ",")
		mainProfile.Implant.Enable3rd = true
	} else {
		mainProfile.Implant.Modules = strings.Split(modules, ",")
	}
	buildConfig.MaleficConfig, err = mainProfile.ToYAML()
	if err != nil {
		return err
	}

	return executeBuild(con, buildConfig)
}
