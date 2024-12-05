package build

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"net/http"
	"time"
)

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
	Conclusion   string    `json:"conclusion"`
	URL          string    `json:"url"`
	HTMLURL      string    `json:"html_url"`
	ArtifactsURL string    `json:"artifacts_url"`
}

// ToProtoBuf converts the Workflow struct to its corresponding Protocol Buffers message
func (w *Workflow) ToProtoBuf() *clientpb.Workflow {
	createdAt := w.CreatedAt.Format(time.RFC3339)
	updatedAt := w.UpdatedAt.Format(time.RFC3339)

	return &clientpb.Workflow{
		Id:        w.ID,
		NodeId:    w.NodeID,
		Name:      w.Name,
		Path:      w.Path,
		Status:    w.Status,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Url:       w.URL,
		HtmlUrl:   w.HTMLURL,
	}
}

// WorkflowDispatchPayload defines the payload for triggering a workflow dispatch event
type WorkflowDispatchPayload struct {
	Ref    string            `json:"ref"`
	Inputs map[string]string `json:"inputs,omitempty"`
}

// TriggerWorkflowDispatch is a reusable function to trigger a GitHub Actions workflow dispatch event
func TriggerWorkflowDispatch(owner, repo, workflowID, token string, inputs map[string]string, req *clientpb.Generate) (*clientpb.Builder, error) {
	config, err := GenerateProfile(req)
	profile, err := types.LoadProfile([]byte(config))
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Err create config: %v", err))
	}
	if inputs["package"] == consts.CommandBuildPulse {
		var artifactID uint32
		if req.ArtifactId != 0 {
			artifactID = req.ArtifactId
		} else {
			yamlID := profile.Pulse.Extras["flags"].(map[string]interface{})["artifact_id"].(int)
			if uint32(yamlID) != 0 {
				artifactID = uint32(yamlID)
			}
			artifactID = 0
		}
		idBuilder, err := db.GetArtifactById(artifactID)
		if err != nil && !errors.Is(err, db.ErrRecordNotFound) {
			return nil, err
		} else if errors.Is(err, db.ErrRecordNotFound) {
			beaconReq := copyMap(inputs)
			base64Encoded := base64.StdEncoding.EncodeToString([]byte(config))
			beaconReq["malefic_config_yaml"] = base64Encoded
			beaconReq["package"] = consts.CommandBuildBeacon
			if len(req.Modules) == 0 {
				req.Modules = profile.Implant.Modules
			}
			beaconBuilder, err := db.SaveBuiladerFromAction(beaconReq, req)
			beaconReq["remark"] = beaconBuilder.Name
			if err != nil {
				return nil, fmt.Errorf("failed to save beacon builder: %v", err)
			}
			err = triggerBuildBeaconWorkflow(owner, repo, workflowID, token, beaconReq)
			if err != nil {
				return nil, err
			}
			beaconBuilder.IsSRDI = true
			go downloadArtifactWhenReady(owner, repo, token, beaconBuilder)
			req.ArtifactId = beaconBuilder.ID
			_, err = GenerateProfile(req)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("Err create config: %v", err))
			}
		} else if !idBuilder.IsSRDI {
			idBuilder.IsSRDI = true
			target, ok := consts.GetBuildTarget(inputs["targets"])
			if !ok {
				return nil, err
			}
			_, _, err := UploadSrdiArtifact(idBuilder, target.OS, target.Arch)
			if err != nil {
				return nil, err
			}
		}
	}
	// Create the payload
	if inputs["package"] != consts.CommandBuildPrelude {
		base64Encoded := base64.StdEncoding.EncodeToString([]byte(config))
		inputs["malefic_config_yaml"] = base64Encoded
	}
	if len(req.Modules) == 0 {
		req.Modules = profile.Implant.Modules
	}
	builder, err := db.SaveBuiladerFromAction(inputs, req)
	if err != nil {
		return nil, fmt.Errorf("failed to save builder: %v", err)
	}
	inputs["remark"] = builder.Name
	err = triggerBuildBeaconWorkflow(owner, repo, workflowID, token, inputs)
	if err != nil {
		return nil, err
	}
	go downloadArtifactWhenReady(owner, repo, token, builder)
	return builder.ToProtobuf([]byte{}), nil
}

// triggerBuildBeaconWorkflow triggers the BuildBeacon workflow when artifact is not found
func triggerBuildBeaconWorkflow(owner, repo, workflowID, token string, inputs map[string]string) error {
	// Construct the GitHub API URL
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/workflows/%s/dispatches", owner, repo, workflowID)

	// Create the payload for the BuildBeacon dispatch event
	payload := WorkflowDispatchPayload{
		Ref:    "master",
		Inputs: inputs,
	}

	// Marshal the payload into JSON
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize payload for BuildBeacon: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request for BuildBeacon: %v", err)
	}

	// Set the headers for GitHub API
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send BuildBeacon request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to trigger BuildBeacon workflow. Status code: %d", resp.StatusCode)
	}

	logs.Log.Info("BuildBeacon workflow triggered successfully!")
	return nil
}

// downloadArtifactWhenReady waits for the artifact to be ready and downloads it
func downloadArtifactWhenReady(owner, repo, token string, builder *models.Builder) {
	for {
		time.Sleep(30 * time.Second)

		// Attempt to download the artifact
		_, err := DownloadArtifact(owner, repo, token, builder.Name)
		if err == nil {
			logs.Log.Info("Artifact downloaded successfully!")
			if !builder.IsSRDI {
				_, _, err := UploadSrdiArtifact(builder, builder.Os, builder.Arch)
				if err != nil {
					logs.Log.Errorf("action to srdi failed")
				}
			}
			break
		} else if errors.Is(err, errs.ErrWorkflowFailed) {
			logs.Log.Errorf("Download artifact failed due to workflow failure: %s", err)
			break
		} else {
			logs.Log.Debugf("Download artifact failed: %s", err)
		}
	}
}

// copyMap creates a shallow copy of the input map
func copyMap(original map[string]string) map[string]string {
	c := make(map[string]string, len(original))
	for k, v := range original {
		c[k] = v
	}
	return c
}

// EnableWorkflow enables a GitHub Actions workflow in a specified repository
func EnableWorkflow(owner, repo, workflowID, token string) error {
	// Construct the GitHub API URL
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/workflows/%s/enable", owner, repo, workflowID)

	// Create the HTTP request
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", apiVersion)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusNoContent {
		logs.Log.Info("Workflow enabled successfully!")
		return nil
	}

	return fmt.Errorf("failed to enable workflow. Status code: %d", resp.StatusCode)
}

// DisableWorkflow disables a GitHub Actions workflow in a specified repository
func DisableWorkflow(owner, repo, workflowID, token string) error {
	// Construct the GitHub API URL
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/workflows/%s/disable", owner, repo, workflowID)

	// Create the HTTP request
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", apiVersion)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode == http.StatusNoContent {
		logs.Log.Info("Workflow disabled successfully!")
		return nil
	}

	return fmt.Errorf("failed to disable workflow. Status code: %d", resp.StatusCode)
}

// ListRepositoryWorkflows fetches the workflows for a given repository
func ListRepositoryWorkflows(owner, repo, token string) ([]Workflow, error) {
	// Construct the GitHub API URL
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs", owner, repo)

	// Create the HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", apiVersion)

	resp, err := httpClient.Do(req)
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

// base64ToBytes decodes a Base64 encoded string and returns the resulting bytes
func base64ToBytes(encoded string) ([]byte, error) {
	// Decode the Base64 encoded string
	decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Base64 string: %v", err)
	}
	return decodedBytes, nil
}
