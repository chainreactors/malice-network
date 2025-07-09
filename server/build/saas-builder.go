package build

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/httputils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/chainreactors/malice-network/server/internal/saas"
	"github.com/chainreactors/utils/encode"
	"google.golang.org/protobuf/encoding/protojson"
	"net/http"
	"time"
)

type SaasBuilder struct {
	config      *clientpb.BuildConfig
	builder     *models.Artifact
	executeUrl  string
	downloadUrl string
}

func NewSaasBuilder(req *clientpb.BuildConfig) *SaasBuilder {
	return &SaasBuilder{
		config: req,
	}
}

func (s *SaasBuilder) Generate() (*clientpb.Artifact, error) {
	saasConfig := configs.GetSaasConfig()
	if !saasConfig.Enable {
		return nil, errors.New("saas is unable in server config")
	}
	var builder *models.Artifact
	var err error
	profileByte, err := GenerateProfile(s.config)
	if err != nil {
		return nil, err
	}
	base64Encoded := encode.Base64Encode(profileByte)
	s.config.Inputs = make(map[string]string)
	s.config.Inputs["malefic_config_yaml"] = base64Encoded
	paramsBase64Encoded := encode.Base64Encode(s.config.ParamsBytes)
	s.config.Inputs["build_params"] = paramsBase64Encoded
	if s.config.ArtifactId != 0 && s.config.Type == consts.CommandBuildBeacon {
		builder, err = db.SaveArtifactFromID(s.config, s.config.ArtifactId, s.config.Source, profileByte)
	} else {
		if s.config.BuildName == "" {
			s.config.BuildName = codenames.GetCodename()
		}
		builder, err = db.SaveArtifactFromConfig(s.config, profileByte)
	}
	if err != nil {
		logs.Log.Errorf("failed to save build %s: %s", builder.Name, err)
		return nil, err
	}
	s.builder = builder
	db.UpdateBuilderStatus(s.builder.ID, consts.BuildStatusWaiting)
	s.executeUrl = fmt.Sprintf("%s/api/build", saasConfig.Url)
	return builder.ToProtobuf([]byte{}), nil
}

func (s *SaasBuilder) Execute() error {
	data, err := protojson.Marshal(s.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config %s: %s", s.config.ProfileName, err)
	}
	headers := saas.SaasHeaders(s.getToken())
	var respObj clientpb.Artifact
	err = httputils.DoJSONRequest("POST", s.executeUrl, bytes.NewReader(data), headers, http.StatusOK, &respObj)
	if err != nil {
		db.UpdateBuilderStatus(s.builder.ID, consts.BuildStatusFailure)
		return fmt.Errorf("failed to post saas service: %w", err)
	}
	return nil
}

func (s *SaasBuilder) Collect() (string, string) {
	statusUrl := fmt.Sprintf("/api/build/status/%s", s.builder.Name)
	downloadUrl := fmt.Sprintf("/api/build/download/%s", s.builder.Name)

	path, status, err := saas.CheckAndDownloadArtifact(statusUrl, downloadUrl, s.getToken(), s.builder, 30*time.Second, 30*time.Minute)
	if err != nil {
		logs.Log.Errorf("failed to collect artifact %s: %s", s.builder.Name, err)
		db.UpdateBuilderStatus(s.builder.ID, consts.BuildStatusFailure)
		return "", consts.BuildStatusFailure
	}
	db.UpdateBuilderStatus(s.builder.ID, status)
	if s.config.Type == consts.CommandBuildBeacon {
		if s.config.ArtifactId != 0 {
			err = db.UpdatePulseRelink(s.config.ArtifactId, s.builder.ID)
			if err != nil {
				logs.Log.Errorf("failed to update pulse relink: %s", err)
			}
		}
	}
	return path, status
}

func (s *SaasBuilder) getToken() string {
	saasConfig := configs.GetSaasConfig()
	return saasConfig.Token
}
