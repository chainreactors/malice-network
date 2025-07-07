package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"io"
	"net/http"
	"time"
)

func (rpc *Server) GetLicenseInfo(ctx context.Context, req *clientpb.Empty) (*clientpb.LicenseInfo, error) {
	saasConfig := configs.GetSaasConfig()

	if saasConfig.Token == "" {
		return nil, errs.ErrLicenseTokenNotFound
	}

	licenseInfo, err := getLicenseFromSaas(saasConfig)
	if err != nil {
		return nil, errs.ErrSaasUnable
	}

	return licenseInfo, nil
}

// LicenseResponse SaaS API响应结构
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

func getLicenseFromSaas(saasConfig *configs.SaasConfig) (*clientpb.LicenseInfo, error) {
	// 构建API URL - 这里需要根据实际的API路径调整
	// 假设API路径是 /api/license/info 或者使用token查询
	licenseUrl := fmt.Sprintf("%s/api/license/info", saasConfig.Url)

	// 创建HTTP请求
	req, err := http.NewRequest("GET", licenseUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// 添加认证头
	req.Header.Set("token", saasConfig.Token)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 解析JSON响应
	var response LicenseResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// 检查API响应是否成功
	if !response.Success {
		return nil, fmt.Errorf("API request failed: %s", string(body))
	}

	// 转换为protobuf格式
	licenseInfo := &clientpb.LicenseInfo{
		UserName:   response.License.Username,
		Type:       response.License.Type,
		ExpireAt:   response.License.ExpireAt,
		BuildCount: response.License.BuildCount,
		MaxBuilds:  response.License.MaxBuilds,
	}

	return licenseInfo, nil
}
