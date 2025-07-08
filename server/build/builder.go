package build

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/cryptography"
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

func SendBuildMsg(builder *clientpb.Artifact, status string, msg string) {
	if core.EventBroker == nil {
		return
	}
	if status == consts.BuildStatusCompleted {
		msg = fmt.Sprintf("Artifact completed %s (type: %s, target: %s, source: %s)", builder.Name, builder.Type, builder.Target, builder.Source)
	} else if status == consts.BuildStatusFailure {
		msg = fmt.Sprintf("Artifact failed %s (type: %s, target: %s, source: %s)", builder.Name, builder.Type, builder.Target, builder.Source)
	} else {
		return
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventBuild,
		IsNotify:  false,
		Message:   msg,
		Important: true,
	})
}

func AmountArtifact(artifactName string) error {
	var result []*clientpb.Pipeline
	encrypt, err := cryptography.EncryptWithGlobalKey([]byte(artifactName))
	if err != nil {
		return err
	}
	hexEncrypt := cryptography.BytesToHex(encrypt)
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
		content := &clientpb.WebContent{
			Path: hexEncrypt,
		}
		lns.PushCtrl(&clientpb.JobCtrl{
			Ctrl: consts.CtrlWebContentAddArtifact,
			Job: &clientpb.Job{
				Pipeline: pipe,
			},
			Content: content,
		})
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventBuild,
		IsNotify:  false,
		Message:   fmt.Sprintf("artifact %s amounts at /%s", artifactName, hexEncrypt),
		Important: true,
	})
	return nil
}
