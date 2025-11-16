package build

import (
	"errors"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/server/build"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"os"
)

func PreludeFlagSet(f *pflag.FlagSet) {
	f.String("autorun", "", "auto run zip path")
}

func PreludeCmd(cmd *cobra.Command, con *core.Console) error {
	//buildConfig, err := prepareBuildConfig(cmd, con, consts.CommandBuildPrelude)
	//if err != nil {
	//	return err
	//}
	autorunZipPath, _ := cmd.Flags().GetString("autorun")
	if autorunZipPath == "" {
		return errors.New("require autorun.zip path")
	}
	zipData, err := os.ReadFile(autorunZipPath)
	if err != nil {
		return err
	}
	buildConfig, err := build.ProcessAutorunZipFromBytes(zipData)
	if err != nil {
		return err
	}
	buildConfig, err = parseSourceConfig(cmd, con, buildConfig)
	if err != nil {
		return err
	}
	buildConfig.BuildType = consts.CommandBuildPrelude
	target, _ := cmd.Flags().GetString("target")
	buildConfig.Target = target

	executeBuild(con, buildConfig)
	return nil
}
