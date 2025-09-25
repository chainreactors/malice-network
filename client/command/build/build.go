package build

import (
	"errors"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func CheckSource(con *repl.Console, buildConfig *clientpb.BuildConfig) (string, error) {
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
func parseBasicConfig(cmd *cobra.Command, con *repl.Console) (*clientpb.BuildConfig, error) {
	// init
	buildConfig := common.ParseGenerateFlags(cmd)

	if buildConfig.Target == "" {
		return nil, errors.New("require build target")
	}

	return buildConfig, nil
}

func parseSourceConfig(cmd *cobra.Command, con *repl.Console, buildConfig *clientpb.BuildConfig) (*clientpb.BuildConfig, error) {
	source, _ := cmd.Flags().GetString("source")
	buildConfig.Source = source
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

// executeBuild 执行构建逻辑
func executeBuild(con *repl.Console, buildConfig *clientpb.BuildConfig) {
	go func() {
		artifact, err := con.Rpc.Build(con.Context(), buildConfig)
		if err != nil {
			con.Log.Errorf("Build %s failed: %v\n", buildConfig.BuildType, err)
			return
		}
		con.Log.Infof("Build started: %s (type: %s, target: %s, source: %s)\n",
			artifact.Name, artifact.Type, artifact.Target, artifact.Source)
	}()
}

func BindCmd(cmd *cobra.Command, con *repl.Console) error {
	buildConfig, err := prepareBuildConfig(cmd, con, consts.CommandBuildBind)
	if err != nil {
		return err
	}

	executeBuild(con, buildConfig)
	return nil
}

func BuildLogCmd(cmd *cobra.Command, con *repl.Console) error {
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
