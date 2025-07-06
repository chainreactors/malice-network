package build

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/types"
	"os"
	"strings"

	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func CheckResource(con *repl.Console, source string, github *clientpb.GithubWorkflowConfig) (string, error) {
	if github == nil {
		config, err := assets.GetSetting()
		if err != nil {
			return "", err
		}
		github = config.Github.ToProtobuf()
	}

	if source != "" {
		if source != consts.ArtifactFromAction && source != consts.ArtifactFromDocker && source != consts.ArtifactFromSaas {
			return source, errors.New("build source invalid")
		} else {
			return source, nil
		}
	}

	_, err := con.Rpc.DockerStatus(con.Context(), &clientpb.Empty{})
	if err == nil {
		source = consts.ArtifactFromDocker
		return source, nil
	}
	_, err = con.Rpc.WorkflowStatus(con.Context(), github)
	if err == nil {
		source = consts.ArtifactFromAction
		return source, nil
	}
	return consts.ArtifactFromSaas, nil
}

// parseBasicConfig 解析基础配置（可复用部分）
func parseBasicConfig(cmd *cobra.Command, con *repl.Console) (*clientpb.BuildConfig, string, error) {
	buildConfig := common.ParseGenerateFlags(cmd)
	if buildConfig.Target == "" {
		return nil, "", errors.New("require build target")
	}

	buildConfig.Github = common.ParseGithubFlags(cmd)

	finalSource, err := CheckResource(con, buildConfig.Source, buildConfig.Github)
	if err != nil {
		return nil, "", err
	}
	return buildConfig, finalSource, nil
}

// prepareBuildConfig 准备标准构建配置
func prepareBuildConfig(cmd *cobra.Command, con *repl.Console, buildType string) (*clientpb.BuildConfig, error) {
	buildConfig, finalSource, err := parseBasicConfig(cmd, con)
	if err != nil {
		return nil, err
	}

	buildConfig.Source = finalSource
	buildConfig.Type = buildType
	if buildType != consts.CommandBuildModules {
		var params types.ProfileParams
		err = json.Unmarshal(buildConfig.ParamsBytes, &params)
		if err != nil {
			return nil, err
		}
		if params.Modules == "" {
			params.Modules = "full"
			buildConfig.ParamsBytes = []byte(params.String())
		}
	}
	return buildConfig, nil
}

// executeBuild 执行构建逻辑
func executeBuild(con *repl.Console, buildConfig *clientpb.BuildConfig) {
	go func() {
		artifact, err := con.Rpc.Build(con.Context(), buildConfig)
		if err != nil {
			con.Log.Errorf("Build %s failed: %v\n", buildConfig.Type, err)
			return
		}
		con.Log.Infof("Build started: %s (type: %s, target: %s, source: %s)\n",
			artifact.Name, artifact.Type, artifact.Target, artifact.Source)
	}()
}

func BeaconCmd(cmd *cobra.Command, con *repl.Console) error {
	buildConfig, err := prepareBuildConfig(cmd, con, consts.CommandBuildBeacon)
	if err != nil {
		return err
	}
	executeBuild(con, buildConfig)
	return nil
}

func BindCmd(cmd *cobra.Command, con *repl.Console) error {
	buildConfig, err := prepareBuildConfig(cmd, con, consts.CommandBuildBind)
	if err != nil {
		return err
	}

	executeBuild(con, buildConfig)
	return nil
}

func PreludeCmd(cmd *cobra.Command, con *repl.Console) error {
	buildConfig, err := prepareBuildConfig(cmd, con, consts.CommandBuildPrelude)
	if err != nil {
		return err
	}

	autorunPath, _ := cmd.Flags().GetString("autorun")
	if autorunPath == "" {
		return errors.New("require autorun.yaml path")
	}
	file, err := os.ReadFile(autorunPath)
	if err != nil {
		return err
	}

	// PreludeCmd需要额外的autorun_yaml参数
	if buildConfig.Source == consts.ArtifactFromAction {
		base64Encoded := base64.StdEncoding.EncodeToString(file)
		buildConfig.Inputs["autorun_yaml "] = base64Encoded
	}

	executeBuild(con, buildConfig)
	return nil
}

func ModulesCmd(cmd *cobra.Command, con *repl.Console) error {
	buildConfig, finalSource, err := parseBasicConfig(cmd, con)
	if err != nil {
		return err
	}
	modules, _ := cmd.Flags().GetString("modules")
	thirdModules, _ := cmd.Flags().GetString("3rd")
	if modules == "" && thirdModules == "" {
		return errors.New("--module and --3rd options are mutually exclusive. please specify only one of them")
	}
	var profileParams types.ProfileParams
	err = json.Unmarshal(buildConfig.ParamsBytes, &profileParams)
	if err != nil {
		return err
	}
	if modules != "" {
		profileParams.Modules = modules
	} else if thirdModules != "" {
		profileParams.Enable3RD = true
		profileParams.Modules = thirdModules
	} else {
		return errors.New("must specify either --modules or --3rd_modules. One of them is required")
	}
	buildConfig.ParamsBytes = []byte(profileParams.String())
	buildConfig.Source = finalSource
	buildConfig.Type = consts.CommandBuildModules

	executeBuild(con, buildConfig)
	return nil
}

func PulseCmd(cmd *cobra.Command, con *repl.Console) error {
	buildConfig, err := prepareBuildConfig(cmd, con, consts.CommandBuildPulse)
	if err != nil {
		return err
	}

	if !strings.Contains(buildConfig.Target, "windows") {
		con.Log.Warn("Pulse only supports Windows targets\n")
		return nil
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
