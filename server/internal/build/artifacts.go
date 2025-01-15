package build

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

var notifiedWorkflows = make(map[string]bool)

func PushArtifact(owner, repo, token, buildName string, isRemove bool) (*models.Builder, error) {
	builder, err := db.GetArtifactByName(buildName)
	if err != nil {
		return builder, err
	}
	if builder.Path != "" {
		_, err := os.ReadFile(builder.Path)
		if err != nil {
			return builder, err
		}
		return builder, nil
	}

	artifactDownloadUrl, workflowID, err := getArtifactDownloadUrl(owner, repo, token, buildName)
	if err != nil {
		return builder, err
	}

	raw, err := downloadFile(artifactDownloadUrl, token)
	if err != nil {
		return builder, fmt.Errorf("download artifact failed: %v", err)
	}

	content, err := fileutils.UnzipOneWithBytes(raw)
	if err != nil {
		return builder, fmt.Errorf("unzip artifact failed: %v", err)
	}
	filename := filepath.Join(configs.BuildOutputPath, encoders.UUID())
	err = os.WriteFile(filename, content, 0644)
	if err != nil {
		return builder, err
	}
	builder.Path, err = filepath.Abs(filename)
	if err != nil {
		return builder, err
	}
	builder.IsSRDI = true
	err = db.UpdateBuilderPath(builder)
	if err != nil {
		return builder, err
	}
	if isRemove {
		err = DeleteSuccessWorkflow(owner, repo, token, workflowID)
		if err != nil {
			return builder, err
		}
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventBuild,
			IsNotify:  false,
			Message:   fmt.Sprintf("workflow %s %s %s has deleted", builder.Name, builder.Type, builder.Target),
			Important: true,
		})
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventBuild,
		IsNotify:  false,
		Message:   fmt.Sprintf("action %s %s %s has finished", builder.Name, builder.Type, builder.Target),
		Important: true,
	})
	return builder, nil
}

// getArtifactDownloadUrl retrieves the artifact download URL from the GitHub API response
func getArtifactDownloadUrl(owner, repo, token, buildName string) (string, int64, error) {
	// Get the artifact list
	artifactUrl, workflowID, err := getArtifact(owner, repo, token, buildName)
	if err != nil && !errors.Is(err, errs.ErrWorkflowFailed) {
		return "", 0, fmt.Errorf("failed to get artifact URL: %v", err)
	} else if errors.Is(err, errs.ErrWorkflowFailed) {
		return "", 0, err
	}

	artifactDownloadUrl, err := fetchArtifactDownloadUrl(artifactUrl, token)
	if err != nil {
		return "", 0, fmt.Errorf("failed to fetch artifact download URL: %v", err)
	}
	return artifactDownloadUrl, workflowID, nil
}

func getArtifact(owner, repo, token, buildName string) (string, int64, error) {
	workflows, err := listRepositoryWorkflows(owner, repo, token)
	if err != nil {
		return "", 0, err
	}
	artifactUrl, workflowID, err := findArtifactsURL(workflows, buildName)
	if err != nil {
		return "", 0, err
	}
	return artifactUrl, workflowID, nil
}

// listRepositoryWorkflows fetches the workflows for a given repository
func listRepositoryWorkflows(owner, repo, token string) ([]Workflow, error) {
	// Construct the GitHub API URL
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs", owner, repo)

	resp, err := sendRequest(url, []byte{}, token, "GET")
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list workflows. Status code: %d", resp.StatusCode)
	}

	// Parse the response body
	var result struct {
		TotalCount int        `json:"total_count"`
		Workflows  []Workflow `json:"workflow_runs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return result.Workflows, nil
}

// findArtifactsURL finds the ArtifactsURL for a workflow by name
func findArtifactsURL(workflows []Workflow, name string) (string, int64, error) {
	for _, wf := range workflows {
		if wf.Name == name {
			if wf.Status == "completed" && wf.Conclusion == "success" {
				return wf.ArtifactsURL, wf.ID, nil
			} else if wf.Conclusion == "failure" {
				return "", 0, errs.ErrWorkflowFailed
			}
			if !notifiedWorkflows[name] {
				core.EventBroker.Publish(core.Event{
					EventType: consts.EventBuild,
					IsNotify:  false,
					Message:   fmt.Sprintf("action %s run in %s.", name, wf.HTMLURL),
					Important: true,
				})
				notifiedWorkflows[name] = true
			}
		}
	}
	return "", 0, errors.New("no artifact found") // Return empty string if not found
}

// fetchArtifactDownloadUrl  fetch artifactUrl for zip download url
func fetchArtifactDownloadUrl(artifactUrl, token string) (string, error) {
	// Send the request to get artifact details
	resp, err := sendRequest(artifactUrl, []byte{}, token, "GET")
	if err != nil {
		return "", fmt.Errorf("failed to send request to artifact URL: %v", err)
	}
	defer resp.Body.Close()

	// Check for successful response
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get artifact details, status code: %d", resp.StatusCode)
	}

	// Parse the response to get the artifact download URL
	var artifactResp struct {
		Artifacts []struct {
			ArchiveDownloadURL string `json:"archive_download_url"`
		} `json:"artifacts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&artifactResp); err != nil {
		return "", fmt.Errorf("failed to parse artifact details: %v", err)
	}

	if len(artifactResp.Artifacts) == 0 {
		return "", fmt.Errorf("no artifact found")
	}

	// Get the artifact download URL
	artifactDownloadUrl := artifactResp.Artifacts[0].ArchiveDownloadURL
	if artifactDownloadUrl == "" {
		return "", fmt.Errorf("no valid artifact download URL found")
	}

	return artifactDownloadUrl, nil
}

// downloadFile downloads the artifact file from GitHub API and saves it locally
func downloadFile(artifactDownloadUrl, token string) ([]byte, error) {
	// Send the request
	resp, err := sendRequest(artifactDownloadUrl, []byte{}, token, "GET")
	if err != nil {
		return nil, fmt.Errorf("failed to send download request: %v", err)
	}
	defer resp.Body.Close()

	// Check if the response status code is OK (200)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download artifact, status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func DeleteSuccessWorkflow(owner, repo, token string, workflowID int64) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs/%d", owner, repo, workflowID)
	resp, err := sendRequest(url, []byte{}, token, "DELETE")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete workflow. Status code: %d", resp.StatusCode)
	}
	return nil
}
