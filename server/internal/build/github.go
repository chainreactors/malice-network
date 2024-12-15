package build

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/chainreactors/malice-network/helper/errs"
	"net/http"
	"time"
)

// Define common HTTP client and API version
var httpClient = &http.Client{
	Timeout: 60 * time.Second,
}

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

// ArtifactsResponse is the response structure for listing artifacts
type ArtifactsResponse struct {
	TotalCount int        `json:"total_count"`
	Artifacts  []Artifact `json:"artifacts"`
}

// sendRequest sends request with JSON body and returns the response
func sendRequest(url string, payload []byte, token string, reqType string) (*http.Response, error) {
	var req *http.Request
	var err error
	if len(payload) > 0 {
		req, err = http.NewRequest(reqType, url, bytes.NewBuffer(payload))
	} else {
		req, err = http.NewRequest(reqType, url, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request: %v", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	return resp, nil
}

func GetWorkflowStatus(owner, repo, workflowID, token string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/workflows/%s", owner, repo, workflowID)
	resp, err := sendRequest(url, []byte{}, token, "GET")
	if err != nil {
		return fmt.Errorf("failed to send request to workflow URL: %v", err)
	}
	defer resp.Body.Close()

	// Check for successful response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get workflow details, status code: %d", resp.StatusCode)
	}

	var workflow Workflow
	if err := json.NewDecoder(resp.Body).Decode(&workflow); err != nil {
		return fmt.Errorf("failed to parse workflow details: %v", err)
	}
	if workflow.State != "active" {
		return errs.ErrorDockerNotActive
	}

	return nil
}
