package build

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const artifactApiURL = "https://api.github.com/repos/%s/%s/actions/artifacts"
const downloadApiUrl = "https://api.github.com/repos/%s/%s/actions/artifacts/%s/%s"

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

func (a *Artifact) ToProtoBuf() *clientpb.Artifact {
	createdAt := a.CreatedAt.Format(time.RFC3339)
	expiresAt := a.ExpiresAt.Format(time.RFC3339)
	updatedAt := a.UpdatedAt.Format(time.RFC3339)

	return &clientpb.Artifact{
		Id:                 strconv.FormatInt(a.ID, 10),
		NodeId:             a.NodeID,
		Name:               a.Name,
		SizeInBytes:        a.SizeInBytes,
		ArchiveDownloadUrl: a.ArchiveDownloadURL,
		Expired:            strconv.FormatBool(a.Expired),
		CreatedAt:          createdAt,
		ExpiresAt:          expiresAt,
		UpdatedAt:          updatedAt,
	}
}

// ArtifactsResponse is the response structure for listing artifacts
type ArtifactsResponse struct {
	TotalCount int        `json:"total_count"`
	Artifacts  []Artifact `json:"artifacts"`
}

// ListArtifacts lists artifacts for a GitHub repository
func ListArtifacts(owner, repo, token string) ([]Artifact, error) {
	// Create the API URL
	escapedOwner := url.QueryEscape(owner)
	escapedRepo := url.QueryEscape(repo)
	apiUrl := fmt.Sprintf(artifactApiURL, escapedOwner, escapedRepo)

	req, err := createGitHubRequest(apiUrl, token)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Send the request
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check for successful response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list artifacts. Status code: %d", resp.StatusCode)
	}

	// Parse the response
	var result ArtifactsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return result.Artifacts, nil
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

// DownloadArtifact downloads the specified artifact
func DownloadArtifact(owner, repo, token, buildName string) (*clientpb.DownloadArtifactsResponse, error) {
	builder, err := db.GetArtifactByName(buildName)
	if err != nil {
		return nil, err
	}
	if builder.Path != "" {
		file, err := os.ReadFile(builder.Path)
		if err != nil {
			return nil, err
		}
		return &clientpb.DownloadArtifactsResponse{
			Zip:  file,
			Name: filepath.Base(builder.Path),
		}, nil
	}

	artifactDownloadUrl, err := getArtifactDownloadUrl(owner, repo, token, buildName)
	if err != nil {
		return nil, err
	}

	zipPath, err := downloadFile(artifactDownloadUrl, token, buildName)
	if err != nil {
		return nil, fmt.Errorf("download artifact failed: %v", err)
	}

	resultPath, fileByte, err := Unzip(zipPath, configs.BuildOutputPath, buildName)
	if err != nil {
		return nil, fmt.Errorf("unzip artifact failed: %v", err)
	}

	builder.Path = resultPath
	err = db.UpdateBuilderPath(builder)
	if err != nil {
		return nil, err
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventBuild,
		IsNotify:  false,
		Message:   fmt.Sprintf("action %s type %s has finished. yon can run `artifact download %s` ", builder.Name, builder.Type, builder.Name),
	})
	return &clientpb.DownloadArtifactsResponse{
		Zip:  fileByte,
		Name: filepath.Base(resultPath),
	}, nil
}

func Unzip(zipFile, outputDir, name string) (string, []byte, error) {
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return "", nil, fmt.Errorf("error opening ZIP file: %v", err)
	}
	defer zipReader.Close()
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", nil, fmt.Errorf("error creating output directory: %v", err)
	}
	if len(zipReader.File) > 1 {
		return "", nil, fmt.Errorf("error: multiple files in zip")
	}
	file := zipReader.File[0]
	filePath := filepath.Join(outputDir, name+filepath.Ext(file.Name))
	if file.FileInfo().IsDir() {
		return "", nil, fmt.Errorf("error extracting directory: %v", file.Name)
	}
	dstFile, err := os.Create(filePath)
	if err != nil {
		return "", nil, fmt.Errorf("error creating file: %v", err)
	}
	defer dstFile.Close()
	srcFile, err := file.Open()
	if err != nil {
		return "", nil, fmt.Errorf("error opening file inside ZIP: %v", err)
	}
	defer srcFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return "", nil, fmt.Errorf("error copying file contents: %v", err)
	}
	fileByte, err := os.ReadFile(filePath)
	if err != nil {
		return "", nil, fmt.Errorf("error reading file: %v", err)
	}
	return filePath, fileByte, nil
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
func downloadFile(artifactDownloadUrl, token, buildName string) (string, error) {
	// Create the request with the correct headers to download the artifact
	req, err := createGitHubRequest(artifactDownloadUrl, token)
	if err != nil {
		return "", fmt.Errorf("failed to create download request: %v", err)
	}

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send download request: %v", err)
	}
	defer resp.Body.Close()

	// Check if the response status code is OK (200)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download artifact, status code: %d", resp.StatusCode)
	}

	// Define the path to save the artifact file
	zipPath := filepath.Join(configs.TempPath, buildName+"."+archiveFormat)

	// Create the file to save the artifact
	outFile, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file to save artifact: %v", err)
	}
	defer outFile.Close()

	// Copy the response body directly into the file to optimize memory usage
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write response body to file: %v", err)
	}

	return zipPath, nil
}
