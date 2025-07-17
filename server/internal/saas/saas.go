package saas

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils"
	"github.com/chainreactors/malice-network/helper/utils/httputils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/chainreactors/tui"
	"io"
	"os"
	"strings"
	"time"
)

// ================= 工具类型和常量 =================

type DownloadResult struct {
	Path   string
	Status string
	Err    error
}

var ErrPollingTimeout = &PollingTimeoutError{}

type PollingTimeoutError struct{}

func (e *PollingTimeoutError) Error() string {
	return "polling timeout"
}

// ================= 工具函数 =================

// 统一SaaS请求头
func SaasHeaders(token string) map[string]string {
	return map[string]string{
		"token": token,
	}
}

// pollUntil会每隔interval调用fn，直到fn返回true或超时timeout
func pollUntil(fn func() (bool, error), interval, timeout time.Duration) error {
	start := time.Now()
	for {
		ok, err := fn()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		if time.Since(start) > timeout {
			return ErrPollingTimeout
		}
		time.Sleep(interval)
	}
}

// ================= SaasClient结构体及方法 =================

type SaasClient struct {
	Token   string
	BaseURL string
}

func NewSaasClient() *SaasClient {
	saasConfig := configs.GetSaasConfig()
	if !saasConfig.Enable {
		return &SaasClient{}
	}
	return &SaasClient{
		Token:   saasConfig.Token,
		BaseURL: saasConfig.Url,
	}
}

// 查询构建状态
func (c *SaasClient) CheckBuildStatus(statusPath string) (string, error) {
	if c.Token == "" {
		return "", fmt.Errorf("no token available for status check")
	}
	headers := SaasHeaders(c.Token)
	url := strings.TrimRight(c.BaseURL, "/") + statusPath
	var result struct {
		Success bool   `json:"success"`
		Status  string `json:"status"`
		Name    string `json:"name"`
		ID      string `json:"id"`
	}
	err := httputils.DoJSONRequest("GET", url, nil, headers, 200, &result)
	if err != nil {
		return "", err
	}
	return result.Status, nil
}

// 下载构建产物
func (c *SaasClient) DownloadArtifact(downloadPath string, builder *models.Artifact) error {
	if c.Token == "" {
		return fmt.Errorf("no token available for download")
	}
	headers := SaasHeaders(c.Token)
	url := strings.TrimRight(c.BaseURL, "/") + downloadPath
	resp, err := httputils.DoRequest("GET", url, nil, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed with code: %d", resp.StatusCode)
	}

	outputPath := fmt.Sprintf("%s/%s", configs.TempPath, encoders.UUID())
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	builder.Path = outputPath
	return db.UpdateBuilderPath(builder)
}

// 轮询并下载产物
func (c *SaasClient) CheckAndDownloadArtifact(statusPath, downloadPath string, builder *models.Artifact, pollInterval, maxPollTime time.Duration) DownloadResult {
	var status string
	pollErr := pollUntil(func() (bool, error) {
		var err error
		status, err = c.CheckBuildStatus(statusPath)
		if err != nil {
			logs.Log.Errorf("check build status failed: %v", err)
			return false, nil // 继续轮询
		}
		if status == consts.BuildStatusFailure {
			return false, fmt.Errorf("failed to build %s by saas", builder.Name)
		}
		return status == consts.BuildStatusCompleted, nil
	}, pollInterval, maxPollTime)
	if pollErr != nil {
		return DownloadResult{"", consts.BuildStatusFailure, pollErr}
	}
	if err := c.DownloadArtifact(downloadPath, builder); err != nil {
		logs.Log.Errorf("download artifact failed: %s", err)
		return DownloadResult{"", consts.BuildStatusFailure, err}
	}
	return DownloadResult{builder.Path, consts.BuildStatusCompleted, nil}
}

// LicenseResponse SaaS API响应结构
// 该结构体比 configs.LicenseResponse 更完整，适用于 License 查询
// 如有需要可迁移到 configs 包
// 这里只在 saas.go 内部使用
type LicenseResponse struct {
	Success bool `json:"success"`
	License struct {
		ID              string `json:"id"`
		Username        string `json:"username"`
		Email           string `json:"email"`
		Token           string `json:"token"`
		Type            string `json:"type"`
		ExpireAt        string `json:"expire_at"`
		BuildCount      int64  `json:"build_count"`
		MaxBuilds       int64  `json:"max_builds"`
		CreatedAt       string `json:"created_at"`
		UpdatedAt       string `json:"updated_at"`
		IsExpired       bool   `json:"is_expired"`
		CanBuild        bool   `json:"can_build"`
		RemainingBuilds int64  `json:"remaining_builds"`
	} `json:"license"`
}

// 获取 License 信息，返回 protobuf LicenseInfo
func (c *SaasClient) GetLicenseInfo() (*clientpb.LicenseInfo, string, error) {
	if c.Token == "" || c.BaseURL == "" {
		return nil, "", fmt.Errorf("invalid SaaS config")
	}
	licenseUrl := fmt.Sprintf("%s/api/license/info", c.BaseURL)
	headers := SaasHeaders(c.Token)
	headers["username"] = fmt.Sprintf("machine_%s", utils.GetMachineID())
	var response LicenseResponse
	err := httputils.DoJSONRequest("GET", licenseUrl, nil, headers, 200, &response)
	if err != nil {
		return nil, "", fmt.Errorf("failed to send HTTP request: %v", err)
	}
	if !response.Success {
		return nil, "", fmt.Errorf("API request failed: %+v", response)
	}
	return &clientpb.LicenseInfo{
		UserName:   response.License.Username,
		Type:       response.License.Type,
		ExpireAt:   response.License.ExpireAt,
		BuildCount: response.License.BuildCount,
		MaxBuilds:  response.License.MaxBuilds,
	}, response.License.Token, nil
}

