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
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

const archiveFormat = "zip"

var notifiedWorkflows = make(map[string]bool)

// createGitHubRequest
func createGitHubRequest(url, token string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	return req, nil
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

// ArtifactsResponse is the response structure for listing artifacts
type ArtifactsResponse struct {
	TotalCount int        `json:"total_count"`
	Artifacts  []Artifact `json:"artifacts"`
}

// findArtifactsURL finds the ArtifactsURL for a workflow by name
func findArtifactsURL(workflows []Workflow, name string) (string, error) {
	for _, wf := range workflows {
		if wf.Name == name {
			if wf.Status == "completed" && wf.Conclusion == "success" {
				return wf.ArtifactsURL, nil
			} else if wf.Conclusion == "failure" {
				return "", errs.ErrWorkflowFailed
			}
			if !notifiedWorkflows[name] {
				core.EventBroker.Publish(core.Event{
					EventType: consts.EventBuild,
					IsNotify:  false,
					Message:   fmt.Sprintf("action %s run in %s.", name, wf.HTMLURL),
				})
				notifiedWorkflows[name] = true
			}
		}
	}
	return "", errors.New("no artifact found") // Return empty string if not found
}

func getArtifact(owner, repo, token, buildName string) (string, error) {
	// Build the download URL
	escapedOwner := url.QueryEscape(owner)
	escapedRepo := url.QueryEscape(repo)
	escapedToken := url.QueryEscape(token)
	escapedBuildName := url.QueryEscape(buildName)

	workflows, err := ListRepositoryWorkflows(escapedOwner, escapedRepo, escapedToken)
	if err != nil {
		return "", err
	}
	artifactUrl, err := findArtifactsURL(workflows, escapedBuildName)
	if err != nil {
		return "", err
	}
	return artifactUrl, nil
}

// fetchArtifactDownloadUrl  fetch artifactUrl for zip download url
func fetchArtifactDownloadUrl(artifactUrl, token string) (string, error) {
	req, err := createGitHubRequest(artifactUrl, token)
	if err != nil {
		return "", fmt.Errorf("failed to create request for artifactUrl: %v", err)
	}

	// Send the request to get artifact details
	resp, err := httpClient.Do(req)
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

func PushArtifact(owner, repo, token, buildName string) error {
	builder, err := db.GetArtifactByName(buildName)
	if err != nil {
		return err
	}
	if builder.Path != "" {
		_, err := os.ReadFile(builder.Path)
		if err != nil {
			return err
		}
		return nil
	}

	artifactDownloadUrl, err := getArtifactDownloadUrl(owner, repo, token, buildName)
	if err != nil {
		return err
	}

	raw, err := downloadFile(artifactDownloadUrl, token, buildName)
	if err != nil {
		return fmt.Errorf("download artifact failed: %v", err)
	}

	content, err := fileutils.UnzipOneWithBytes(raw)
	if err != nil {
		return fmt.Errorf("unzip artifact failed: %v", err)
	}
	filename := filepath.Join(configs.BuildOutputPath, encoders.UUID())
	err = os.WriteFile(filename, content, 0644)
	if err != nil {
		return err
	}
	builder.Path = filename
	err = db.UpdateBuilderPath(builder)
	if err != nil {
		return err
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventBuild,
		IsNotify:  false,
		Message:   fmt.Sprintf("action %s %s %s has finished", builder.Name, builder.Type, builder.Target),
	})
	return nil
}

// getArtifactDownloadUrl retrieves the artifact download URL from the GitHub API response
func getArtifactDownloadUrl(owner, repo, token, buildName string) (string, error) {
	// Build the query URL
	escapedOwner := url.QueryEscape(owner)
	escapedRepo := url.QueryEscape(repo)
	escapedToken := url.QueryEscape(token)
	escapedBuildName := url.QueryEscape(buildName)

	// Get the artifact list
	artifactUrl, err := getArtifact(escapedOwner, escapedRepo, escapedToken, escapedBuildName)
	if err != nil && !errors.Is(err, errs.ErrWorkflowFailed) {
		return "", fmt.Errorf("failed to get artifact URL: %v", err)
	} else if errors.Is(err, errs.ErrWorkflowFailed) {
		return "", err
	}

	artifactDownloadUrl, err := fetchArtifactDownloadUrl(artifactUrl, token)
	if err != nil {
		return "", fmt.Errorf("failed to fetch artifact download URL: %v", err)
	}
	return artifactDownloadUrl, nil
}

// downloadFile downloads the artifact file from GitHub API and saves it locally
func downloadFile(artifactDownloadUrl, token, buildName string) ([]byte, error) {
	// Create the request with the correct headers to download the artifact
	req, err := createGitHubRequest(artifactDownloadUrl, token)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %v", err)
	}

	// Send the request
	resp, err := http.DefaultClient.Do(req)
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
