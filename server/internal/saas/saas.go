package saas

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/chainreactors/tui"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/utils"
	"github.com/chainreactors/malice-network/helper/utils/httputils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

const (
	BuildStageSubmit   = "submit"
	BuildStageStatus   = "status"
	BuildStageDownload = "download"
)

const maxBuildResultReasonRunes = 2048

// BuildResult describes the terminal outcome of one SaaS build attempt.
type BuildResult struct {
	Path   string
	Status string
	Stage  string
	Err    error
}

var ErrPollingTimeout = &PollingTimeoutError{}

type PollingTimeoutError struct{}

func (e *PollingTimeoutError) Error() string {
	return "polling timeout"
}

// IsNetworkError reports transport failures where no usable SaaS response was received.
func IsNetworkError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, net.ErrClosed) {
		return true
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
		if errors.Is(err, context.Canceled) {
			return false
		}
		if errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(err, io.EOF) ||
			errors.Is(err, io.ErrUnexpectedEOF) ||
			errors.Is(err, net.ErrClosed) {
			return true
		}
	}

	var netErr net.Error
	return errors.As(err, &netErr)
}

func formatBuildResultLog(now time.Time, result BuildResult) string {
	timestamp := now.Format(time.RFC3339)
	switch result.Status {
	case consts.BuildStatusCompleted:
		return fmt.Sprintf("%s [COMPLETED] SaaS build completed\n", timestamp)
	case consts.BuildStatusNetworkError:
		return fmt.Sprintf("%s [NETWORK_ERROR] stage=%s reason=%s\n", timestamp, result.Stage, buildResultReason(result.Err))
	default:
		return fmt.Sprintf("%s [FAILED] stage=%s reason=%s\n", timestamp, result.Stage, buildResultReason(result.Err))
	}
}

func buildResultReason(err error) string {
	if err == nil {
		return "unknown error"
	}
	reason := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, err.Error())
	reason = strings.Join(strings.Fields(reason), " ")
	runes := []rune(reason)
	if len(runes) > maxBuildResultReasonRunes {
		reason = string(runes[:maxBuildResultReasonRunes])
	}
	return reason
}

