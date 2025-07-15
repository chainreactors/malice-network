package build

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
)

var (
	ERRORSRDI    = errors.New("srdi error")
	ERROROBJCOPY = errors.New("objcopy error")
)

// Builder
type Builder interface {
	Generate() (*clientpb.Artifact, error)

	Execute() error

	Collect() (string, string)
}

func NewBuilder(req *clientpb.BuildConfig) (Builder, error) {
	if req.Type == consts.CommandBuildPulse {
		if req.ArtifactId == 0 {
			profile, err := db.GetProfile(req.ProfileName)
			if err != nil {
				return nil, err
			}
			req.ArtifactId = profile.Pulse.Flags.ArtifactID
		}

		if len(req.ParamsBytes) > 0 {
			var newParams types.ProfileParams
			err := json.Unmarshal(req.ParamsBytes, &newParams)
			if err != nil {
				return nil, err
			}
			newParams.OriginBeaconID = req.ArtifactId
			req.ParamsBytes = []byte(newParams.String())
		}
	}
	var builder Builder
	switch req.Source {
	case consts.ArtifactFromAction:
		builder = NewActionBuilder(req)
	case consts.ArtifactFromDocker:
		builder = NewDockerBuilder(req)
	case consts.ArtifactFromSaas:
		builder = NewSaasBuilder(req)
	default:
		return nil, errors.New("failed to create builder")
	}

	return builder, nil
}

type BuilderState struct {
	ID     uint32 // Artifact.ID
	Status string // 状态
}

const maxDockerBuildConcurrency = 1

var (
	// 用信号量控制最大并发数
	dockerBuildSemaphore = make(chan struct{}, maxDockerBuildConcurrency)
)

func SendBuildMsg(artifact *clientpb.Artifact, status string, params []byte) {
	if core.EventBroker == nil {
		return
	}
	event := core.Event{
		EventType: consts.EventBuild,
		IsNotify:  false,
		Important: true,
	}
	if status == consts.BuildStatusCompleted {
		event.Message = fmt.Sprintf("Artifact completed %s (type: %s, target: %s, source: %s)", artifact.Name, artifact.Type, artifact.Target, artifact.Source)
		profileParams, err := types.UnmarshalProfileParams(params)
		if err != nil {
			logs.Log.Errorf("failed to unmarshal profile params: %v", err)
			return
		}
		if profileParams.AutoDownload {
			event.Op = consts.CtrlArtifactDownload
			event.Job = &clientpb.Job{Name: artifact.Name}
		}

	} else if status == consts.BuildStatusFailure {
		event.Message = fmt.Sprintf("Artifact failed %s (type: %s, target: %s, source: %s)", artifact.Name, artifact.Type, artifact.Target, artifact.Source)
	} else {
		return
	}
	core.EventBroker.Publish(event)
}

func AmountArtifact(artifactName string) error {
	var result []*clientpb.Pipeline
	core.Listeners.Range(func(key, value any) bool {
		lns := value.(*core.Listener)
		for _, pipeline := range lns.AllPipelines() {
			if pipeline.Type == "website" {
				result = append(result, pipeline)
			}
		}
		return true
	})
	for _, pipe := range result {
		lns, _ := core.Listeners.Get(pipe.ListenerId)
		lns.PushCtrl(&clientpb.JobCtrl{
			Ctrl: consts.CtrlWebContentAddArtifact,
			Job: &clientpb.Job{
				Pipeline: pipe,
			},
			Content: &clientpb.WebContent{
				Path: artifactName,
			},
		})
	}
	return nil
}
