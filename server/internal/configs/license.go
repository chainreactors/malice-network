package configs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/utils"
	"io"
	"net/http"
	"time"
)

// LicenseRegistrationData 许可证注册数据结构
type LicenseRegistrationData struct {
	Username   string `json:"username"`
	Email      string `json:"email"`
	Type       string `json:"type"`
	MaxBuilds  int    `json:"max_builds"`
	BuildCount int    `json:"build_count"`
}

// LicenseResponse SaaS API响应结构
type LicenseResponse struct {
	Success bool `json:"success"`
	License struct {
		Token string `json:"Token"`
	} `json:"license"`
}

// httpClient 全局HTTP客户端，避免重复创建
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func RegisterLicense() error {
	saasConfig := GetSaasConfig()
	if saasConfig == nil {
		return fmt.Errorf("failed to get SaaS config")
	}

	if saasConfig.Token != "" || !saasConfig.Enable {
		return nil
	}

	licenseData, err := buildLicenseData()
	if err != nil {
		return fmt.Errorf("failed to build license data: %v", err)
	}

	token, err := sendLicenseRegistration(saasConfig.Url, licenseData)
	if err != nil {
		return fmt.Errorf("failed to register license: %v", err)
	}

	if err := updateSaasConfig(saasConfig, token); err != nil {
		return fmt.Errorf("failed to update SaaS config: %v", err)
	}

	logs.Log.Infof("register saas success: %s", token)
	return nil
}

// buildLicenseData
func buildLicenseData() (*LicenseRegistrationData, error) {
	machineID := utils.GetMachineID()
	if machineID == "" {
		return nil, fmt.Errorf("failed to get machine ID")
	}

	return &LicenseRegistrationData{
		Username:   fmt.Sprintf("machine_%s", machineID),
		Email:      "community@example.com",
		Type:       "community",
		MaxBuilds:  0,
		BuildCount: 0,
	}, nil
}

// sendLicenseRegistration
func sendLicenseRegistration(baseURL string, licenseData *LicenseRegistrationData) (string, error) {
	licenseURL := fmt.Sprintf("%s/api/license/", baseURL)

	jsonData, err := json.Marshal(licenseData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal license data: %v", err)
	}
	req, err := http.NewRequest("POST", licenseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response LicenseResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if !response.Success {
		return "", fmt.Errorf("license registration failed: %s", string(body))
	}

	if response.License.Token == "" {
		return "", fmt.Errorf("no token returned in response")
	}

	return response.License.Token, nil
}

// updateSaasConfig 更新SaaS配置
func updateSaasConfig(saasConfig *SaasConfig, token string) error {
	saasConfig.Token = token
	return UpdateSaasConfig(saasConfig)
}
