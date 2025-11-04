package db

import (
	"encoding/json"
	"errors"
	"fmt"
	consts "github.com/chainreactors/IoM-go/consts"
	clientpb "github.com/chainreactors/IoM-go/proto/client/clientpb"
	types2 "github.com/chainreactors/IoM-go/types"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"gorm.io/gorm"
)

// ============================================
// Profile Operations
// ============================================

// validateProfileName validates the profile name
func validateProfileName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}
	if len(name) > 100 {
		return fmt.Errorf("profile name too long (max 100 characters)")
	}
	return nil
}

// NewProfile creates a new profile
func NewProfile(profile *clientpb.Profile) error {
	// Validate input
	if err := validateProfileName(profile.Name); err != nil {
		return err
	}

	// Check if profile name already exists
	var existingProfile models.Profile
	result := Session().Where("name = ?", profile.Name).First(&existingProfile)
	if result.Error == nil {
		// Found existing profile with same name, return friendly error message
		return nil
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// If it's not "record not found" error, it's another database error
		return result.Error
	}

	// for pipeline
	if profile.Content == nil && profile.PipelineId != "" {
		pipelineModel, err := FindPipeline(profile.PipelineId)
		if err != nil {
			return fmt.Errorf("pipline not found, err: %s", err)
		}

		params, _ := types.UnmarshalProfileParams([]byte(profile.Params))
		if params != nil && params.REMPipeline != "" {
			remPipelineModel, err := FindPipeline(params.REMPipeline)
			if err != nil {
				return fmt.Errorf("pipline not found, err: %s", err)
			}
			profileConfig := remPipelineModel.DefaultRemProfile(pipelineModel)
			profile.Content, err = profileConfig.ToYAML()
			if err != nil {
				return err
			}
		} else {
			profileConfig, err := pipelineModel.ToProfile(nil)
			if err != nil {
				return err
			}
			profile.Content, err = profileConfig.ToYAML()
			if err != nil {
				return err
			}
		}

	}
	if profile.Content == nil {
		profile.Content = consts.DefaultProfile
	}
	// if not for pipeline
	contentType := fileutils.DetectContentType(profile.Content)
	if contentType == "zip" {
		profilePath := filepath.Join(configs.ProfilePath, profile.Name)
		err := os.MkdirAll(profilePath, 0700)
		if err != nil {
			return err
		}
		err = fileutils.DecompressBase64ToFiles(string(profile.Content), profilePath)
		if err != nil {
			return fmt.Errorf("failed to decompress zip content: %w", err)
		}

		configPath := filepath.Join(profilePath, "implant.yaml")
		if !fileutils.Exist(configPath) {
			return fmt.Errorf("implant.yaml not found in zip content")
		}

		yamlContent, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read implant.yaml: %w", err)
		}

		config, err := types.LoadProfile(yamlContent)
		if err != nil {
			return fmt.Errorf("failed to parse yaml config: %w", err)
		}

		if err := config.ValidateProfileFiles(profilePath); err != nil {
			return fmt.Errorf("profile validation failed: %w", err)
		}

		profile.Content = yamlContent
	}

	model := &models.Profile{
		Name:       profile.Name,
		ParamsData: profile.Params,
		PipelineID: profile.PipelineId,
		Raw:        profile.Content,
	}

	return Session().Create(model).Error
}

