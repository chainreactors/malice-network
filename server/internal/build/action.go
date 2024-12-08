package build

import (
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
	"github.com/chainreactors/utils/encode"
	"net/http"
	"net/url"
	"time"
)

// TriggerWorkflowDispatch is a reusable function to trigger a GitHub Actions workflow dispatch event
func TriggerWorkflowDispatch(owner, repo, workflowID, token string, inputs map[string]string, req *clientpb.Generate) (*clientpb.Builder, error) {
	config, err := GenerateProfile(req)
	if err != nil {
		return nil, err
	}
	profile, err := types.LoadProfile([]byte(config))
	if err != nil {
		return nil, err
	}

	base64Encoded := encode.Base64Encode([]byte(config))

	escapedOwner := url.QueryEscape(owner)
	escapedRepo := url.QueryEscape(repo)
	escapedToken := url.QueryEscape(token)

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
			beaconReq["malefic_config_yaml"] = base64Encoded
			beaconReq["package"] = consts.CommandBuildBeacon
			if len(req.Modules) == 0 {
				req.Modules = profile.Implant.Modules
			}
			beaconBuilder, err := db.SaveBuilderFromAction(beaconReq, req)
			beaconReq["remark"] = beaconBuilder.Name
			if beaconReq["targets"] == consts.TargetX86Windows {
				beaconReq["targets"] = consts.TargetX86WindowsGnu
			} else {
				beaconReq["targets"] = consts.TargetX64WindowsGnu
			}
			if err != nil {
				return nil, fmt.Errorf("failed to save beacon builder: %v", err)
			}
			err = triggerBuildBeaconWorkflow(escapedOwner, escapedRepo, workflowID, escapedToken, beaconReq)
			if err != nil {
				return nil, err
			}
			beaconBuilder.IsSRDI = true
			go downloadArtifactWhenReady(escapedOwner, escapedRepo, escapedToken, beaconBuilder)
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
			_, err := SRDIArtifact(idBuilder, target.OS, target.Arch)
			if err != nil {
				return nil, err
			}
		}
	}
	inputs["malefic_config_yaml"] = base64Encoded
	if len(req.Modules) == 0 {
		req.Modules = profile.Implant.Modules
	}
	builder, err := db.SaveBuilderFromAction(inputs, req)
	if err != nil {
		return nil, fmt.Errorf("failed to save builder: %v", err)
	}
	inputs["remark"] = builder.Name
	err = triggerBuildBeaconWorkflow(escapedOwner, escapedRepo, workflowID, escapedToken, inputs)
	if err != nil {
		return nil, err
	}
	go downloadArtifactWhenReady(escapedOwner, escapedRepo, escapedToken, builder)
	return builder.ToProtobuf(), nil
}

// triggerBuildBeaconWorkflow triggers the BuildBeacon workflow when artifact is not found
func triggerBuildBeaconWorkflow(owner, repo, workflowID, token string, inputs map[string]string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/workflows/%s/dispatches", owner, repo, workflowID)
	payload := WorkflowDispatchPayload{
		Ref:    "master",
		Inputs: inputs,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize payload for BuildBeacon: %v", err)
	}
	resp, err := sendRequest(url, body, token, "POST")
	if err != nil {
		return err
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
		err := PushArtifact(owner, repo, token, builder.Name)
		if err == nil {
			logs.Log.Info("Artifact downloaded successfully!")
			if builder.IsSRDI {
				_, err := SRDIArtifact(builder, builder.Os, builder.Arch)
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
		time.Sleep(30 * time.Second)

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
