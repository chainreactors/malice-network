package build

import (
	"encoding/base64"
	"errors"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
	"os"
	"strconv"
	"strings"
)

func checkGithubArg(cmd *cobra.Command, isList bool) (string, string, string, string, bool, error) {
	owner, repo, token, file, remove := common.ParseGithubFlags(cmd)
	setting, err := assets.GetSetting()
	if err != nil {
		return "", "", "", "", false, err
	}
	if owner == "" {
		owner = setting.GithubOwner
	}
	if repo == "" {
		repo = setting.GithubRepo
	}
	if token == "" {
		token = setting.GithubToken
	}
	if !isList {
		if file == "" {
			file = setting.GithubWorkflowFile
		}
		if file == "" {
			file = "generate.yaml"
		}
	}
	return owner, repo, token, file, remove, nil
}

func checkResource(cmd *cobra.Command, resource string, con *repl.Console) (string, error) {
	if resource != "" {
		if resource != consts.ArtifactFromAction && resource != consts.ArtifactFromDocker && resource != consts.ArtifactFromSaas {
			return resource, errors.New("build resource is violate")
		}
	} else {
		_, err := con.Rpc.DockerStatus(con.Context(), &clientpb.Empty{})
		if err == nil {
			resource = consts.ArtifactFromDocker
			return resource, nil
		}
		owner, repo, token, file, _, err := checkGithubArg(cmd, false)
		if err != nil {
			resource = consts.ArtifactFromSaas
			return resource, nil
		}
		_, err = con.Rpc.WorkflowStatus(con.Context(), &clientpb.BuildConfig{
			Owner:      owner,
			Repo:       repo,
			Token:      token,
			WorkflowId: file,
		})
		if err == nil {
			resource = consts.ArtifactFromAction
			return resource, nil
		}
		resource = consts.ArtifactFromSaas
	}
	return resource, nil
}

func BeaconCmd(cmd *cobra.Command, con *repl.Console) error {
	name, address, buildTarget, modules, _, params, resource := common.ParseGenerateFlags(cmd)
	if buildTarget == "" {
		return errors.New("require build target")
	}
	finalResource, err := checkResource(cmd, resource, con)
	if err != nil {
		return err
	}
	owner, repo, token, file, remove, err := checkGithubArg(cmd, false)
	if err != nil {
		return err
	}
	go func() {
		var buildConfig *clientpb.BuildConfig
		if finalResource == consts.ArtifactFromAction {
			inputs := map[string]string{
				"package": consts.CommandBuildBeacon,
				"targets": buildTarget,
			}
			if len(modules) > 0 {
				inputs["malefic_modules_features"] = strings.Join(modules, ",")
			}
			buildConfig = &clientpb.BuildConfig{
				Owner:       owner,
				Repo:        repo,
				Token:       token,
				WorkflowId:  file,
				Inputs:      inputs,
				ProfileName: name,
				MaleficHost: address,
				Params:      params.String(),
				IsRemove:    remove,
				Resource:    finalResource,
			}
		} else {
			buildConfig = &clientpb.BuildConfig{
				ProfileName: name,
				MaleficHost: address,
				Type:        consts.CommandBuildBeacon,
				Target:      buildTarget,
				Modules:     modules,
				Params:      params.String(),
				Srdi:        true,
				Resource:    finalResource,
			}
		}
		artifact, err := con.Rpc.Build(con.Context(), buildConfig)
		if err != nil {
			con.Log.Errorf("Build beacon failed: %v\n", err)
			return
		}
		con.Log.Infof("Build %v type %v target %v by %v start\n", artifact.Name, artifact.Type, artifact.Target, artifact.Resource)
	}()
	return nil
}

func BindCmd(cmd *cobra.Command, con *repl.Console) error {
	name, address, buildTarget, modules, _, params, resource := common.ParseGenerateFlags(cmd)
	if buildTarget == "" {
		return errors.New("require build target")
	}
	finalResource, err := checkResource(cmd, resource, con)
	if err != nil {
		return err
	}
	owner, repo, token, file, remove, err := checkGithubArg(cmd, false)
	if err != nil {
		return err
	}
	go func() {
		var buildConfig *clientpb.BuildConfig
		if finalResource == consts.ArtifactFromAction {
			inputs := map[string]string{
				"package": consts.CommandBuildBind,
				"targets": buildTarget,
			}
			if len(modules) > 0 {
				inputs["malefic_modules_features"] = strings.Join(modules, ",")
			}
			buildConfig = &clientpb.BuildConfig{
				Owner:       owner,
				Repo:        repo,
				Token:       token,
				WorkflowId:  file,
				Inputs:      inputs,
				ProfileName: name,
				MaleficHost: address,
				Params:      params.String(),
				IsRemove:    remove,
				Resource:    finalResource,
			}
		} else {
			buildConfig = &clientpb.BuildConfig{
				ProfileName: name,
				MaleficHost: address,
				Type:        consts.CommandBuildBind,
				Target:      buildTarget,
				Modules:     modules,
				Params:      params.String(),
				Srdi:        true,
				Resource:    finalResource,
			}
		}
		artifact, err := con.Rpc.Build(con.Context(), buildConfig)
		if err != nil {
			con.Log.Errorf("Build bind failed: %v\n", err)
			return
		}
		con.Log.Infof("Build %v type %v target %v by %v start\n", artifact.Name, artifact.Type, artifact.Target, artifact.Resource)
	}()
	return nil
}

