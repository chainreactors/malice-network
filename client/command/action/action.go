package action

import (
	"encoding/base64"
	"errors"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

func checkGithubArg(cmd *cobra.Command, isList bool) (string, string, string, string, error) {
	owner, repo, token, file := common.ParseGithubFlags(cmd)
	profile := assets.GetProfile().Settings
	if owner == "" {
		owner = profile.GithubOwner
	}
	if repo == "" {
		repo = profile.GithubRepo
	}
	if token == "" {
		token = profile.GithubToken
	}
	if owner == "" || repo == "" || token == "" {
		return "", "", "", "", errors.New("require github owner/repo/token")
	}
	if !isList {
		if file == "" {
			file = profile.GithubWorkflowFile
		}
		if file == "" {
			return "", "", "", "", errors.New("require github workflowID")
		}
	}
	return owner, repo, token, file, nil
}

func RunWorkFlowCmd(cmd *cobra.Command, con *repl.Console) error {
	owner, repo, token, file, err := checkGithubArg(cmd, false)
	if err != nil {
		return err
	}
	name, address, buildTarget, modules, ca, interval, jitter, _ := common.ParseGenerateFlags(cmd)
	if buildTarget == "" {
		return errors.New("require build target")
	}
	buildType := cmd.Flag("type").Value.String()
	artifactID, _ := cmd.Flags().GetUint32("artifact-id")
	params := &types.ProfileParams{
		Interval: interval,
		Jitter:   jitter,
	}
	inputs := map[string]string{
		"package": buildType,
		"targets": buildTarget,
	}
	if buildType == consts.CommandBuildModules && len(modules) == 0 {
		inputs["malefic_modules_features"] = "full"
	} else if len(modules) > 0 {
		inputs["malefic_modules_features"] = strings.Join(modules, ",")
	}
	if buildType == consts.CommandBuildPrelude {
		autorunPath, _ := cmd.Flags().GetString("autorun")
		if autorunPath == "" {
			return errors.New("require autorun.yaml path")
		}
		fileData, err := os.ReadFile(autorunPath)
		if err != nil {
			return err
		}
		base64Encoded := base64.StdEncoding.EncodeToString(fileData)
		inputs["malefic_config_yaml"] = base64Encoded
	}
	req := &clientpb.WorkflowRequest{
		Owner:      owner,
		Repo:       repo,
		Token:      token,
		WorkflowId: file,
		Inputs:     inputs,
		Profile:    name,
		Address:    address,
		Ca:         ca,
		Params:     params.String(),
		ArtifactId: artifactID,
	}
	resp, err := RunWorkFlow(con, req)
	if err != nil {
		return err
	}
	con.Log.Infof("Create workflow %s type %s targrt %s success\n", resp.Name, resp.Type, resp.Target)
	return nil
}

func RunWorkFlow(con *repl.Console, req *clientpb.WorkflowRequest) (*clientpb.Builder, error) {
	builder, err := con.Rpc.TriggerWorkflowDispatch(con.Context(), req)
	if err != nil {
		return builder, err
	}
	return builder, nil
}
