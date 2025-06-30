package build

import (
	"encoding/base64"
	"errors"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
	"os"
	"strconv"
	"strings"
)

func CheckResource(owner, repo, token, file, source string, con *repl.Console) (string, error) {
	if source != "" {
		if source != consts.ArtifactFromAction && source != consts.ArtifactFromDocker && source != consts.ArtifactFromSaas {
			return source, errors.New("build resource is violate")
		}
	} else {
		_, err := con.Rpc.DockerStatus(con.Context(), &clientpb.Empty{})
		if err == nil {
			source = consts.ArtifactFromDocker
			return source, nil
		}
		if err != nil {
			source = consts.ArtifactFromSaas
			return source, nil
		}
		_, err = con.Rpc.WorkflowStatus(con.Context(), &clientpb.BuildConfig{
			Owner:      owner,
			Repo:       repo,
			Token:      token,
			WorkflowId: file,
		})
		if err == nil {
			source = consts.ArtifactFromAction
			return source, nil
		}
		source = consts.ArtifactFromSaas
	}
	return source, nil
}

func setActionBuildConfig(owner, repo, token, file string, remove bool, inputs map[string]string, buildConfig *clientpb.BuildConfig) *clientpb.BuildConfig {
	buildConfig.Owner = owner
	buildConfig.Repo = repo
	buildConfig.Token = token
	buildConfig.WorkflowId = file
	buildConfig.Inputs = inputs
	buildConfig.IsRemove = remove
	return buildConfig
}

func BeaconCmd(cmd *cobra.Command, con *repl.Console) error {
	buildConfig := common.ParseGenerateFlags(cmd)
	if buildConfig.Target == "" {
		return errors.New("require build target")
	}
	owner, repo, token, file, remove := common.ParseGithubFlags(cmd)
	finalResource, err := CheckResource(owner, repo, token, file, buildConfig.Source, con)
	if err != nil {
		return err
	}
	go func() {
		if finalResource == consts.ArtifactFromAction {
			inputs := map[string]string{
				"package": consts.CommandBuildBeacon,
				"targets": buildConfig.Target,
			}
			if len(buildConfig.Modules) > 0 {
				inputs["malefic_modules_features"] = strings.Join(buildConfig.Modules, ",")
			}
			buildConfig = setActionBuildConfig(owner, repo, token, file, remove, inputs, buildConfig)
		}
		buildConfig.Source = finalResource
		buildConfig.Type = consts.CommandBuildBeacon
		artifact, err := con.Rpc.Build(con.Context(), buildConfig)
		if err != nil {
			con.Log.Errorf("Build beacon failed: %v\n", err)
			return
		}
		con.Log.Infof("Build %v type %v target %v by %v start\n", artifact.Name, artifact.Type, artifact.Target, artifact.Source)
	}()
	return nil
}

func BindCmd(cmd *cobra.Command, con *repl.Console) error {
	buildConfig := common.ParseGenerateFlags(cmd)
	if buildConfig.Target == "" {
		return errors.New("require build target")
	}
	owner, repo, token, file, remove := common.ParseGithubFlags(cmd)
	finalResource, err := CheckResource(owner, repo, token, file, buildConfig.Source, con)
	if err != nil {
		return err
	}
	go func() {
		if finalResource == consts.ArtifactFromAction {
			inputs := map[string]string{
				"package": consts.CommandBuildBind,
				"targets": buildConfig.Target,
			}
			if len(buildConfig.Modules) > 0 {
				inputs["malefic_modules_features"] = strings.Join(buildConfig.Modules, ",")
			}
			buildConfig = setActionBuildConfig(owner, repo, token, file, remove, inputs, buildConfig)
		}
		buildConfig.Source = finalResource
		buildConfig.Type = consts.CommandBuildBind
		artifact, err := con.Rpc.Build(con.Context(), buildConfig)
		if err != nil {
			con.Log.Errorf("Build bind failed: %v\n", err)
			return
		}
		con.Log.Infof("Build %v type %v target %v by %v start\n", artifact.Name, artifact.Type, artifact.Target, artifact.Source)
	}()
	return nil
}

