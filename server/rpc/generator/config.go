package generator

import (
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/config"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"path/filepath"
)

var generateConfig = "config.yaml"

func DbToConfig(req *clientpb.Generate) error {
	var profileDB models.Profile
	var profile configs.GeneratorConfig
	var err error
	if req.Name != "" {
		profileDB, err = db.GetProfile(req.Name)
		if err != nil {
			return err
		}
	}
	path := filepath.Join(configs.BuildPath, generateConfig)
	err = config.LoadConfig(path, &profile)
	if err != nil {
		return err
	}
	err = profileDB.UpdateGeneratorConfig(profile, req, path)
	if err != nil {
		return err
	}
	return nil
}
