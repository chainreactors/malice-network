package build

import (
	"errors"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/profile"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"os"
)

func PreludeFlagSet(f *pflag.FlagSet) {
	f.String("autorun", "", "auto run zip path")
}

func PreludeCmd(cmd *cobra.Command, con *repl.Console) error {
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
	buildConfig, err := profile.ProcessAutorunZipFromBytes(zipData)
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
