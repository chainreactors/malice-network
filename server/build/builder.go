package build

import (
	"errors"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
)

var (
	ERRORSRDI    = errors.New("srdi error")
	ERROROBJCOPY = errors.New("objcopy error")
)

// Builder
type Builder interface {
	GenerateConfig() (*clientpb.Builder, error)

	ExecuteBuild() error

	CollectArtifact() (string, string)
}

func NewBuilder(req *clientpb.BuildConfig) Builder {
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
	ID     uint32 // Builder.ID
	Status string // 状态
}

const maxDockerBuildConcurrency = 2

var (
	// 用信号量控制最大并发数
	dockerBuildSemaphore = make(chan struct{}, maxDockerBuildConcurrency)
)
