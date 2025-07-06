package saas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/utils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"io"
	"net/http"
	"os"
	"time"
)

func ReDownloadSaasArtifact() error {
	saasConfig := configs.GetSaasConfig()
	if saasConfig.Token != "" && saasConfig.Enable {
		artifacts, err := db.GetArtifactWithSaas()
		if err != nil {
			return err
		}
		if len(artifacts) > 0 {
			for _, artifact := range artifacts {
				if artifact.Status != consts.BuildStatusCompleted && artifact.Status != consts.BuildStatusFailure {
					go func() {
						statusUrl := fmt.Sprintf("%s/api/build/status/%s", saasConfig.Url, artifact.Name)
						downloadUrl := fmt.Sprintf("%s/api/build/download/%s", saasConfig.Url, artifact.Name)
						_, _, err = CheckAndDownloadArtifact(statusUrl, downloadUrl, saasConfig.Token, artifact, 30*time.Second, 30*time.Minute)
						if err != nil {
							return
						}
					}()
				}
			}
		}
	}
	return nil
}

// CheckAndDownloadArtifact 外部调用的检查状态并下载产物的组合函数
func CheckAndDownloadArtifact(statusUrl string, downloadUrl string, token string, builder *models.Artifact,
	pollInterval time.Duration, maxPollTime time.Duration) (string, string, error) {

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
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return "", fmt.Errorf("bad request (400): %s", string(body))
		case http.StatusUnauthorized:
			return "", fmt.Errorf("unauthorized (401): invalid token")
		case http.StatusForbidden:
			return "", fmt.Errorf("forbidden (403): license expired or no permission")
		//case http.StatusNotFound:
		//	return "", fmt.Errorf("not found (404): build not found")
		case http.StatusInternalServerError:
			return "", fmt.Errorf("internal server error (500): database query failed - %s", string(body))
		default:
			return "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
		}
	}

	logs.Log.Debugf("Status response body: %s", string(body))

	var result struct {
		Success bool   `json:"success"`
		Status  string `json:"status"`
		Name    string `json:"name"`
		ID      uint32 `json:"id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
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

func RegisterLicense() error {
	saasConfig := configs.GetSaasConfig()
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

	saasConfig.Token = token
	if err := configs.UpdateSaasConfig(saasConfig); err != nil {
		return fmt.Errorf("failed to update SaaS config: %v", err)
	}

	logs.Log.Infof("register saas success: %s", token)
	return ReDownloadSaasArtifact()
}

// buildLicenseData
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

// sendLicenseRegistration
func sendLicenseRegistration(baseURL string, licenseData *configs.LicenseRegistrationData) (string, error) {
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response configs.LicenseResponse
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