// ================= 对外暴露的主流程函数 =================

// 重新下发SaaS构建任务
func ReDownloadSaasArtifact() error {
	client := NewSaasClient()
	if client.Token == "" || client.BaseURL == "" {
		return errs.ErrSaasUnable
	}
	artifacts, err := db.GetArtifactWithSaas()
	if err != nil {
		return err
	}
	if len(artifacts) == 0 {
		return nil
	}
	for _, artifact := range artifacts {
		if artifact.Status == consts.BuildStatusCompleted || artifact.Status == consts.BuildStatusFailure {
			continue
		}
		go func(art *models.Artifact) {
			statusPath := "/api/build/status/" + art.Name
			downloadPath := "/api/build/download/" + art.Name
			result := client.CheckAndDownloadArtifact(statusPath, downloadPath, art, 30*time.Second, 30*time.Minute)
			if result.Err != nil {
				logs.Log.Errorf("ReDownloadSaasArtifact: artifact %s failed: %v", art.Name, result.Err)
			}
			if result.Status == consts.BuildStatusCompleted || result.Status == consts.BuildStatusFailure {
				db.UpdateBuilderStatus(art.ID, result.Status)
			}
		}(artifact)
	}
	return nil
}

// 注册License
func RegisterLicense() error {
	// 1. 获取SaaS配置
	saasConfig := configs.GetSaasConfig()
	if saasConfig == nil {
		return fmt.Errorf("failed to get SaaS config")
	}
	// 2. 已有token或未启用SaaS则无需注册
	if !saasConfig.Enable {
		return nil
	} else {
		SecurityAuthAlert()
	}

	if saasConfig.Token != "" {
		client := NewSaasClient()
		_, token, err := client.GetLicenseInfo()
		if err != nil {
			return err
		}
		saasConfig.Token = token
		if err := configs.UpdateSaasConfig(saasConfig); err != nil {
			return fmt.Errorf("failed to update SaaS config: %v", err)
		}
		return ReDownloadSaasArtifact()
	}

	// 3. 构建注册数据
	licenseData, err := buildLicenseData()
	if err != nil {
		return fmt.Errorf("failed to build license data: %v", err)
	}

	// 4. 注册license，获取token
	token, err := sendLicenseRegistration(saasConfig.Url, licenseData)
	if err != nil {
		return fmt.Errorf("failed to register license: %v", err)
	}
	// 5. 保存token到配置
	saasConfig.Token = token
	if err := configs.UpdateSaasConfig(saasConfig); err != nil {
		return fmt.Errorf("failed to update SaaS config: %v", err)
	}
	// 6. 打印注册成功日志
	logs.Log.Infof("register saas success: %s", token)
	// 7. 重新下发SaaS构建任务
	return ReDownloadSaasArtifact()
}

// ================= 辅助/内部函数 =================

func buildLicenseData() (*configs.LicenseRegistrationData, error) {
	machineID := utils.GetMachineID()
	if machineID == "" {
		return nil, fmt.Errorf("failed to get machine ID")
	}
	return &configs.LicenseRegistrationData{
		Username:   fmt.Sprintf("machine_%s", machineID),
		Email:      "community@example.com",
		Type:       "community",
		MaxBuilds:  0,
		BuildCount: 0,
	}, nil
}

func sendLicenseRegistration(baseURL string, licenseData *configs.LicenseRegistrationData) (string, error) {
	headers := map[string]string{}
	var response configs.LicenseResponse
	err := httputils.DoPOST(baseURL+"/api/license/", licenseData, headers, 200, &response)
	if err != nil {
		return "", fmt.Errorf("failed to send HTTP request: %v", err)
	}
	if !response.Success {
		return "", fmt.Errorf("license registration failed: %+v", response)
	}
	if response.License.Token == "" {
		return "", fmt.Errorf("no token returned in response")
	}
	return response.License.Token, nil
}

// 对外导出：兼容外部包调用
func CheckAndDownloadArtifact(statusPath, downloadPath, token string, builder *models.Artifact, pollInterval, maxPollTime time.Duration) (string, string, error) {
	client := NewSaasClient()
	if client.Token == "" || client.BaseURL == "" {
		return "", "", errs.ErrSaasUnable
	}
	client.Token = token
	result := client.CheckAndDownloadArtifact(statusPath, downloadPath, builder, pollInterval, maxPollTime)
	return result.Path, result.Status, result.Err
}

func SecurityAuthAlert() {
	logs.Log.Info(tui.RedFg.Render("使用SaaS服务即视为您已阅读并同意我们的用户协议。详细协议内容请访问：https://wiki.chainreactors.red/IoM/#4"))
	logs.Log.Info(tui.RedFg.Render("By using the SaaS service, you are deemed to have read and agreed to our User Agreement. For detailed agreement content, please visit:, please visit: https://wiki.chainreactors.red/IoM/#4"))
}
