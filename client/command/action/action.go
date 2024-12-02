package action

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
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
	params := &types.ProfileParams{
		Interval: interval,
		Jitter:   jitter,
	}
	//fileData, err := os.ReadFile(buildPath)
	//if err != nil {
	//	log.Fatalf("failed to read file: %v", err)
	//}
	//base64Encoded := base64.StdEncoding.EncodeToString(fileData)
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
	}
	_, err = RunWorkFlow(con, req)
	if err != nil {
		return err
	}
	con.Log.Infof("Create workflow success\n")
	return nil
}

func EnableWorkFlowCmd(cmd *cobra.Command, con *repl.Console) error {
	owner, repo, token, file, err := checkGithubArg(cmd, false)
	if err != nil {
		return err
	}
	req := &clientpb.WorkflowRequest{
		Owner:      owner,
		Repo:       repo,
		Token:      token,
		WorkflowId: file,
	}
	_, err = EnableWorkFlow(con, req)
	if err != nil {
		return err
	}
	con.Log.Infof("Enable workflow %s success\n", file)
	return nil
}

func DisableWorkFlowCmd(cmd *cobra.Command, con *repl.Console) error {
	owner, repo, token, file, err := checkGithubArg(cmd, false)
	if err != nil {
		return err
	}
	req := &clientpb.WorkflowRequest{
		Owner:      owner,
		Repo:       repo,
		Token:      token,
		WorkflowId: file,
	}
	_, err = DisableWorkFlow(con, req)
	if err != nil {
		return err
	}
	con.Log.Infof("Disable workflow %s success\n", file)
	return nil
}

func ListWorkFlowCmd(cmd *cobra.Command, con *repl.Console) error {
	owner, repo, token, file, err := checkGithubArg(cmd, true)
	if err != nil {
		return err
	}
	req := &clientpb.WorkflowRequest{
		Owner:      owner,
		Repo:       repo,
		Token:      token,
		WorkflowId: file,
	}
	resp, err := ListWorkFlow(con, req)
	if err != nil {
		return err
	}
	if len(resp.Workflows) == 0 {
		con.Log.Infof("No workflow\n")
		return nil
	}
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("NodeID", "Node ID", 15),
		table.NewColumn("Name", "Name", 15),
		table.NewColumn("Path", "Path", 20),
		table.NewColumn("State", "State", 10),
		table.NewColumn("created_at", "Created at", 15),
		table.NewColumn("updated_at", "Updated at", 15),
	}, true)
	for _, wf := range resp.Workflows {
		row = table.NewRow(
			table.RowData{
				"NodeID":     wf.NodeId,
				"Name":       wf.Name,
				"Path":       wf.Path,
				"Status":     wf.Status,
				"created_at": wf.CreatedAt,
				"updated_at": wf.UpdatedAt,
			})
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	fmt.Println(tableModel.View())
	return nil
}

func RunWorkFlow(con *repl.Console, req *clientpb.WorkflowRequest) (*clientpb.Builder, error) {
	builder, err := con.Rpc.TriggerWorkflowDispatch(con.Context(), req)
	if err != nil {
		return builder, err
	}
	return builder, nil
}

func EnableWorkFlow(con *repl.Console, req *clientpb.WorkflowRequest) (bool, error) {
	_, err := con.Rpc.EnableWorkflow(con.Context(), req)
	if err != nil {
		return false, err
	}
	return true, nil
}

func DisableWorkFlow(con *repl.Console, req *clientpb.WorkflowRequest) (bool, error) {
	_, err := con.Rpc.DisableWorkflow(con.Context(), req)
	if err != nil {
		return false, err
	}
	return true, nil
}

func ListWorkFlow(con *repl.Console, req *clientpb.WorkflowRequest) (*clientpb.ListWorkflowsResponse, error) {
	resp, err := con.Rpc.ListRepositoryWorkflows(con.Context(), req)
	if err != nil {
		return resp, err
	}
	return resp, nil
}
