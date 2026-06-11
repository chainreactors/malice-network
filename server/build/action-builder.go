package build

import (
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/implanttypes"
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
	return nil, fmt.Errorf("github action builder is not yet implemented")
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