func PreludeCmd(cmd *cobra.Command, con *repl.Console) error {
	buildConfig := common.ParseGenerateFlags(cmd)
	if buildConfig.Target == "" {
		return errors.New("require build target")
	}
	autorunPath, _ := cmd.Flags().GetString("autorun")
	if autorunPath == "" {
		return errors.New("require autorun.yaml path")
	}
	file, err := os.ReadFile(autorunPath)
	if err != nil {
		return err
	}
	owner, repo, token, fileID, remove := common.ParseGithubFlags(cmd)
	finalResource, err := CheckResource(owner, repo, token, fileID, buildConfig.Source, con)
	if err != nil {
		return err
	}
	go func() {
		if finalResource == consts.ArtifactFromAction {
			base64Encoded := base64.StdEncoding.EncodeToString(file)
			inputs := map[string]string{
				"package": consts.CommandBuildPrelude,
				"targets": buildConfig.Target,
			}
			inputs["autorun_yaml "] = base64Encoded
			if len(buildConfig.Modules) > 0 {
				inputs["malefic_modules_features"] = strings.Join(buildConfig.Modules, ",")
			}
			buildConfig = setActionBuildConfig(owner, repo, token, fileID, remove, inputs, buildConfig)
		}
		buildConfig.Source = finalResource
		buildConfig.Type = consts.CommandBuildPrelude
		artifact, err := con.Rpc.Build(con.Context(), buildConfig)
		if err != nil {
			con.Log.Errorf("Build prelude failed: %v\n", err)
			return
		}
		con.Log.Infof("Build %v type %v target %v by %v start\n", artifact.Name, artifact.Type, artifact.Target, artifact.Source)
	}()
	return nil
}

func ModulesCmd(cmd *cobra.Command, con *repl.Console) error {
	buildConfig := common.ParseGenerateFlags(cmd)
	if buildConfig.Target == "" {
		return errors.New("require build target")
	}
	owner, repo, token, file, remove := common.ParseGithubFlags(cmd)
	finalResource, err := CheckResource(owner, repo, token, file, buildConfig.Source, con)
	if err != nil {
		return err
	}
	go func() {
		if finalResource == consts.ArtifactFromAction {
			inputs := map[string]string{
				"package": consts.CommandBuildModules,
				"targets": buildConfig.Target,
			}
			if len(buildConfig.Modules) == 0 {
				inputs["malefic_modules_features"] = "full"
			} else if len(buildConfig.Modules) > 0 {
				inputs["malefic_modules_features"] = strings.Join(buildConfig.Modules, ",")
			}
			buildConfig = setActionBuildConfig(owner, repo, token, file, remove, inputs, buildConfig)
		}
		buildConfig.Source = finalResource
		buildConfig.Type = consts.CommandBuildModules
		artifact, err := con.Rpc.Build(con.Context(), buildConfig)
		if err != nil {
			con.Log.Errorf("Build prelude failed: %v\n", err)
			return
		}
		con.Log.Infof("Build %v type %v target %v by %v start\n", artifact.Name, artifact.Type, artifact.Target, artifact.Source)
	}()
	return nil
}

func PulseCmd(cmd *cobra.Command, con *repl.Console) error {
	buildConfig := common.ParseGenerateFlags(cmd)
	if buildConfig.Target == "" {
		return errors.New("require build target")
	}
	if !strings.Contains(buildConfig.Target, "windows") {
		con.Log.Warn("pulse only support windows target\n")
		return nil
	}
	artifactId, _ := cmd.Flags().GetUint32("artifact-id")
	owner, repo, token, file, remove := common.ParseGithubFlags(cmd)
	finalResource, err := CheckResource(owner, repo, token, file, buildConfig.Source, con)
	if err != nil {
		return err
	}
	go func() {
		if finalResource == consts.ArtifactFromAction {
			inputs := map[string]string{
				"package": consts.CommandBuildPulse,
				"targets": buildConfig.Target,
			}
			if len(buildConfig.Modules) > 0 {
				inputs["malefic_modules_features"] = strings.Join(buildConfig.Modules, ",")
			}
			buildConfig = setActionBuildConfig(owner, repo, token, file, remove, inputs, buildConfig)
		}
		buildConfig.Source = finalResource
		buildConfig.Type = consts.CommandBuildPulse
		buildConfig.ArtifactId = artifactId
		artifact, err := con.Rpc.Build(con.Context(), buildConfig)
		if err != nil {
			con.Log.Errorf("Build loader failed: %v\n", err)
			return
		}
		con.Log.Infof("Build %v type %v target %v by %v start\n", artifact.Name, artifact.Type, artifact.Target, artifact.Source)
	}()
	return nil
}

func BuildLogCmd(cmd *cobra.Command, con *repl.Console) error {
	id := cmd.Flags().Arg(0)
	buildID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return err
	}
	num, _ := cmd.Flags().GetInt("limit")
	builder, err := con.Rpc.BuildLog(con.Context(), &clientpb.Builder{
		Id:  uint32(buildID),
		Num: uint32(num),
	})
	if err != nil {
		return err
	}
	if len(builder.Log) == 0 {
		con.Log.Infof("No log for %s\n", id)
		return nil
	}
	con.Log.Console(string(builder.Log))
	return nil
}
