package action

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

func RunBeaconWorkFlowCmd(cmd *cobra.Command, con *repl.Console) error {
	owner, repo, token, file, remove, err := checkGithubArg(cmd, false)
	if err != nil {
		return err
	}
	name, address, buildTarget, modules, ca, _, params := common.ParseGenerateFlags(cmd)
	if buildTarget == "" {
		return errors.New("require build target")
	}

	inputs := map[string]string{
		"package": consts.CommandBuildBeacon,
		"targets": buildTarget,
	}
	if len(modules) > 0 {
		inputs["malefic_modules_features"] = strings.Join(modules, ",")
	}
	req := &clientpb.BuildConfig{
		Owner:       owner,
		Repo:        repo,
		Token:       token,
		WorkflowId:  file,
		Inputs:      inputs,
		ProfileName: name,
		MaleficHost: address,
		Ca:          ca,
		Params:      params.String(),
		IsRemove:    remove,
		Resource:    consts.ArtifactFromAction,
	}
	resp, err := RunWorkFlow(con, req)
	if err != nil {
		return err
	}
	con.Log.Infof("Create workflow %s type %s targrt %s success\n", resp.Name, resp.Type, resp.Target)
	return nil
}

func RunBindWorkFlowCmd(cmd *cobra.Command, con *repl.Console) error {
	owner, repo, token, file, remove, err := checkGithubArg(cmd, false)
	if err != nil {
		return err
	}
	name, address, buildTarget, modules, ca, _, params := common.ParseGenerateFlags(cmd)
	if buildTarget == "" {
		return errors.New("require build target")
	}
	inputs := map[string]string{
		"package": consts.CommandBuildBind,
		"targets": buildTarget,
	}
	if len(modules) > 0 {
		inputs["malefic_modules_features"] = strings.Join(modules, ",")
	}
	req := &clientpb.BuildConfig{
		Owner:       owner,
		Repo:        repo,
		Token:       token,
		WorkflowId:  file,
		Inputs:      inputs,
		ProfileName: name,
		MaleficHost: address,
		Ca:          ca,
		Params:      params.String(),
		IsRemove:    remove,
		Resource:    consts.ArtifactFromAction,
	}
	resp, err := RunWorkFlow(con, req)
	if err != nil {
		return err
	}
	con.Log.Infof("Create workflow %s type %s targrt %s success\n", resp.Name, resp.Type, resp.Target)
	return nil
}
func RunPreludeWorkFlowCmd(cmd *cobra.Command, con *repl.Console) error {
	owner, repo, token, file, remove, err := checkGithubArg(cmd, false)
	if err != nil {
		return err
	}
	name, address, buildTarget, modules, ca, _, params := common.ParseGenerateFlags(cmd)
	if buildTarget == "" {
		return errors.New("require build target")
	}
	inputs := map[string]string{
		"package": consts.CommandBuildPrelude,
		"targets": buildTarget,
	}
	if len(modules) > 0 {
		inputs["malefic_modules_features"] = strings.Join(modules, ",")
	}
	autorunPath, _ := cmd.Flags().GetString("autorun")
	if autorunPath == "" {
		return errors.New("require autorun.yaml path")
	}
	fileData, err := os.ReadFile(autorunPath)
	if err != nil {
		return err
	}
	base64Encoded := base64.StdEncoding.EncodeToString(fileData)
	inputs["autorun_yaml "] = base64Encoded

	req := &clientpb.BuildConfig{
		Owner:       owner,
		Repo:        repo,
		Token:       token,
		WorkflowId:  file,
		Inputs:      inputs,
		ProfileName: name,
		MaleficHost: address,
		Ca:          ca,
		Params:      params.String(),
		IsRemove:    remove,
		Resource:    consts.ArtifactFromAction,
	}
	resp, err := RunWorkFlow(con, req)
	if err != nil {
		return err
	}
	con.Log.Infof("Create workflow %s type %s targrt %s success\n", resp.Name, resp.Type, resp.Target)
	return nil
}
func RunModulesWorkFlowCmd(cmd *cobra.Command, con *repl.Console) error {
	owner, repo, token, file, remove, err := checkGithubArg(cmd, false)
	if err != nil {
		return err
	}
	name, address, buildTarget, modules, ca, _, params := common.ParseGenerateFlags(cmd)
	if buildTarget == "" {
		return errors.New("require build target")
	}

	inputs := map[string]string{
		"package": consts.CommandBuildModules,
		"targets": buildTarget,
	}
	if len(modules) == 0 {
		inputs["malefic_modules_features"] = "full"
	} else if len(modules) > 0 {
		inputs["malefic_modules_features"] = strings.Join(modules, ",")
	}
	req := &clientpb.BuildConfig{
		Owner:       owner,
		Repo:        repo,
		Token:       token,
		WorkflowId:  file,
		Inputs:      inputs,
		ProfileName: name,
		MaleficHost: address,
		Ca:          ca,
		Params:      params.String(),
		IsRemove:    remove,
		Resource:    consts.ArtifactFromAction,
	}
	resp, err := RunWorkFlow(con, req)
	if err != nil {
		return err
	}
	con.Log.Infof("Create workflow %s type %s targrt %s success\n", resp.Name, resp.Type, resp.Target)
	return nil
}

func RunPulseWorkFlowCmd(cmd *cobra.Command, con *repl.Console) error {
	owner, repo, token, file, remove, err := checkGithubArg(cmd, false)
	if err != nil {
		return err
	}
	name, address, buildTarget, _, _, _, _ := common.ParseGenerateFlags(cmd)
	if !strings.Contains(buildTarget, "windows") {
		con.Log.Warn("pulse only support windows target\n")
		return nil
	}
	artifactID, _ := cmd.Flags().GetUint32("artifact-id")

	inputs := map[string]string{
		"package": consts.CommandBuildPulse,
		"targets": buildTarget,
	}

	req := &clientpb.BuildConfig{
		Owner:       owner,
		Repo:        repo,
		Token:       token,
		WorkflowId:  file,
		Inputs:      inputs,
		ProfileName: name,
		MaleficHost: address,
		ArtifactId:  artifactID,
		IsRemove:    remove,
		Resource:    consts.ArtifactFromAction,
	}
	resp, err := RunWorkFlow(con, req)
	if err != nil {
		return err
	}
	con.Log.Infof("Create workflow %s type %s targrt %s success\n", resp.Name, resp.Type, resp.Target)
	return nil
}

func RunWorkFlow(con *repl.Console, req *clientpb.BuildConfig) (*clientpb.Builder, error) {
	builder, err := con.Rpc.Build(con.Context(), req)
	if err != nil {
		return builder, err
	}
	return builder, nil
}