// RecordBuildResult persists the terminal status and its single-line artifact log.
func RecordBuildResult(builder *models.Artifact, result BuildResult) error {
	if builder == nil {
		return fmt.Errorf("artifact is nil")
	}
	logEntry := formatBuildResultLog(time.Now(), result)
	if err := db.UpdateBuilderResult(builder.ID, result.Status, logEntry); err != nil {
		return err
	}
	builder.Status = result.Status
	return nil
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

type LicenseData struct {
	Username   string `json:"username"`
	Email      string `json:"email,omitempty"`
	Token      string `json:"token,omitempty"`
	Type       string `json:"type"`
	ExpireAt   string `json:"expire_at,omitempty"`
	BuildCount int64  `json:"build_count,omitempty"`
	MaxBuilds  int64  `json:"max_builds,omitempty"`
}

// LicenseResponse SaaS API 统一响应结构
type LicenseResponse struct {
	Success bool        `json:"success"`
	License LicenseData `json:"license"`
}

// 转换为 protobuf LicenseInfo
func (l *LicenseData) ToLicenseInfo() *clientpb.LicenseInfo {
	return &clientpb.LicenseInfo{
		UserName:   l.Username,
		Type:       l.Type,
		ExpireAt:   l.ExpireAt,
		BuildCount: l.BuildCount,
		MaxBuilds:  l.MaxBuilds,
	}
}

// 创建默认的社区版 License 数据
func NewCommunityLicense() *LicenseData {
	machineID := utils.GetMachineID()
	if machineID == "" {
		return nil
	}
	return &LicenseData{
		Username:   fmt.Sprintf("machine_%s", machineID),
		Email:      "community@example.com",
		Type:       "community",
		MaxBuilds:  0,
		BuildCount: 0,
	}
}

type SaasClient struct {
	Token       string
	BaseURL     string
	LicenseType string
}

func GetSaasClient() *SaasClient {
	saasConfig := configs.GetSaasConfig()
	if saasConfig == nil || !saasConfig.Enable {
		return &SaasClient{}
	}
	return &SaasClient{
		Token:   saasConfig.Token,
		BaseURL: saasConfig.Url,
	}
}

func (c *SaasClient) SetLicenseType(typ string) {
	c.LicenseType = typ
}

// 获取 LicenseType
func (c *SaasClient) GetLicenseType() string {
	return c.LicenseType
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
func (c *SaasClient) CheckAndDownloadArtifact(statusPath, downloadPath string, builder *models.Artifact, pollInterval, maxPollTime time.Duration) BuildResult {
	var lastNetworkErr error
	pollErr := pollUntil(func() (bool, error) {
		status, err := c.CheckBuildStatus(statusPath)
		if err != nil {
			logs.Log.Errorf("check build status failed: %v", err)
			if IsNetworkError(err) {
				lastNetworkErr = err
				return false, nil
			}
			return false, fmt.Errorf("check SaaS build status: %w", err)
		}

		// Clear the last transport error after a successful poll so a build
		// timeout is not misclassified as a network failure.
		lastNetworkErr = nil
		switch status {
		case consts.BuildStatusWaiting, consts.BuildStatusRunning:
			return false, nil
		case consts.BuildStatusCompleted:
			return true, nil
		case consts.BuildStatusFailure:
			return false, fmt.Errorf("SaaS service reported build failure for %s", builder.Name)
		default:
			return false, fmt.Errorf("unexpected SaaS build status %q", status)
		}
	}, pollInterval, maxPollTime)
	if pollErr != nil {
		status := consts.BuildStatusFailure
		if errors.Is(pollErr, ErrPollingTimeout) {
			if lastNetworkErr != nil {
				status = consts.BuildStatusNetworkError
				pollErr = fmt.Errorf("status polling timed out; last error: %w", lastNetworkErr)
			} else {
				pollErr = fmt.Errorf("SaaS build did not complete within %s", maxPollTime)
			}
		} else if IsNetworkError(pollErr) {
			status = consts.BuildStatusNetworkError
		}
		return BuildResult{Status: status, Stage: BuildStageStatus, Err: pollErr}
	}
	if err := c.DownloadArtifact(downloadPath, builder); err != nil {
		logs.Log.Errorf("download artifact failed: %s", err)
		status := consts.BuildStatusFailure
		if IsNetworkError(err) {
			status = consts.BuildStatusNetworkError
		}
		return BuildResult{Status: status, Stage: BuildStageDownload, Err: fmt.Errorf("download SaaS artifact: %w", err)}
	}
	return BuildResult{Path: builder.Path, Status: consts.BuildStatusCompleted}
}

// 获取 License 信息
func (c *SaasClient) GetLicenseInfo() (*clientpb.LicenseInfo, string, error) {
	return c.GetLicenseInfoContext(context.Background())
}

func (c *SaasClient) GetLicenseInfoContext(ctx context.Context) (*clientpb.LicenseInfo, string, error) {
	if c.Token == "" || c.BaseURL == "" {
		return nil, "", fmt.Errorf("invalid SaaS config")
	}

	licenseUrl := fmt.Sprintf("%s/api/license/info", c.BaseURL)
	headers := SaasHeaders(c.Token) // 只发送token

	var response LicenseResponse
	err := httputils.DoJSONRequestContext(ctx, "GET", licenseUrl, nil, headers, 200, &response)
	if err != nil {
		return nil, "", fmt.Errorf("failed to send HTTP request: %w", err)
	}

	if !response.Success {
		return nil, "", fmt.Errorf("API request failed: %+v", response)
	}

	c.SetLicenseType(response.License.Type)
	return response.License.ToLicenseInfo(), response.License.Token, nil
}

// 注册 License
func (c *SaasClient) RegisterLicense() (string, error) {
	return c.RegisterLicenseContext(context.Background())
}

func (c *SaasClient) RegisterLicenseContext(ctx context.Context) (string, error) {
	if c.BaseURL == "" {
		return "", fmt.Errorf("invalid SaaS config")
	}

	machineID := utils.GetMachineID()
	if machineID == "" {
		return "", fmt.Errorf("failed to get machine ID")
	}

	username := fmt.Sprintf("machine_%s", machineID)

	url := fmt.Sprintf("%s/api/license/register", c.BaseURL)
	payload := map[string]string{
		"username": username,
	}

	var response LicenseResponse
	err := httputils.DoPOSTContext(ctx, url, payload, map[string]string{}, 200, &response)
	if err != nil {
		return "", fmt.Errorf("failed to send HTTP request: %w", err)
	}

	if !response.Success {
		return "", fmt.Errorf("license registration failed: %+v", response)
	}

	if response.License.Token == "" {
		return "", fmt.Errorf("no token returned in response")
	}

	logs.Log.Infof("Successfully registered with token: %s", response.License.Token)
	return response.License.Token, nil
}

// ================= 对外暴露的主流程函数 =================

// 重新下发SaaS构建任务
func ReDownloadSaasArtifact() error {
	client := GetSaasClient()
	if client.Token == "" || client.BaseURL == "" {
		return types.ErrSaasUnable
	}
	artifacts, err := db.GetArtifactWithSaas()
	if err != nil {
		return err
	}
	if len(artifacts) == 0 {
		return nil
	}
	for _, artifact := range artifacts {
		if artifact.Status == consts.BuildStatusCompleted ||
			artifact.Status == consts.BuildStatusFailure ||
			artifact.Status == consts.BuildStatusNetworkError {
			continue
		}
		artifact := artifact
		core.GoGuarded("saas-redownload:"+artifact.Name, func() error {
			statusPath := "/api/build/status/" + artifact.Name
			downloadPath := "/api/build/download/" + artifact.Name
			result := client.CheckAndDownloadArtifact(statusPath, downloadPath, artifact, 30*time.Second, 30*time.Minute)
			if result.Err != nil {
				logs.Log.Errorf("ReDownloadSaasArtifact: artifact %s failed: %v", artifact.Name, result.Err)
			}
			if err := RecordBuildResult(artifact, result); err != nil {
				logs.Log.Errorf("ReDownloadSaasArtifact: failed to record artifact %s result: %v", artifact.Name, err)
			}
			return nil
		}, core.LogGuardedError("saas-redownload:"+artifact.Name))
	}
	return nil
}

// 注册License
func RegisterLicense() error {
	return RegisterLicenseContext(context.Background())
}

func RegisterLicenseContext(ctx context.Context) error {
	// 1. 获取SaaS配置
	saasConfig := configs.GetSaasConfig()
	if saasConfig == nil {
		return fmt.Errorf("failed to get SaaS config")
	}

	// 2. 未启用SaaS则无需注册
	if !saasConfig.Enable {
		return nil
	}

	SecurityAuthAlert()

	// 3. 已有token则验证并更新
	if saasConfig.Token != "" {
		client := GetSaasClient()
		_, token, err := client.GetLicenseInfoContext(ctx)
		if err != nil {
			return err
		}
		saasConfig.Token = token
		if err := configs.UpdateSaasConfig(saasConfig); err != nil {
			return fmt.Errorf("failed to update SaaS config: %v", err)
		}
		return ReDownloadSaasArtifact()
	}

	// 4. 注册新license
	client := GetSaasClient()
	token, err := client.RegisterLicenseContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to register license: %w", err)
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

// 对外导出：兼容外部包调用
func CheckAndDownloadArtifact(statusPath, downloadPath, token string, builder *models.Artifact, pollInterval, maxPollTime time.Duration) BuildResult {
	client := GetSaasClient()
	if client.BaseURL == "" {
		return BuildResult{Status: consts.BuildStatusFailure, Stage: BuildStageStatus, Err: types.ErrSaasUnable}
	}
	if token != "" {
		client.Token = token
	}
	if client.Token == "" {
		return BuildResult{Status: consts.BuildStatusFailure, Stage: BuildStageStatus, Err: types.ErrSaasUnable}
	}
	return client.CheckAndDownloadArtifact(statusPath, downloadPath, builder, pollInterval, maxPollTime)
}

func SecurityAuthAlert() {
	//logs.Log.Info(tui.RedFg.Render("使用本SaaS服务即表示您已阅读并同意《用户协议》。详情请访问：https://wiki.chainreactors.red/IoM/#4"))
	logs.Log.Info(tui.RedFg.Render("By using this SaaS service, you acknowledge that you have read and agreed to our User Agreement. For details, please visit: https://wiki.chainreactors.red/IoM/#4"))
}