func PreludeCmd(cmd *cobra.Command, con *repl.Console) error {
	name, address, buildTarget, modules, _, params, resource := common.ParseGenerateFlags(cmd)
	if buildTarget == "" {
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
	finalResource, err := checkResource(cmd, resource, con)
	if err != nil {
		return err
	}
	owner, repo, token, fileID, remove, err := checkGithubArg(cmd, false)
	if err != nil {
		return err
	}
	go func() {
		var buildConfig *clientpb.BuildConfig
		if finalResource == consts.ArtifactFromAction {
			base64Encoded := base64.StdEncoding.EncodeToString(file)
			inputs := map[string]string{
				"package": consts.CommandBuildPrelude,
				"targets": buildTarget,
			}
			inputs["autorun_yaml "] = base64Encoded
			if len(modules) > 0 {
				inputs["malefic_modules_features"] = strings.Join(modules, ",")
			}
			buildConfig = &clientpb.BuildConfig{
				Owner:       owner,
				Repo:        repo,
				Token:       token,
				WorkflowId:  fileID,
				Inputs:      inputs,
				ProfileName: name,
				MaleficHost: address,
				Params:      params.String(),
				IsRemove:    remove,
				Resource:    finalResource,
			}
		} else {
			buildConfig = &clientpb.BuildConfig{
				ProfileName: name,
				MaleficHost: address,
				Type:        consts.CommandBuildPrelude,
				Target:      buildTarget,
				Modules:     modules,
				Params:      params.String(),
				Srdi:        true,
				Bin:         file,
				Resource:    finalResource,
			}
		}
		artifact, err := con.Rpc.Build(con.Context(), buildConfig)
		if err != nil {
			con.Log.Errorf("Build prelude failed: %v\n", err)
			return
		}
		con.Log.Infof("Build %v type %v target %v by %v start\n", artifact.Name, artifact.Type, artifact.Target, artifact.Resource)
	}()
	return nil
}

func ModulesCmd(cmd *cobra.Command, con *repl.Console) error {
	name, address, buildTarget, modules, srdi, params, resource := common.ParseGenerateFlags(cmd)
	if buildTarget == "" {
		return errors.New("require build target")
	}
	finalResource, err := checkResource(cmd, resource, con)
	if err != nil {
		return err
	}
	owner, repo, token, file, remove, err := checkGithubArg(cmd, false)
	if err != nil {
		return err
	}
	go func() {
		var buildConfig *clientpb.BuildConfig
		if finalResource == consts.ArtifactFromAction {
			inputs := map[string]string{
				"package": consts.CommandBuildModules,
				"targets": buildTarget,
			}
			if len(modules) == 0 {
				inputs["malefic_modules_features"] = "full"
			} else if len(modules) > 0 {
				inputs["malefic_modules_features"] = strings.Join(modules, ",")
			}
			buildConfig = &clientpb.BuildConfig{
				Owner:       owner,
				Repo:        repo,
				Token:       token,
				WorkflowId:  file,
				Inputs:      inputs,
				ProfileName: name,
				MaleficHost: address,
				Params:      params.String(),
				IsRemove:    remove,
				Resource:    finalResource,
			}
		} else {
			buildConfig = &clientpb.BuildConfig{
				ProfileName: name,
				MaleficHost: address,
				Target:      buildTarget,
				Type:        consts.CommandBuildModules,
				Modules:     modules,
				Srdi:        srdi,
				Resource:    finalResource,
			}
		}
		artifact, err := con.Rpc.Build(con.Context(), buildConfig)
		if err != nil {
			con.Log.Errorf("Build prelude failed: %v\n", err)
			return
		}
		con.Log.Infof("Build %v type %v target %v by %v start\n", artifact.Name, artifact.Type, artifact.Target, artifact.Resource)
	}()
	return nil
}

func PulseCmd(cmd *cobra.Command, con *repl.Console) error {
	name, address, buildTarget, modules, _, _, resource := common.ParseGenerateFlags(cmd)
	if buildTarget == "" {
		return errors.New("require build target")
	}
	if !strings.Contains(buildTarget, "windows") {
		con.Log.Warn("pulse only support windows target\n")
		return nil
	}
	artifactId, _ := cmd.Flags().GetUint32("artifact-id")
	owner, repo, token, file, remove, err := checkGithubArg(cmd, false)
	if err != nil {
		return err
	}
	finalResource, err := checkResource(cmd, resource, con)
	if err != nil {
		return err
	}
	go func() {
		var buildConfig *clientpb.BuildConfig
		if finalResource == consts.ArtifactFromAction {
			inputs := map[string]string{
				"package": consts.CommandBuildPulse,
				"targets": buildTarget,
			}
			if len(modules) > 0 {
				inputs["malefic_modules_features"] = strings.Join(modules, ",")
			}
			buildConfig = &clientpb.BuildConfig{
				Owner:       owner,
				Repo:        repo,
				Token:       token,
				WorkflowId:  file,
				Inputs:      inputs,
				ProfileName: name,
				MaleficHost: address,
				ArtifactId:  artifactId,
				IsRemove:    remove,
				Resource:    finalResource,
			}
		} else {
			buildConfig = &clientpb.BuildConfig{
				ProfileName: name,
				MaleficHost: address,
				Target:      buildTarget,
				Type:        consts.CommandBuildPulse,
				Srdi:        true,
				ArtifactId:  artifactId,
				Resource:    finalResource,
			}
		}
		artifact, err := con.Rpc.Build(con.Context(), buildConfig)
		if err != nil {
			con.Log.Errorf("Build loader failed: %v\n", err)
			return
		}
		con.Log.Infof("Build %v type %v target %v by %v start\n", artifact.Name, artifact.Type, artifact.Target, artifact.Resource)
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

func RunSaas(con *repl.Console, req *clientpb.BuildConfig) (*clientpb.Builder, error) {
	builder, err := con.Rpc.Build(con.Context(), req)
	if err != nil {
		return builder, err
	}
	return builder, nil
}