// GetProfile recovers profile from database
func GetProfile(name string) (*types.ProfileConfig, error) {
	var profileModel *models.Profile

	result := Session().Preload("Pipeline").Where("name = ?", name).First(&profileModel)
	if result.Error != nil {
		return nil, result.Error
	}
	if profileModel.PipelineID != "" && profileModel.Pipeline == nil {
		return nil, types2.ErrNotFoundPipeline
	}
	//if profileModel.PulsePipelineID != "" && profileModel.PulsePipeline == nil {
	//	return nil, errs.ErrNotFoundPipeline
	//}
	err := profileModel.DeserializeImplantConfig()
	if err != nil {
		return nil, err
	}
	profile, err := types.LoadProfile(profileModel.Raw)
	if err != nil {
		return nil, err
	}
	if profileModel.Name != "" {
		profile.Basic.Name = profileModel.Name
	}

	if profileModel.Pipeline != nil {
		// 为了向后兼容，创建一个简单的目标
		target := types.Target{
			Address: profileModel.Pipeline.Address(),
		}

		// 如果是 TLS 管道，设置 TLS 配置
		if profileModel.Pipeline.Tls.Enable {
			target.TLS = &types.TLSProfile{
				Enable: true,
			}
		}

		profile.Basic.Targets = []types.Target{target}
		profile.Basic.Encryption = profileModel.Pipeline.Encryption.Choice().Type
		profile.Basic.Key = profileModel.Pipeline.Encryption.Choice().Key
		// profile.Basic.Protocol = profileModel.Pipeline.Type
		// profile.Basic.TLS.Enable = profileModel.Pipeline.Tls.Enable

		if profileModel.Pipeline.Secure != nil && profileModel.Pipeline.Secure.Enable {
			// Profile中需要保存implant编译时需要的密钥：
			// - server公钥：implant用来加密发给server的数据
			// - implant私钥：implant用来解密server发来的数据

			profile.Basic.Secure = &types.SecureProfile{
				Enable:            true,
				ServerPublicKey:   profileModel.Pipeline.Secure.ServerPublicKey,
				ImplantPrivateKey: profileModel.Pipeline.Secure.ImplantPrivateKey,
			}
		}
		// 注意：protocol 字段已移除，现在通过 targets 中的具体配置来确定协议
	}
	if params := profileModel.Params; params != nil {
		profile.Basic.Cron = profileModel.Params.Cron
		profile.Basic.Jitter = profileModel.Params.Jitter
		if params.REMPipeline != "" {
			// 对于 REM 协议，我们需要添加到 targets 中
			pipeline, err := FindPipeline(params.REMPipeline)
			if err != nil {
				return nil, err
			}
			// 添加 REM 目标到 targets 列表
			profile.Basic.Targets = append(profile.Basic.Targets, types.Target{
				Address: pipeline.Address(),
				REM: &types.REMProfile{
					Link: pipeline.PipelineParams.Link,
				},
			})
		}

	}
	if profile.Pulse != nil && profileModel.Pipeline != nil {
		profile.Pulse.Target = profileModel.Pipeline.Address()
		profile.Pulse.Protocol = profileModel.Pipeline.Type
	}

	return profile, nil
}

// GetProfileContent GetProfile recovers profile from database
func GetProfileContent(profileName string) ([]byte, error) {
	var profileModel *models.Profile

	result := Session().Preload("Pipeline").Where("name = ?", profileName).First(&profileModel)
	if result.Error != nil {
		return nil, result.Error
	}
	//if profileModel.PipelineID != "" && profileModel.Pipeline == nil {
	//	return nil, errs.ErrNotFoundPipeline
	//}
	//if profileModel.PulsePipelineID != "" && profileModel.PulsePipeline == nil {
	//	return nil, errs.ErrNotFoundPipeline
	//}
	//err := profileModel.DeserializeImplantConfig()
	//if err != nil {
	//	return nil, err
	//}
	// profile, err := types.LoadProfile(profileModel.Raw)

	return profileModel.Raw, nil
}

func GetProfiles() ([]*models.Profile, error) {
	var profiles []*models.Profile
	result := Session().Preload("Pipeline").Order("created_at ASC").Find(&profiles)
	return profiles, result.Error
}

func GetProfileByName(profileName string) (*models.Profile, error) {
	var profile *models.Profile
	result := Session().Preload("Pipeline").Where("name = ?", profileName).Order("created_at ASC").First(&profile)
	return profile, result.Error
}

// FindBuildersByPipelineID 遍历所有 builder，找到 profile.pipelineID = pipelineID 的 builder
func FindBuildersByPipelineID(pipelineID string) ([]*models.Artifact, error) {
	var builders []*models.Artifact
	err := Session().Preload("Profile").Find(&builders).Error
	if err != nil {
		return nil, err
	}

	var validBuilders []*models.Artifact
	for _, b := range builders {
		if b.Profile.PipelineID == pipelineID {
			validBuilders = append(validBuilders, b)
		}
	}
	return validBuilders, nil
}

func DeleteProfileByName(profileName string) error {
	// Check if profile exists first
	var existingProfile models.Profile
	result := Session().Where("name = ?", profileName).First(&existingProfile)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return fmt.Errorf("profile '%s' not found", profileName)
	} else if result.Error != nil {
		return result.Error
	}

	// Execute deletion
	err := Session().Where("name = ?", profileName).Delete(&models.Profile{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete profile '%s': %v", profileName, err)
	}
	err = fileutils.ForceRemoveAll(filepath.Join(configs.ProfilePath, profileName))
	if err != nil {
		return err
	}
	return nil
}

func UpdateProfileRaw(profileName string, raw []byte) error {
	return Session().Model(&models.Profile{}).Where("name = ?", profileName).Update("raw", raw).Error
}

// ============================================
// Artifact Operations
// ============================================

