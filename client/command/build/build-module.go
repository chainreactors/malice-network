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
	if err != nil {
		return err
	}
	buildConfig.BuildType = consts.CommandBuildModules

	modules, _ := cmd.Flags().GetString("modules")
	thirdModules, _ := cmd.Flags().GetString("3rd")
	if modules == "" && thirdModules == "" {
		return errors.New("one of --modules or --3rd must be specified")
	}
	if modules != "" && thirdModules != "" {
		return errors.New("--modules and --3rd options are mutually exclusive. please specify only one of them")
	}
	// config and check source
	buildConfig, err = parseSourceConfig(cmd, con, buildConfig)
	if err != nil {
		return err
	}
	if err := parseOutputType(cmd, buildConfig); err != nil {
		return err
	}
	// set profile about modules
	buildConfig.MaleficConfig, err = BuildModuleMaleficConfig(splitModuleList(modules), splitModuleList(thirdModules))
	if err != nil {
		return err
	}

	return ExecuteBuild(con, buildConfig)
}

func BuildModuleMaleficConfig(modules, thirdModules []string) ([]byte, error) {
	modules = normalizeModuleList(modules)
	thirdModules = normalizeModuleList(thirdModules)

	if len(modules) == 0 && len(thirdModules) == 0 {
		return nil, errors.New("one of --modules or --3rd must be specified")
	}
	if len(modules) != 0 && len(thirdModules) != 0 {
		return nil, errors.New("--modules and --3rd options are mutually exclusive. please specify only one of them")
	}

	mainProfile := implanttypes.ProfileConfig{}
	mainProfile.SetDefaults()
	if mainProfile.Implant == nil {
		mainProfile.Implant = &implanttypes.ImplantProfile{}
	}
	mainProfile.Implant.Modules = nil
	mainProfile.Implant.ThirdModules = nil
	mainProfile.Implant.Enable3rd = false
	if len(thirdModules) != 0 {
		mainProfile.Implant.ThirdModules = thirdModules
		mainProfile.Implant.Enable3rd = true
	} else {
		mainProfile.Implant.Modules = modules
	}
	return mainProfile.ToYAML()
}

func splitModuleList(raw string) []string {
	if raw == "" {
		return nil
	}
	return normalizeModuleList(strings.Split(raw, ","))
}

func normalizeModuleList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	return result
}
