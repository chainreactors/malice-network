package build

import (
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

type ActionBuilder struct {
	config     *clientpb.BuildConfig
	builder    *models.Artifact
	workflowID string
	profile    *implanttypes.ProfileConfig
}

func NewActionBuilder(req *clientpb.BuildConfig) *ActionBuilder {
	// todo
	//inputs := map[string]string{
	//	"package": req.BuildType,
	//	"targets": req.Target,
	//}
	// req.Inputs = inputs

	return &ActionBuilder{
		config: req,
	}
}

func (a *ActionBuilder) Generate() (*clientpb.Artifact, error) {
	// get config
	actionConfig := a.config.GetGithubAction()
	if actionConfig == nil {
		config := configs.GetGithubConfig()
		if config == nil {
			return nil, fmt.Errorf("please set github config use flag or server config")
		}
		actionConfig = &clientpb.GithubActionBuildConfig{
			Owner:      config.Owner,
			Repo:       config.Repo,
			Token:      config.Token,
			WorkflowId: config.Workflow,
		}
		a.config.SourceConfig = &clientpb.BuildConfig_GithubAction{
			GithubAction: actionConfig,
		}
	}
	if actionConfig.Owner == "" || actionConfig.Repo == "" || actionConfig.Token == "" {
		return nil, fmt.Errorf("incomplete github action configuration")
	}
	//
	var builder *models.Artifact
	//var err error
	//var profileParams types.ProfileParams
	//err = json.Unmarshal(a.config.ParamsBytes, &profileParams)
	//if err != nil {
	//	return nil, err
	//}
	//if profileParams.Modules != "" {
	//	a.config.Inputs["malefic_modules_features"] = profileParams.Modules
	//}
	//if profileParams.Enable3RD {
	//	a.config.Inputs["package"] = "3rd"
	//}
	//if a.config.BuildName == "" {
	//	a.config.BuildName = codenames.GetCodename()
	//}
	//profileByte, err := GenerateProfile(a.config)
	//if err != nil {
	//	return nil, err
	//}
	//if a.config.ArtifactId != 0 && a.config.Type == consts.CommandBuildBeacon {
	//	builder, err = db.SaveArtifactFromID(a.config, a.config.ArtifactId, a.config.Source, profileByte)
	//} else {
	//	builder, err = db.SaveArtifactFromConfig(a.config, profileByte)
	//}
	//if err != nil {
	//	logs.Log.Errorf("failed to save build: %s", err)
	//	return nil, err
	//}
	//a.builder = builder
	//a.config.Inputs["remark"] = a.builder.Name
	//a.config.ArtifactId = a.builder.ID
	//a.config.Inputs["remark"] = a.builder.Name
	//base64Encoded := encode.Base64Encode(profileByte)
	//a.config.Inputs["malefic_config_yaml"] = base64Encoded
	//profile, err := types.LoadProfile(profileByte)
	//if err != nil {
	//	return nil, err
	//}

	//a.profile = profile
	db.UpdateBuilderStatus(a.builder.ID, consts.BuildStatusWaiting)

	return builder.ToProtobuf([]byte{}), nil
}

func (a *ActionBuilder) Execute() error {
	db.UpdateBuilderStatus(a.builder.ID, consts.BuildStatusRunning)
	actionConfig := a.config.GetGithubAction()
	err := runWorkFlow(actionConfig.Owner, actionConfig.Repo, actionConfig.WorkflowId, actionConfig.Token, actionConfig.Inputs)
	if err != nil {
		db.UpdateBuilderStatus(a.builder.ID, consts.BuildStatusFailure)
		return err
	}
	return nil
}

func (a *ActionBuilder) Collect() (string, string, error) {
	actionConfig := a.config.GetGithubAction()
	path, err := downloadArtifactWhenReady(
		actionConfig.Owner,
		actionConfig.Repo,
		actionConfig.Token,
		actionConfig.IsRemove,
		//actionConfig.ArtifactId,
		0,
		a.builder,
	)
	if err == nil {
		return path, consts.BuildStatusCompleted, nil
	} else {
		return "", consts.BuildStatusFailure, err
	}
}

//func (a *ActionBuilder) GetBeaconID() uint32 {
//	return a.config.ArtifactId
//}
//
//func (a *ActionBuilder) SetBeaconID(id uint32) error {
//	a.config.ArtifactId = id
//	if a.config.Params == "" {
//		params := &types.ProfileParams{
//			OriginBeaconID: id,
//		}
//		a.config.Params = params.String()
//	} else {
//		var newParams *types.ProfileParams
//		err := json.Unmarshal([]byte(a.config.Params), &newParams)
//		if err != nil {
//			return err
//		}
//		newParams.OriginBeaconID = a.config.ArtifactId
//		a.config.Params = newParams.String()
//	}
//	return nil
//}
