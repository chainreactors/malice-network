package build

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/httputils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var apiVersion = "2022-11-28"

// Workflow represents a GitHub Actions workflow
type Workflow struct {
	ID           int64     `json:"id"`
	NodeID       string    `json:"node_id"`
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Status       string    `json:"status"`
	State        string    `json:"state"`
	Conclusion   string    `json:"conclusion"`
	URL          string    `json:"url"`
	HTMLURL      string    `json:"html_url"`
	ArtifactsURL string    `json:"artifacts_url"`
}

// WorkflowDispatchPayload defines the payload for triggering a workflow dispatch event
type WorkflowDispatchPayload struct {
	Ref    string            `json:"ref"`
	Inputs map[string]string `json:"inputs,omitempty"`
}

// Artifact represents a GitHub Actions artifact
type Artifact struct {
	ID                 int64     `json:"id"`
	NodeID             string    `json:"node_id"`
	Name               string    `json:"name"`
	SizeInBytes        int64     `json:"size_in_bytes"`
	URL                string    `json:"url"`
	ArchiveDownloadURL string    `json:"archive_download_url"`
	Expired            bool      `json:"expired"`
	CreatedAt          time.Time `json:"created_at"`
	ExpiresAt          time.Time `json:"expires_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	WorkflowRun        struct {
		ID               int64  `json:"id"`
		RepositoryID     int64  `json:"repository_id"`
		HeadRepositoryID int64  `json:"head_repository_id"`
		HeadBranch       string `json:"head_branch"`
		HeadSHA          string `json:"head_sha"`
	} `json:"workflow_run"`
}

type Step struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

type Job struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	Steps      []Step `json:"steps"`
}

type JobsResponse struct {
	TotalCount int   `json:"total_count"`
	Jobs       []Job `json:"jobs"`
}

// ArtifactsResponse is the response structure for listing artifacts
type ArtifactsResponse struct {
	TotalCount int        `json:"total_count"`
	Artifacts  []Artifact `json:"artifacts"`
}

// 统一GitHub API请求头
func githubHeaders(token string) map[string]string {
	return map[string]string{
		"Accept":               "application/vnd.github+json",
		"Authorization":        "Bearer " + token,
		"X-GitHub-Api-Version": apiVersion,
	}
}

func GetWorkflowStatus(config *clientpb.GithubWorkflowConfig) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/workflows/%s", config.Owner, config.Repo, config.WorkflowId)
	headers := githubHeaders(config.Token)
	var workflow Workflow
	err := httputils.DoGET(url, headers, &workflow)
	if err != nil {
		return fmt.Errorf("failed to get workflow details: %v", err)
	}
	if workflow.State != "active" {
		return errs.ErrorDockerNotActive
	}
	return nil
}

func runWorkFlow(owner, repo, workflowID, token string, inputs map[string]string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/workflows/%s/dispatches", owner, repo, workflowID)
	headers := githubHeaders(token)
	payload := WorkflowDispatchPayload{
		Ref:    "master",
		Inputs: inputs,
	}
	payloadByte, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %s", err)
	}
	resp, err := httputils.DoRequest("POST", url, bytes.NewBuffer(payloadByte), headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to trigger Build workflow. Status code: %d", resp.StatusCode)
	}
	logs.Log.Info("Build workflow triggered successfully!")
	return nil
}

// downloadArtifactWhenReady waits for the artifact to be ready and downloads it
func downloadArtifactWhenReady(owner, repo, token string, isRemove bool, artifactID uint32, builder *models.Artifact) (string, error) {
	for {
		_, err := PushArtifact(owner, repo, token, builder.Name, isRemove)
		if err == nil {
			logs.Log.Info("Artifact downloaded successfully!")
			db.UpdateBuilderStatus(builder.ID, consts.BuildStatusCompleted)
			if builder.Type == consts.CommandBuildBeacon {
				if artifactID != 0 {
					err = db.UpdatePulseRelink(artifactID, builder.ID)
					if err != nil {
						logs.Log.Errorf("failed to update pulse relink: %s", err)
					}
				}
			}
			return builder.Path, nil
		} else if errors.Is(err, errs.ErrWorkflowFailed) {
			logs.Log.Errorf("Download artifact failed due to workflow failure: %s", err)
			db.UpdateBuilderStatus(builder.ID, consts.BuildStatusFailure)
			return "", errs.ErrWorkflowFailed
		} else {
			logs.Log.Debugf("Download artifact failed: %s", err)
		}
		time.Sleep(30 * time.Second)
	}
}

var notifiedWorkflows = make(map[string]bool)

// 获取仓库所有workflow运行
func listRepositoryWorkflows(owner, repo, token string) ([]Workflow, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs", owner, repo)
	headers := githubHeaders(token)
	var result struct {
		TotalCount int        `json:"total_count"`
		Workflows  []Workflow `json:"workflow_runs"`
	}
	err := httputils.DoGET(url, headers, &result)
	if err != nil {
		return nil, err
	}
	return result.Workflows, nil
}

// 查找指定workflow
func findWorkflowByName(workflows []Workflow, name string) (*Workflow, error) {
	for _, wf := range workflows {
		if wf.Name == name {
			return &wf, nil
		}
	}
	return nil, fmt.Errorf("workflow %s not found", name)
}

// 获取artifact下载链接
func getArtifactDownloadURL(owner, repo, token, buildName string) (string, int64, error) {
	workflows, err := listRepositoryWorkflows(owner, repo, token)
	if err != nil {
		return "", 0, err
	}
	wf, err := findWorkflowByName(workflows, buildName)
	if err != nil {
		return "", 0, err
	}
	if wf.Status != "completed" || wf.Conclusion != "success" {
		if wf.Conclusion == "failure" {
			return "", 0, errs.ErrWorkflowFailed
		}
		if !notifiedWorkflows[buildName] {
			core.EventBroker.Publish(core.Event{
				EventType: consts.EventBuild,
				IsNotify:  false,
				Message:   fmt.Sprintf("action %s run in %s.", buildName, wf.HTMLURL),
				Important: true,
			})
			notifiedWorkflows[buildName] = true
		}
		return "", 0, fmt.Errorf("workflow %s not completed or not successful", buildName)
	}
	headers := githubHeaders(token)
	var artifactsResp struct {
		TotalCount int        `json:"total_count"`
		Artifacts  []Artifact `json:"artifacts"`
	}
	err = httputils.DoGET(wf.ArtifactsURL, headers, &artifactsResp)
	if err != nil {
		return "", 0, err
	}
	if len(artifactsResp.Artifacts) == 0 {
		return "", 0, fmt.Errorf("no artifact found")
	}
	return artifactsResp.Artifacts[0].ArchiveDownloadURL, wf.ID, nil
}

// 下载artifact文件
func downloadArtifactFile(url, token string) ([]byte, error) {
	headers := githubHeaders(token)
	resp, err := httputils.DoRequest("GET", url, nil, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed, status: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// 删除成功workflow
func DeleteSuccessWorkflow(owner, repo, token string, workflowID int64) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs/%d", owner, repo, workflowID)
	headers := githubHeaders(token)
	resp, err := httputils.DoRequest("DELETE", url, nil, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete workflow. Status code: %d", resp.StatusCode)
	}
	return nil
}

// 主流程
func PushArtifact(owner, repo, token, buildName string, isRemove bool) (*models.Artifact, error) {
	builder, err := db.GetArtifactByName(buildName)
	if err != nil {
		return nil, err
	}
	if builder.Path != "" {
		if _, err := os.ReadFile(builder.Path); err == nil {
			return builder, nil
		}
	}
	artifactURL, workflowID, err := getArtifactDownloadURL(owner, repo, token, buildName)
	if err != nil {
		return nil, err
	}
	raw, err := downloadArtifactFile(artifactURL, token)
	if err != nil {
		return nil, err
	}
	content, err := fileutils.UnzipOneWithBytes(raw)
	if err != nil {
		return nil, err
	}
	filename := filepath.Join(configs.BuildOutputPath, encoders.UUID())
	if err := os.WriteFile(filename, content, 0644); err != nil {
		return nil, err
	}
	builder.Path, err = filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	if err := db.UpdateBuilderPath(builder); err != nil {
		return nil, err
	}
	if isRemove {
		if err := DeleteSuccessWorkflow(owner, repo, token, workflowID); err != nil {
			return nil, err
		}
	}
	return builder, nil
}

// 获取action状态
func GetActionStatus(owner, repo, token, artifactName string) (string, string, error) {
	workflows, err := listRepositoryWorkflows(owner, repo, token)
	if err != nil {
		return "", "", err
	}
	wf, err := findWorkflowByName(workflows, artifactName)
	if err != nil {
		return "", "", err
	}
	return wf.Status, wf.Conclusion, nil
}

// 辅助函数
func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
