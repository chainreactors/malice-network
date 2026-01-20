package build

import (
	"errors"
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func CheckSource(con *core.Console, buildConfig *clientpb.BuildConfig) (string, error) {
	source := buildConfig.Source
	if source != consts.ArtifactFromGithubAction &&
		source != consts.ArtifactFromDocker &&
		source != consts.ArtifactFromSaas &&
		source != "" {
		return source, errors.New("source '" + source + "' is invalid")
	}

	resp, err := con.Rpc.CheckSource(con.Context(), buildConfig)
	if err != nil {
		return "", err
	}
	return resp.Source, nil
}

// parseBasicConfig
func parseBasicConfig(cmd *cobra.Command, con *core.Console) (*clientpb.BuildConfig, error) {
	// init
	buildConfig := common.ParseGenerateFlags(cmd)

	if buildConfig.Target == "" {
		return nil, errors.New("require build target")
	}

	return buildConfig, nil
}

func parseSourceConfig(cmd *cobra.Command, con *core.Console, buildConfig *clientpb.BuildConfig) (*clientpb.BuildConfig, error) {
	source, _ := cmd.Flags().GetString("source")
	buildConfig.Source = source
	comment, _ := cmd.Flags().GetString("comment")
	if comment != "" {
		buildConfig.Comment = comment
	}
	// use github action
	actionConfig := common.ParseGithubFlags(cmd)
	if actionConfig != nil {
		buildConfig.SourceConfig = &clientpb.BuildConfig_GithubAction{
			GithubAction: actionConfig,
		}
	}
	source, err := CheckSource(con, buildConfig)
	if err != nil {
		return nil, err
	}
	buildConfig.Source = source
	return buildConfig, nil
}

// ExecuteBuild executes the build logic.
func ExecuteBuild(con *core.Console, buildConfig *clientpb.BuildConfig) error {
	artifact, err := con.Rpc.Build(con.Context(), buildConfig)
	if err != nil {
		return fmt.Errorf("build %s failed: %w", buildConfig.BuildType, err)
	}
	con.Log.Infof("Build started: %s (type: %s, target: %s, source: %s)\n",
		artifact.Name, artifact.Type, artifact.Target, artifact.Source)
	return nil
}

func BindCmd(cmd *cobra.Command, con *core.Console) error {
	buildConfig, err := prepareBuildConfig(cmd, con, consts.CommandBuildBind)
	if err != nil {
		return err
	}

	return ExecuteBuild(con, buildConfig)
}

// parseLibFlag sets buildConfig.Lib based on the --lib flag and validates compatibility with buildType/target.
func parseLibFlag(cmd *cobra.Command, buildConfig *clientpb.BuildConfig) error {
	libFlag, _ := cmd.Flags().GetBool("lib")
	return ValidateLibFlag(buildConfig, libFlag, cmd.Flags().Changed("lib"))
}

// ValidateLibFlag validates the lib flag and sets buildConfig.Lib.
func ValidateLibFlag(buildConfig *clientpb.BuildConfig, libFlag bool, libFlagChanged bool) error {
	target, ok := consts.GetBuildTarget(buildConfig.Target)
	if !ok {
		return errors.New("invalid target: " + buildConfig.Target)
	}

	switch buildConfig.BuildType {
	case consts.CommandBuildModules, consts.CommandBuild3rdModules:
		if libFlagChanged && !libFlag {
			return errors.New("modules build requires --lib")
		}
		if target.OS != consts.Windows {
			return errors.New("modules build only supports Windows targets")
		}
		buildConfig.Lib = true
	case consts.CommandBuildPrelude:
		if libFlag {
			return errors.New("prelude build does not support --lib")
		}
		buildConfig.Lib = false
	case consts.CommandBuildPulse:
		if libFlag {
			return errors.New("pulse build does not support --lib")
		}
		if target.OS != consts.Windows {
			return errors.New("pulse build only supports Windows targets")
		}
		buildConfig.Lib = false
	default:
		// beacon/bind allow both exe and lib
		buildConfig.Lib = libFlag
	}
	return nil
}

func BuildLogCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	num, _ := cmd.Flags().GetInt("limit")
	builder, err := con.Rpc.BuildLog(con.Context(), &clientpb.Artifact{
		Name:   name,
		LogNum: uint32(num),
	})
	if err != nil {
		return err
	}
	if len(builder.Log) == 0 {
		con.Log.Infof("No logs found for build name %s\n", name)
		return nil
	}
	con.Log.Console(string(builder.Log))
	return nil
}
