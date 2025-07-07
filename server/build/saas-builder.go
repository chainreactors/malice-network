package build

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/chainreactors/malice-network/server/internal/saas"
	"github.com/chainreactors/utils/encode"
	"google.golang.org/protobuf/encoding/protojson"
	"io"
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
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("POST", s.executeUrl, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create build request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")
	token := s.getToken()
	if token != "" {
		req.Header.Set("token", token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post saas service failed: %s", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read build service response: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(string(body))
	}

	var builder clientpb.Artifact
	if err = json.Unmarshal(body, &builder); err != nil {
		return fmt.Errorf("failed to unmarshal build service response: %s", err)
	}
	return nil
}

func (s *SaasBuilder) Collect() (string, string) {
	saasConfig := configs.GetSaasConfig()
	statusUrl := fmt.Sprintf("%s/api/build/status/%s", saasConfig.Url, s.builder.Name)
	downloadUrl := fmt.Sprintf("%s/api/build/download/%s", saasConfig.Url, s.builder.Name)

	// 使用外部函数进行状态检查和下载
	path, status, err := saas.CheckAndDownloadArtifact(statusUrl, downloadUrl, s.getToken(), s.builder, 30*time.Second, 30*time.Minute)
	if err != nil {
		logs.Log.Errorf("failed to collect artifact %s: %s", s.builder.Name, err)
		return "", consts.BuildStatusFailure
	}
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

// 外部可调用的函数

//func (s *SaasBuilder) GetBeaconID() uint32 {
//	return s.config.ArtifactId
//}
//
//func (s *SaasBuilder) SetBeaconID(id uint32) error {
//	s.config.ArtifactId = id
//	if s.config.Params == "" {
//		params := &types.ProfileParams{
//			OriginBeaconID: id,
//		}
//		s.config.Params = params.String()
//	} else {
//		var newParams *types.ProfileParams
//		err := json.Unmarshal([]byte(s.config.Params), &newParams)
//		if err != nil {
//			return err
//		}
//		newParams.OriginBeaconID = s.config.ArtifactId
//		s.config.Params = newParams.String()
//	}
//	return nil
//}
