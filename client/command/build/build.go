package build

import (
	"errors"
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func CheckSource(con *core.Console, buildConfig *clientpb.BuildConfig) (string, error) {
	if buildConfig == nil {
		buildConfig = &clientpb.BuildConfig{}
	}
	source := buildConfig.Source
	if source == consts.ArtifactFromPatch {
		return source, nil
	}
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
	actionConfig := resolveGithubActionConfig(cmd)
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

func resolveGithubActionConfig(cmd *cobra.Command) *clientpb.GithubActionBuildConfig {
	actionConfig := common.ParseGithubFlags(cmd)
	if actionConfig != nil {
		return actionConfig
	}

	settings, err := assets.LoadSettings()
	if err != nil || settings == nil || settings.Github == nil {
		return nil
	}

	return settings.Github.ToProtobuf()
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

// parseOutputType parses --lib and --shellcode flags and sets buildConfig.OutputType.
func parseOutputType(cmd *cobra.Command, buildConfig *clientpb.BuildConfig) error {
	libFlag, _ := cmd.Flags().GetBool("lib")
	shellcodeFlag := false
	if cmd.Flags().Lookup("shellcode") != nil {
		shellcodeFlag, _ = cmd.Flags().GetBool("shellcode")
	}
	return ValidateOutputType(buildConfig, libFlag, cmd.Flags().Changed("lib"), shellcodeFlag)
}

// ValidateOutputType validates output type flags and sets buildConfig.OutputType.
// OutputType values: "" (executable, default), "lib" (dll/so/dylib), "shellcode" (raw .bin, pulse only)
func ValidateOutputType(buildConfig *clientpb.BuildConfig, libFlag bool, libFlagChanged bool, shellcodeFlag bool) error {
	target, ok := consts.GetBuildTarget(buildConfig.Target)
	if !ok {
		return errors.New("invalid target: " + buildConfig.Target)
	}

	if libFlag && shellcodeFlag {
		return errors.New("--lib and --shellcode are mutually exclusive")
	}

	switch buildConfig.BuildType {
	case consts.CommandBuildModules, consts.CommandBuild3rdModules:
		if libFlagChanged && !libFlag {
			return errors.New("modules build requires --lib")
		}
		if target.OS != consts.Windows {
			return errors.New("modules build only supports Windows targets")
		}
		buildConfig.OutputType = "lib"
	case consts.CommandBuildPrelude:
		if libFlag {
			return errors.New("prelude build does not support --lib")
		}
		if shellcodeFlag {
			return errors.New("prelude build does not support --shellcode")
		}
		buildConfig.OutputType = ""
	case consts.CommandBuildPulse:
		if target.OS != consts.Windows {
			return errors.New("pulse build only supports Windows targets")
		}
		if shellcodeFlag {
			buildConfig.OutputType = "shellcode"
		} else if libFlag {
			buildConfig.OutputType = "lib"
		} else {
			buildConfig.OutputType = ""
		}
	default:
		// beacon/bind allow exe and lib
		if shellcodeFlag {
			return errors.New(buildConfig.BuildType + " build does not support --shellcode")
		}
		if libFlag {
			buildConfig.OutputType = "lib"
		} else {
			buildConfig.OutputType = ""
		}
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