func SaveArtifactFromConfig(req *clientpb.BuildConfig) (*models.Artifact, error) {
	target, ok := consts.GetBuildTarget(req.Target)
	if !ok {
		return nil, types2.ErrInvalidateTarget
	}
	builder := models.Artifact{
		Name:        req.BuildName,
		ProfileName: req.ProfileName,
		Target:      req.Target,
		Type:        req.BuildType,
		Source:      req.Source,
		Arch:        target.Arch,
		Os:          target.OS,
		ProfileByte: req.MaleficConfig,
		//ParamsData:  string(req.ParamsBytes),
	}

	if Session() == nil {
		return &builder, nil
	}
	if err := Session().Create(&builder).Error; err != nil {
		return nil, err
	}

	return &builder, nil
}

func SaveArtifactFromID(req *clientpb.BuildConfig, ID uint32) (*models.Artifact, error) {
	target, ok := consts.GetBuildTarget(req.Target)
	if !ok {
		return nil, types2.ErrInvalidateTarget
	}
	artifact := models.Artifact{
		ID:          ID,
		Name:        req.BuildName,
		ProfileName: req.ProfileName,
		Target:      req.Target,
		Type:        req.BuildType,
		Source:      req.Source,
		Arch:        target.Arch,
		Os:          target.OS,
		ProfileByte: req.MaleficConfig,
		//ParamsData:  string(req.ParamsBytes),
	}

	if err := Session().Create(&artifact).Error; err != nil {
		return nil, err

	}

	return &artifact, nil
}

func UpdateBuilderPath(builder *models.Artifact) error {
	if Session() == nil {
		return nil
	}
	return Session().Model(builder).
		Select("path").
		Updates(builder).
		Error
}

func UpdatePulseRelink(pusleID, beanconID uint32) error {
	// todo
	//pulse, err := GetArtifactById(pusleID)
	//if err != nil {
	//	return err
	//}
	//pulse.Params.RelinkBeaconID = beanconID
	//err = Session().Model(pulse).
	//	Select("ParamsData").
	//	Updates(pulse).
	//	Error
	//if err != nil {
	//	return err
	//}
	//originBeacon, err := GetArtifactById(pulse.Params.OriginBeaconID)
	//if err != nil {
	//	return err
	//}
	//originBeacon.Params.RelinkBeaconID = beanconID
	//err = Session().Model(originBeacon).
	//	Select("ParamsData").
	//	Updates(originBeacon).
	//	Error
	//if err != nil {
	//	return err
	//}
	return nil
}

func SaveArtifact(name, artifactType, platform, arch, source string) (*models.Artifact, error) {
	absBuildOutputPath, err := filepath.Abs(configs.TempPath)
	if err != nil {
		return nil, err
	}

	artifact := &models.Artifact{
		Name:   name,
		Os:     platform,
		Arch:   arch,
		Type:   artifactType,
		Source: source,
	}

	artifact.Path = filepath.Join(absBuildOutputPath, encoders.UUID())

	if err := Session().Create(artifact).Error; err != nil {
		return nil, err
	}
	return artifact, nil
}

func GetValidArtifacts() ([]*models.Artifact, error) {
	var artifacts []*models.Artifact
	result := Session().Preload("Profile").Preload("Profile.Pipeline").Find(&artifacts)
	if result.Error != nil {
		return nil, result.Error
	}

	var validArtifacts []*models.Artifact
	for _, artifact := range artifacts {
		if artifact.Path != "" {
			if _, err := os.Stat(artifact.Path); err == nil {
				validArtifacts = append(validArtifacts, artifact)
			}
		}
	}
	return validArtifacts, nil
}

// FindArtifact
func FindArtifact(target *clientpb.Artifact, bin bool) (*clientpb.Artifact, error) {
	var artifact *models.Artifact
	var result *gorm.DB
	// 根据 ID 或名称查找构建器
	if target.Id != 0 {
		result = Session().Where("id = ? AND status = ?", target.Id, consts.BuildStatusCompleted).Last(&artifact)
	} else if target.Name != "" {
		result = Session().Where("name = ? AND status = ?", target.Name, consts.BuildStatusCompleted).Last(&artifact)
	} else if target.Profile != "" {
		result = Session().Where("profile_name = ? AND status = ?", target.Profile, consts.BuildStatusCompleted).Last(&artifact)
	} else {
		var builders []*models.Artifact
		result = Session().Where("os = ? AND arch = ? AND type = ? AND status = ?", target.Platform, target.Arch, target.Type, consts.BuildStatusCompleted).
			Preload("Profile.Pipeline").
			Find(&builders)
		for _, v := range builders {
			if v.Type == consts.ImplantPulse && v.Profile.PipelineID == target.Pipeline {
				artifact = v
				break
			}
			if v.Profile.PipelineID == target.Pipeline {
				artifact = v
				break
			}
		}
	}
	if result.Error != nil {
		return nil, fmt.Errorf("error finding artifact: %v, target: %+v", result.Error, target)
	}
	if artifact == nil {
		return nil, types2.ErrNotFoundArtifact
	}
	if bin {
		content, err := os.ReadFile(artifact.Path)
		if err != nil && artifact.Status == consts.BuildStatusFailure {
			return nil, fmt.Errorf("error reading file for artifact: %s, error: %v", artifact.Name, err)
		}
		return artifact.ToProtobuf(content), nil
	} else {
		return artifact.ToProtobuf([]byte{}), nil
	}

}

