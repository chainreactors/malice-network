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
	"strings"
)

type ActionBuilder struct {
	config     *clientpb.BuildConfig
	builder    *models.Artifact
	workflowID string
	profile    *types.ProfileConfig
}

func NewActionBuilder(req *clientpb.BuildConfig) *ActionBuilder {
	inputs := map[string]string{
		"package": req.Type,
		"targets": req.Target,
	}
	if len(req.Modules) > 0 {
		inputs["malefic_modules_features"] = strings.Join(req.Modules, ",")
	}
	req.Inputs = inputs

	return &ActionBuilder{
		config: req,
	}
}

func (a *ActionBuilder) GenerateConfig() (*clientpb.Artifact, error) {
	githubConfig := a.config.Github
	if githubConfig != nil {
		if githubConfig.Owner == "" || githubConfig.Repo == "" || githubConfig.Token == "" {
			config := configs.GetGithubConfig()
			if config == nil {
				return nil, fmt.Errorf("please set github config use flag or server config")
			}
			githubConfig.Owner = config.Owner
			githubConfig.Repo = config.Repo
			githubConfig.Token = config.Token
			githubConfig.WorkflowId = config.Workflow
		}
	}

	var builder *models.Artifact
	var err error
	profileByte, err := GenerateProfile(a.config)
	if err != nil {
		return builder.ToArtifact([]byte{}), err
	}
	if a.config.ArtifactId != 0 && a.config.Type == consts.CommandBuildBeacon {
		builder, err = db.SaveArtifactFromID(a.config, a.config.ArtifactId, a.config.Source, profileByte)
	} else {
		if a.config.BuildName == "" {
			a.config.BuildName = codenames.GetCodename()
		}
		builder, err = db.SaveArtifactFromConfig(a.config, profileByte)
	}
	if err != nil {
		logs.Log.Errorf("failed to save build%s: %s", builder.Name, err)
		return nil, err
	}
	a.builder = builder
	a.config.Inputs["remark"] = a.builder.Name
	a.config.ArtifactId = a.builder.ID
	a.config.Inputs["remark"] = a.builder.Name
	base64Encoded := encode.Base64Encode(profileByte)
	a.config.Inputs["malefic_config_yaml"] = base64Encoded
	profile, err := types.LoadProfile(profileByte)
	if err != nil {
		return builder.ToArtifact([]byte{}), err
	}

	a.profile = profile
	db.UpdateBuilderStatus(a.builder.ID, consts.BuildStatusWaiting)

	return builder.ToArtifact([]byte{}), nil
}

func (a *ActionBuilder) ExecuteBuild() error {
	if len(a.config.Modules) == 0 {
		a.config.Modules = a.profile.Implant.Modules
	}
	db.UpdateBuilderStatus(a.builder.ID, consts.BuildStatusRunning)

	err := runWorkFlow(a.config.Github.Owner, a.config.Github.Repo, a.config.Github.WorkflowId, a.config.Github.Token, a.config.Inputs)
	if err != nil {
		db.UpdateBuilderStatus(a.builder.ID, consts.BuildStatusFailure)
		return err
	}
	return nil
}

func (a *ActionBuilder) CollectArtifact() (string, string) {
	go downloadArtifactWhenReady(a.config.Github.Owner, a.config.Github.Repo, a.config.Github.Token, a.config.Github.IsRemove, a.config.ArtifactId, a.builder)
	return a.builder.Path, ""
}
