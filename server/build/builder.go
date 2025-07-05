package build

import (
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"google.golang.org/protobuf/proto"
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

	GetBeaconID() uint32

	SetBeaconID(id uint32) error
}

func NewBuilder(req *clientpb.BuildConfig) (Builder, Builder, error) {
	var pulseBuilder Builder
	switch req.Source {
	case consts.ArtifactFromAction:
		pulseBuilder = NewActionBuilder(req)
	case consts.ArtifactFromDocker:
		pulseBuilder = NewDockerBuilder(req)
	case consts.ArtifactFromSaas:
		pulseBuilder = NewSaasBuilder(req)
	default:
		return nil, nil, errors.New("failed to create builder")
	}
	if req.Type == consts.CommandBuildPulse {
		var beaconBuilder Builder
		var artifactID uint32
		if req.ArtifactId != 0 {
			artifactID = req.ArtifactId
		} else {
			profile, _ := db.GetProfile(req.ProfileName)
			yamlID := profile.Pulse.Flags.ArtifactID
			if uint32(yamlID) != 0 {
				artifactID = yamlID
			} else {
				artifactID = 0
			}
		}
		builders, err := db.FindBeaconArtifact(artifactID, req.ProfileName)
		if err != nil {
			return nil, nil, err
		}
		if len(builders) > 0 {
			artifactID = builders[0].ID
			req.ArtifactId = artifactID
		} else {
			beaconReq := proto.Clone(req).(*clientpb.BuildConfig)
			beaconReq.Type = consts.CommandBuildBeacon
			if beaconReq.Target == consts.TargetX86Windows {
				beaconReq.Target = consts.TargetX86WindowsGnu
			} else {
				beaconReq.Target = consts.TargetX64WindowsGnu
			}
			switch beaconReq.Source {
			case consts.ArtifactFromAction:
				beaconBuilder = NewActionBuilder(beaconReq)
			case consts.ArtifactFromDocker:
				beaconBuilder = NewDockerBuilder(beaconReq)
			case consts.ArtifactFromSaas:
				beaconBuilder = NewSaasBuilder(beaconReq)
			}
			return beaconBuilder, pulseBuilder, nil
		}
	}
	return nil, pulseBuilder, nil
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