func FindArtifactFromPipeline(pipelineName string) (*models.Artifact, error) {
	var artifacts []*models.Artifact
	result := Session().Preload("Profile").Where(" type = ?", consts.CommandBuildBeacon).Find(&artifacts)
	if result.Error != nil {
		return nil, result.Error
	}
	for _, artifact := range artifacts {
		if artifact.Profile.PipelineID == pipelineName {
			return artifact, nil
		}
	}
	return nil, ErrRecordNotFound
}

func GetArtifact(req *clientpb.Artifact) (*models.Artifact, error) {
	if req.Id != 0 {
		return GetArtifactById(req.Id)
	} else if req.Name != "" {
		return GetArtifactByName(req.Name)
	} else {
		return nil, types2.ErrNotFoundArtifact
	}
}

func GetArtifactByName(name string) (*models.Artifact, error) {
	var artifact models.Artifact
	result := Session().Preload("Profile").Where("name = ?", name).First(&artifact)
	if result.Error != nil {
		return nil, result.Error
	}
	return &artifact, nil
}

func GetArtifactById(id uint32) (*models.Artifact, error) {
	var artifact models.Artifact
	result := Session().Preload("Profile").Where("id = ?", id).First(&artifact)
	if result.Error != nil {
		return nil, result.Error
	}
	return &artifact, nil
}

func GetArtifactWithSaas() ([]*models.Artifact, error) {
	var artifacts []*models.Artifact
	result := Session().Where("source = ?", consts.ArtifactFromSaas).Find(&artifacts)
	if result.Error != nil {
		return nil, result.Error
	}
	return artifacts, nil
}

// GetBeaconBuilderByRelinkID 查找 type=beacon 且 RelinkBeaconID=指定id 的 builder
func GetBeaconBuilderByRelinkID(relinkID uint32) ([]*models.Artifact, error) {
	var artifacts []*models.Artifact
	err := Session().Where("type = ?", "beacon").Find(&artifacts).Error
	if err != nil {
		return nil, err
	}

	var result []*models.Artifact
	for _, b := range artifacts {
		var params types.ProfileParams
		if b.ParamsData != "" {
			if err := json.Unmarshal([]byte(b.ParamsData), &params); err == nil {
				if params.RelinkBeaconID == relinkID {
					result = append(result, b)
				}
			}
		}
	}
	return result, nil
}

func DeleteArtifactByName(artifactName string) error {
	model := &models.Artifact{}
	err := Session().Where("name = ?", artifactName).First(&model).Error
	if err != nil {
		return err
	}
	if model.Path != "" {
		err = os.Remove(model.Path)
		if err != nil {
			return err
		}
	}
	err = Session().Delete(model).Error
	if err != nil {
		return err
	}
	return nil
}

func UpdateBuilderLog(name string, logEntry string) {
	if Session() == nil {
		return
	}
	err := Session().Model(&models.Artifact{}).
		Where("name = ?", name).
		Update("log", gorm.Expr("ifnull(log, '') || ?", logEntry)).
		Error

	if err != nil {
		logs.Log.Errorf("Error updating log for Artifact name %s: %v", name, err)
	}
}

func GetBuilderLogs(builderName string, limit int) (string, error) {
	var builder models.Artifact
	if err := Session().Where("name = ?", builderName).First(&builder).Error; err != nil {
		return "", err
	}

	split := strings.Split(builder.Log, "\n")

	if limit > 0 && len(split) > limit {
		split = split[len(split)-limit:]
	}
	result := strings.Join(split, "\n")

	return result, nil
}

func UpdateBuilderStatus(builderID uint32, status string) {
	if Session() == nil {
		return
	}
	err := Session().Model(&models.Artifact{}).
		Where("id = ?", builderID).
		Update("status", status).
		Error
	if err != nil {
		logs.Log.Errorf("Error updating log for Artifact id %d: %v", builderID, err)
	}
	return
}
