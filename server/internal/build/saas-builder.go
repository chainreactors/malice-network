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
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
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
	builder     *models.Builder
	executeUrl  string
	downloadUrl string
}

func NewSaasBuilder(req *clientpb.BuildConfig) *SaasBuilder {
	return &SaasBuilder{
		config: req,
	}
}

func (s *SaasBuilder) GenerateConfig() (*clientpb.Builder, error) {
	var builder *models.Builder
	var err error
	if s.config.ArtifactId != 0 && s.config.Type == consts.CommandBuildBeacon {
		builder, err = db.SaveArtifactFromID(s.config, s.config.ArtifactId, s.config.Resource)
	} else {
		if s.config.BuildName == "" {
			s.config.BuildName = codenames.GetCodename()
		}
		builder, err = db.SaveArtifactFromConfig(s.config)
	}
	if err != nil {
		logs.Log.Errorf("save build db error: %v", err)
		return nil, err
	}
	s.builder = builder
	db.UpdateBuilderStatus(s.builder.ID, consts.BuildStatusWaiting)
	saasConfig := configs.GetSaasConfig()
	s.executeUrl = fmt.Sprintf("http://%v:%v/api/build", saasConfig.Host, saasConfig.Port)
	return builder.ToProtobuf(), nil
}

func (s *SaasBuilder) ExecuteBuild() error {
	profileByte, err := GenerateProfile(s.config)
	if err != nil {
		sendSaasCtrlMsg(true, s.config, err)
		return err
	}
	base64Encoded := encode.Base64Encode(profileByte)
	s.config.Inputs = make(map[string]string)
	s.config.Inputs["malefic_config_yaml"] = base64Encoded
	data, err := protojson.Marshal(s.config)
	if err != nil {
		sendSaasCtrlMsg(true, s.config, fmt.Errorf("marshal build config failed: %w", err))
		return fmt.Errorf("marshal build config failed: %w", err)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("POST", s.executeUrl, bytes.NewReader(data))
	if err != nil {
		sendSaasCtrlMsg(true, s.config, fmt.Errorf("create build request failed: %w", err))
		return fmt.Errorf("create build request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	token := s.getToken()
	if token != "" {
		req.Header.Set("token", token)
	}
	resp, err := client.Do(req)
	if err != nil {
		sendSaasCtrlMsg(true, s.config, fmt.Errorf("post to build service failed: %w", err))
		return fmt.Errorf("post to build service failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		sendSaasCtrlMsg(true, s.config, fmt.Errorf("read build service response failed: %w", err))
		return fmt.Errorf("read build service response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		sendSaasCtrlMsg(true, s.config, errors.New(string(body)))
		return errors.New(string(body))
	}

	var builder clientpb.Builder
	if err := json.Unmarshal(body, &builder); err != nil {
		sendSaasCtrlMsg(true, s.config, fmt.Errorf("unmarshal build service response failed: %w", err))
		return fmt.Errorf("unmarshal build service response failed: %w", err)
	}

	logs.Log.Infof("saas build create success，builder: %v", builder.Name)
	return nil
}

func (s *SaasBuilder) CollectArtifact() {
	// 轮询间隔
	pollInterval := 30 * time.Second
	// 最大轮询时间（比如30分钟）
	maxPollTime := 30 * time.Minute
	startTime := time.Now()

	saasConfig := configs.GetSaasConfig()
	statusUrl := fmt.Sprintf("http://%v:%v/api/build/status/%v", saasConfig.Host, saasConfig.Port, s.builder.Name)
	downloadUrl := fmt.Sprintf("http://%v:%v/api/build/download/%v", saasConfig.Host, saasConfig.Port, s.builder.Name)

	for {
		if time.Since(startTime) > maxPollTime {
			logs.Log.Errorf("build %s polling timeout", s.builder.Name)
			db.UpdateBuilderStatus(s.builder.ID, consts.BuildStatusFailure)
			return
		}
		status, err := s.checkBuildStatus(statusUrl)
		if err != nil {
			logs.Log.Errorf("check build status failed: %v", err)
			time.Sleep(pollInterval)
			continue
		}

		if status == consts.BuildStatusFailure {
			logs.Log.Errorf("build %s failed", s.builder.Name)
			db.UpdateBuilderStatus(s.builder.ID, consts.BuildStatusFailure)
			sendSaasCtrlMsg(true, s.config, nil)
			return
		}

		if status == consts.BuildStatusCompleted {
			err := s.downloadArtifact(downloadUrl)
			if err != nil {
				logs.Log.Errorf("download artifact failed: %v", err)
				time.Sleep(pollInterval)
				continue
			}
			logs.Log.Infof("build %s completed and downloaded successfully", s.builder.Name)
			sendSaasCtrlMsg(true, s.config, nil)
			db.UpdateBuilderStatus(s.builder.ID, consts.BuildStatusCompleted)
			return
		}

		logs.Log.Debugf("build %s status: %s, continue polling...", s.builder.Name, status)
		time.Sleep(pollInterval)
	}
}

// 检查构建状态
func (s *SaasBuilder) checkBuildStatus(statusUrl string) (string, error) {
	token := s.getToken()
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

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status check failed with code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.Status, nil
}

// 下载构建产物
func (s *SaasBuilder) downloadArtifact(downloadUrl string) error {
	token := s.getToken()
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

	s.builder.Path = outputPath
	err = db.UpdateBuilderPath(s.builder)
	if err != nil {
		return err
	}

	target, ok := consts.GetBuildTarget(s.config.Target)
	if !ok {
		return errs.ErrInvalidateTarget
	}
	if s.builder.Type == consts.CommandBuildPulse {
		logs.Log.Infof("objcopy start ...")
		_, err = OBJCOPYPulse(s.builder, target.OS, target.Arch)
		if err != nil {
			return fmt.Errorf("objcopy error: %v", err)
		}
		logs.Log.Infof("objcopy end ...")
	} else {
		_, err = SRDIArtifact(s.builder, target.OS, target.Arch)
		if err != nil {
			return fmt.Errorf("SRDI error %v", err)
		}
	}
	return nil
}

func (s *SaasBuilder) getToken() string {
	saasConfig := configs.GetSaasConfig()
	return saasConfig.Token
}

func sendSaasCtrlMsg(isEnd bool, req *clientpb.BuildConfig, err error) {
	if core.EventBroker == nil {
		return
	}
	if err != nil {
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventBuild,
			IsNotify:  false,
			Message:   fmt.Sprintf("%s type(%s) by saas has a err %v. ", req.BuildName, req.Type, err),
			Important: true,
		})
	}
	if isEnd {
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventBuild,
			IsNotify:  false,
			Message:   fmt.Sprintf("%s type(%s) by saas  has completed. run `artifact list` to get the artifact.", req.BuildName, req.Type),
			Important: true,
		})
	} else {
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventBuild,
			IsNotify:  false,
			Message:   fmt.Sprintf("%s type(%s) by saas has started )...", req.BuildName, req.Type),
			Important: true,
		})
	}
}
