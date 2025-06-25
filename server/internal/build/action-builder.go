package build

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/chainreactors/utils/encode"
)

type ActionBuilder struct {
	config     *clientpb.BuildConfig
	builder    *models.Builder
	workflowID string
	profile    *types.ProfileConfig
}

func NewActionBuilder(req *clientpb.BuildConfig) *ActionBuilder {
	req.Target = req.Inputs["targets"]
	req.Type = req.Inputs["package"]
	return &ActionBuilder{
		config: req,
	}
}

func (a *ActionBuilder) GenerateConfig() (*clientpb.Builder, error) {
	if a.config.Owner == "" || a.config.Repo == "" || a.config.Token == "" {
		config := configs.GetGithubConfig()
		if config == nil {
			return nil, fmt.Errorf("please set github config use flag or server config")
		}
		a.config.Owner = config.Owner
		a.config.Repo = config.Repo
		a.config.Token = config.Token
		a.config.WorkflowId = config.Workflow
	}
	var builder *models.Builder
	var err error
	if a.config.ArtifactId != 0 && a.config.Type == consts.CommandBuildBeacon {
		builder, err = db.SaveArtifactFromID(a.config, a.config.ArtifactId, a.config.Resource)
	} else {
		if a.config.BuildName == "" {
			a.config.BuildName = codenames.GetCodename()
		}
		builder, err = db.SaveArtifactFromConfig(a.config)
	}
	if err != nil {
		logs.Log.Errorf("save build db error: %v", err)
		return nil, err
	}
	a.builder = builder
	a.config.Inputs["remark"] = a.builder.Name
	a.config.ArtifactId = a.builder.ID
	profileByte, err := GenerateProfile(a.config)
	if err != nil {
		return builder.ToProtobuf(), err
	}
	a.config.Inputs["remark"] = a.builder.Name
	base64Encoded := encode.Base64Encode(profileByte)
	a.config.Inputs["malefic_config_yaml"] = base64Encoded
	profile, err := types.LoadProfile(profileByte)
	if err != nil {
		return builder.ToProtobuf(), err
	}

	a.profile = profile
	db.UpdateBuilderStatus(a.builder.ID, consts.BuildStatusWaiting)

	return builder.ToProtobuf(), nil
}

func (a *ActionBuilder) ExecuteBuild() error {
	if len(a.config.Modules) == 0 {
		a.config.Modules = a.profile.Implant.Modules
	}
	db.UpdateBuilderStatus(a.builder.ID, consts.BuildStatusRunning)

	err := runWorkFlow(a.config.Owner, a.config.Repo, a.config.WorkflowId, a.config.Token, a.config.Inputs)
	if err != nil {
		db.UpdateBuilderStatus(a.builder.ID, consts.BuildStatusFailure)
		return err
	}
	return nil
}

func (a *ActionBuilder) CollectArtifact() {
	go downloadArtifactWhenReady(a.config.Owner, a.config.Repo, a.config.Token, a.config.IsRemove, a.builder)
	return
}
