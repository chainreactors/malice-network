package saas

import (
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/chainreactors/utils/encode"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/build"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
)

// POST /api/build
func BuildHandler(c *gin.Context) {
	fmt.Println("收到构建请求")
	var req clientpb.BuildConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Println("参数解析失败：", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	artifact, err := BuildProcess(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 新增：从header获取token，查找license，写入history
	token := c.GetHeader("token")
	if token != "" {
		licenses, _ := db.ListLicenses()
		for _, l := range licenses {
			if l.Token == token {
				history := &models.History{
					LicenseID: l.ID,
					BuildName: artifact.Name,
				}
				err := db.CreateHistory(history)
				if err != nil {
					fmt.Println("写入history失败：", err)
				}
				break
			}
		}
	}

	c.JSON(http.StatusOK, artifact)
}

func BuildProcess(req *clientpb.BuildConfig) (*clientpb.Builder, error) {
	req.Resource = consts.ArtifactFromDocker
	var profile *types.ProfileConfig
	var err error
	if req.Inputs != nil {
		profileBase64 := req.Inputs["malefic_config_yaml"]
		profileByte := encode.Base64Decode(profileBase64)
		profile, err = types.LoadProfile(profileByte)
		if err != nil {
			return nil, err
		}
		err = build.WirteProfile(profile)
		if err != nil {
			return nil, err
		}
	}
	if req.Type == consts.CommandBuildPulse || req.Inputs["package"] == consts.CommandBuildPulse {
		var artifactID uint32
		if req.ArtifactId != 0 {
			artifactID = req.ArtifactId
		} else {
			yamlID := profile.Pulse.Extras["flags"].(map[string]interface{})["artifact_id"].(int)
			if uint32(yamlID) != 0 {
				artifactID = uint32(yamlID)
			} else {
				artifactID = 0
			}
		}
		_, err := db.GetArtifactById(artifactID)
		if err != nil && !errors.Is(err, db.ErrRecordNotFound) {
			return nil, err
		}
		if errors.Is(err, db.ErrRecordNotFound) {
			beaconReq := proto.Clone(req).(*clientpb.BuildConfig)
			beaconReq.Srdi = true
			if req.Resource == consts.ArtifactFromAction {
				beaconReq.Inputs["package"] = consts.CommandBuildBeacon
				if beaconReq.Inputs["targets"] == consts.TargetX86Windows {
					beaconReq.Inputs["targets"] = consts.TargetX86WindowsGnu
				} else {
					beaconReq.Inputs["targets"] = consts.TargetX64WindowsGnu
				}
			} else {
				beaconReq.Type = consts.CommandBuildBeacon
				if beaconReq.Target == consts.TargetX86Windows {
					beaconReq.Target = consts.TargetX86WindowsGnu
				} else {
					beaconReq.Target = consts.TargetX64WindowsGnu
				}
			}
			beaconBuilder := build.NewBuilder(beaconReq)
			artifact, err := beaconBuilder.GenerateConfig()
			if err != nil {
				return nil, err
			}
			req.ArtifactId = artifact.Id
			go func() {
				executeErr := beaconBuilder.ExecuteBuild()
				if executeErr == nil {
					beaconBuilder.CollectArtifact()
				}
			}()
		}
	}

	builder := build.NewBuilder(req)
	artifact, err := builder.GenerateConfig()
	if err != nil {
		return nil, err
	}
	go func() {
		executeErr := builder.ExecuteBuild()
		if executeErr == nil {
			builder.CollectArtifact()
		}
	}()
	return artifact, nil
}

// GET /api/build/download/:buildname
func DownloadBuildHandler(c *gin.Context) {
	buildName := c.Param("buildname")
	token := c.GetHeader("token")

	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	// 1. 通过token找到license
	licenses, err := db.ListLicenses()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get licenses"})
		return
	}
	var targetLicense *models.License
	for _, license := range licenses {
		if license.Token == token {
			targetLicense = license
			break
		}
	}
	if targetLicense == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	if time.Now().After(targetLicense.ExpireAt) {
		c.JSON(http.StatusForbidden, gin.H{"error": "license expired"})
		return
	}

	// 2. 用新方法查找History
	targetHistory, err := db.GetHistoryByBuildNameAndLicenseID(buildName, targetLicense.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "no permission to download this build"})
		return
	}
	builder := targetHistory.Build
	if builder.Path == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "build file not found"})
		return
	}

	// 4. 发送文件
	filename := filepath.Base(builder.Path)
	file, err := os.Open(builder.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "open file failed"})
		return
	}
	defer file.Close()
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Type", "application/octet-stream")
	c.File(builder.Path)
}

// GET /api/build/status/:buildname
func BuildStatusHandler(c *gin.Context) {
	buildName := c.Param("buildname")
	token := c.GetHeader("token")

	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	licenses, err := db.ListLicenses()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get licenses"})
		return
	}
	var targetLicense *models.License
	for _, license := range licenses {
		if license.Token == token {
			targetLicense = license
			break
		}
	}
	if targetLicense == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	if time.Now().After(targetLicense.ExpireAt) {
		c.JSON(http.StatusForbidden, gin.H{"error": "license expired"})
		return
	}

	// 用新方法查找History
	targetHistory, err := db.GetHistoryByBuildNameAndLicenseID(buildName, targetLicense.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "no permission to check this build status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": targetHistory.Build.Status,
		"name":   buildName,
		"id":     targetHistory.Build.ID,
	})
}
