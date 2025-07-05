package build

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/chainreactors/utils/encode"
	"google.golang.org/protobuf/encoding/protojson"
	"io"
	"net/http"
	"os"
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

func (s *SaasBuilder) GenerateConfig() (*clientpb.Artifact, error) {
	var builder *models.Artifact
	var err error
	profileByte, err := GenerateProfile(s.config)
	if err != nil {
		return nil, err
	}
	base64Encoded := encode.Base64Encode(profileByte)
	s.config.Inputs = make(map[string]string)
	s.config.Inputs["malefic_config_yaml"] = base64Encoded
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
	saasConfig := configs.GetSaasConfig()
	s.executeUrl = fmt.Sprintf("%s/api/build", saasConfig.Url)
	return builder.ToArtifact([]byte{}), nil
}

func (s *SaasBuilder) ExecuteBuild() error {
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

func (s *SaasBuilder) CollectArtifact() (string, string) {
	saasConfig := configs.GetSaasConfig()
	statusUrl := fmt.Sprintf("%s/api/build/status/%s", saasConfig.Url, s.builder.Name)
	downloadUrl := fmt.Sprintf("%s/api/build/download/%s", saasConfig.Url, s.builder.Name)

	// 使用外部函数进行状态检查和下载
	path, status, err := CheckAndDownloadArtifact(statusUrl, downloadUrl, s.getToken(), s.builder, 30*time.Second, 30*time.Minute)
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
	SendBuildMsg(s.builder, consts.BuildStatusCompleted, "")
	return path, status
}

func (s *SaasBuilder) getToken() string {
	saasConfig := configs.GetSaasConfig()
	return saasConfig.Token
}

func sendSaasCtrlMsg(isEnd bool, req *models.Artifact, err error, status string) {
	if core.EventBroker == nil {
		return
	}
	if err != nil {
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventBuild,
			IsNotify:  false,
			Message:   fmt.Sprintf("%s type(%s) by saas has a err %v. ", req.Name, req.Type, err),
			Important: true,
		})
	}
	if isEnd {
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventBuild,
			IsNotify:  false,
			Message:   fmt.Sprintf("%s type(%s) by saas has %s.", req.Name, req.Type, status),
			Important: true,
		})
	} else {
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventBuild,
			IsNotify:  false,
			Message:   fmt.Sprintf("%s type(%s) by saas has started )...", req.Name, req.Type),
			Important: true,
		})
	}
}

// 外部可调用的函数

// CheckBuildStatusExternal 外部调用的构建状态检查函数
func CheckBuildStatusExternal(statusUrl string, token string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("no token available for status check")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", statusUrl, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("token", token)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%d", resp.StatusCode)
	}

	logs.Log.Debugf("Status response body: %s", string(body))

	var result struct {
		Success bool   `json:"success"`
		Status  string `json:"status"`
		Name    string `json:"name"`
		ID      uint32 `json:"id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		// 如果解析失败，尝试旧的格式
		var oldResult struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(body, &oldResult); err != nil {
			return "", fmt.Errorf("failed to parse status response: %v, body: %s", err, string(body))
		}
		return oldResult.Status, nil
	}

	return result.Status, nil
}

// DownloadArtifactWithBuilder 外部调用的构建产物下载函数（带Builder信息）
func DownloadArtifactWithBuilder(downloadUrl string, token string, builder *models.Artifact) error {
	if token == "" {
		return fmt.Errorf("no token available for download")
	}

	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("GET", downloadUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("token", token)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	outputPath := fmt.Sprintf("%s/%s", configs.BuildOutputPath, encoders.UUID())
	err = os.WriteFile(outputPath, body, 0644)
	if err != nil {
		return err
	}

	// 更新Builder的路径
	builder.Path = outputPath
	err = db.UpdateBuilderPath(builder)
	if err != nil {
		return err
	}
	return nil
}

// CheckAndDownloadArtifact 外部调用的检查状态并下载产物的组合函数
func CheckAndDownloadArtifact(statusUrl string, downloadUrl string, token string, builder *models.Artifact,
	pollInterval time.Duration, maxPollTime time.Duration) (string, string, error) {

	if pollInterval == 0 {
		pollInterval = 30 * time.Second
	}
	if maxPollTime == 0 {
		maxPollTime = 30 * time.Minute
	}

	startTime := time.Now()

	for {
		if time.Since(startTime) > maxPollTime {
			logs.Log.Errorf("build polling timeout")
			if builder != nil {
				db.UpdateBuilderStatus(builder.ID, consts.BuildStatusFailure)
			}
			return "", consts.BuildStatusFailure, fmt.Errorf("build polling timeout")
		}

		status, err := CheckBuildStatusExternal(statusUrl, token)
		if err != nil {
			logs.Log.Errorf("check build status failed: %v", err)
			time.Sleep(pollInterval)
			continue
		}

		if status == consts.BuildStatusFailure {
			if builder != nil {
				db.UpdateBuilderStatus(builder.ID, consts.BuildStatusFailure)
			}
			SendBuildMsg(builder, consts.BuildStatusFailure, "")
			return "", consts.BuildStatusFailure, fmt.Errorf("failed to build %s by saas", builder.Name)
		}

		if status == consts.BuildStatusCompleted {
			downloadErr := DownloadArtifactWithBuilder(downloadUrl, token, builder)

			if downloadErr != nil {
				logs.Log.Errorf("download artifact failed: %s", downloadErr)
				time.Sleep(pollInterval)
				continue
			}
			logs.Log.Infof("build completed and downloaded successfully")
			if builder != nil {
				db.UpdateBuilderStatus(builder.ID, consts.BuildStatusCompleted)
			}
			return builder.Path, consts.BuildStatusCompleted, nil
		}
		time.Sleep(pollInterval)
	}
}

func (s *SaasBuilder) GetBeaconID() uint32 {
	return s.config.ArtifactId
}

func (s *SaasBuilder) SetBeaconID(id uint32) error {
	s.config.ArtifactId = id
	if s.config.Params == "" {
		params := &types.ProfileParams{
			OriginBeaconID: id,
		}
		s.config.Params = params.String()
	} else {
		var newParams *types.ProfileParams
		err := json.Unmarshal([]byte(s.config.Params), &newParams)
		if err != nil {
			return err
		}
		newParams.OriginBeaconID = s.config.ArtifactId
		s.config.Params = newParams.String()
	}
	return nil
}
