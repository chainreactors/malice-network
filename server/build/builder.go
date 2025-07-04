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
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

var (
	ERRORSRDI    = errors.New("srdi error")
	ERROROBJCOPY = errors.New("objcopy error")
)

// Builder
type Builder interface {
	GenerateConfig() (*clientpb.Artifact, error)

	ExecuteBuild() error

	CollectArtifact() (string, string)
}

func NewBuilder(req *clientpb.BuildConfig) Builder {
	if req.Type == consts.CommandBuildPulse {
		if req.Params == "" {
			params := &types.ProfileParams{
				OriginBeaconID: req.ArtifactId,
			}
			req.Params = params.String()
		} else {
			var newParams *types.ProfileParams
			err := json.Unmarshal([]byte(req.Params), &newParams)
			if err != nil {
				logs.Log.Infof("failed to add artifact id: %s", err)
				return nil
			}
			newParams.OriginBeaconID = req.ArtifactId
			req.Params = newParams.String()
		}
	}
	switch req.Source {
	case consts.ArtifactFromAction:
		return NewActionBuilder(req)
	case consts.ArtifactFromDocker:
		return NewDockerBuilder(req)
	case consts.ArtifactFromSaas:
		return NewSaasBuilder(req)
	default:
		return nil
	}
}

type BuilderState struct {
	ID     uint32 // Artifact.ID
	Status string // 状态
}

const maxDockerBuildConcurrency = 2

var (
	// 用信号量控制最大并发数
	dockerBuildSemaphore = make(chan struct{}, maxDockerBuildConcurrency)
)

func SendBuildMsg(builder *models.Artifact, status string, msg string) {
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

func SendFailedMsg(builder *clientpb.Artifact) {
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventBuild,
		IsNotify:  false,
		Message:   fmt.Sprintf("Artifact failed %s (type: %s, target: %s, source: %s)", builder.Name, builder.Type, builder.Target, builder.Source),
		Important: true,
	})
}
