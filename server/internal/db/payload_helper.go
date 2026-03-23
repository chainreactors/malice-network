package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	types "github.com/chainreactors/IoM-go/types"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/gofrs/uuid"
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

// readProfileDisk reads all profile configuration files from disk.
func readProfileDisk(profilePath string) (implantConfig []byte, preludeConfig []byte, resources *clientpb.BuildResources, err error) {
	implantPath := filepath.Join(profilePath, "implant.yaml")
	implantConfig, err = os.ReadFile(implantPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read implant.yaml: %w", err)
	}

	preludePath := filepath.Join(profilePath, "prelude.yaml")
	if fileutils.Exist(preludePath) {
		preludeConfig, _ = os.ReadFile(preludePath)
	}

	resourcesDir := filepath.Join(profilePath, "resources")
	if fileutils.Exist(resourcesDir) {
		entries, readErr := os.ReadDir(resourcesDir)
		if readErr == nil {
			var resourceEntries []*clientpb.ResourceEntry
			for _, e := range entries {
				if !e.IsDir() {
					content, fErr := os.ReadFile(filepath.Join(resourcesDir, e.Name()))
					if fErr == nil {
						resourceEntries = append(resourceEntries, &clientpb.ResourceEntry{
							Filename: e.Name(),
							Content:  content,
						})
					}
				}
			}
			if len(resourceEntries) > 0 {
				resources = &clientpb.BuildResources{Entries: resourceEntries}
			}
		}
	}
	return
}

// writeProfileDisk writes configuration files to the disk directory.
func writeProfileDisk(profilePath string, implantConfig []byte, preludeConfig []byte, resources *clientpb.BuildResources) error {
	if err := os.MkdirAll(profilePath, 0700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(profilePath, "implant.yaml"), implantConfig, 0644); err != nil {
		return err
	}
	if preludeConfig != nil {
		if err := os.WriteFile(filepath.Join(profilePath, "prelude.yaml"), preludeConfig, 0644); err != nil {
			return err
		}
	}
	if resources != nil && len(resources.Entries) > 0 {
		resourcesDir := filepath.Join(profilePath, "resources")
		if err := os.MkdirAll(resourcesDir, 0755); err != nil {
			return err
		}
		for _, entry := range resources.Entries {
			if err := os.WriteFile(filepath.Join(resourcesDir, entry.Filename), entry.Content, 0644); err != nil {
				return err
			}
		}
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
	_, err := NewProfileQuery().WhereName(profile.Name).First()
	if err == nil {
		// Found existing profile with same name, return friendly error message
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// If it's not "record not found" error, it's another database error
		return err
	}

	// for pipeline
	if profile.ImplantConfig == nil && profile.PipelineId != "" {
		pipelineModel, err := FindPipeline(profile.PipelineId)
		if err != nil {
			return fmt.Errorf("pipline not found, err: %s", err)
		}

		params, _ := implanttypes.UnmarshalProfileParams([]byte(profile.Params))
		if params != nil && params.REMPipeline != "" {
			remPipelineModel, err := FindPipeline(params.REMPipeline)
			if err != nil {
				return fmt.Errorf("pipline not found, err: %s", err)
			}
			profileConfig := remPipelineModel.DefaultRemProfile(pipelineModel)
			profile.ImplantConfig, err = profileConfig.ToYAML()
			if err != nil {
				return err
			}
		} else {
			profileConfig, err := pipelineModel.ToProfile(nil)
			if err != nil {
				return err
			}
			profile.ImplantConfig, err = profileConfig.ToYAML()
			if err != nil {
				return err
			}
		}

	}
	if profile.ImplantConfig == nil {
		profile.ImplantConfig = consts.DefaultProfile
	}

	// Generate UUID first for the disk path.
	id, err := uuid.NewV4()
	if err != nil {
		return err
	}
	profilePath := filepath.Join(configs.ProfilePath, id.String())

	// Handle zip uploads.
	contentType := fileutils.DetectContentType(profile.ImplantConfig)
	if contentType == "zip" {
		if err := os.MkdirAll(profilePath, 0700); err != nil {
			return err
		}
		if err := fileutils.DecompressBase64ToFiles(string(profile.ImplantConfig), profilePath); err != nil {
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

		config, err := implanttypes.LoadProfile(yamlContent)
		if err != nil {
			return fmt.Errorf("failed to parse yaml config: %w", err)
		}

		if err := config.ValidateProfileFiles(profilePath); err != nil {
			return fmt.Errorf("profile validation failed: %w", err)
		}

		profile.ImplantConfig = yamlContent

		// After ZIP decompression, overwrite with prelude/resources from proto if provided separately.
		if profile.PreludeConfig != nil {
			if err := os.WriteFile(filepath.Join(profilePath, "prelude.yaml"), profile.PreludeConfig, 0644); err != nil {
				return err
			}
		}
		if profile.Resources != nil && len(profile.Resources.Entries) > 0 {
			resourcesDir := filepath.Join(profilePath, "resources")
			if err := os.MkdirAll(resourcesDir, 0755); err != nil {
				return err
			}
			for _, entry := range profile.Resources.Entries {
				if err := os.WriteFile(filepath.Join(resourcesDir, entry.Filename), entry.Content, 0644); err != nil {
					return err
				}
			}
		}
	} else {
		// Non-zip: validate and write to disk.
		_, err := implanttypes.LoadProfile(profile.ImplantConfig)
		if err != nil {
			return fmt.Errorf("profile validation failed: %w", err)
		}
		if err := writeProfileDisk(profilePath, profile.ImplantConfig, profile.PreludeConfig, profile.Resources); err != nil {
			return err
		}
	}

	model := &models.Profile{
		ID:         id,
		Name:       profile.Name,
		ParamsData: profile.Params,
		PipelineID: profile.PipelineId,
	}

	return Session().Create(model).Error
}
func GetProfile(name string) (*implanttypes.ProfileConfig, error) {
	profileModel, err := NewProfileQuery().WhereName(name).WithPipeline().First()
	if err != nil {
		return nil, err
	}
	if profileModel.PipelineID != "" && profileModel.Pipeline == nil {
		return nil, types.ErrNotFoundPipeline
	}

	// Read implant.yaml from disk.
	implantConfig, err := os.ReadFile(filepath.Join(profileModel.DiskPath(), "implant.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to read implant.yaml from disk: %w", err)
	}

	profile, err := implanttypes.LoadProfile(implantConfig)
	if err != nil {
		return nil, err
	}
	if profileModel.Name != "" {
		profile.Basic.Name = profileModel.Name
	}

	if profileModel.Pipeline != nil {
		// Create a simple target for backwards compatibility.
		target := implanttypes.Target{
			Address: profileModel.Pipeline.Address(),
		}

		// Set TLS configuration for TLS pipelines.
		if profileModel.Pipeline.Tls.Enable {
			target.TLS = &implanttypes.TLSProfile{
				Enable: true,
			}
		}

		profile.Basic.Targets = []implanttypes.Target{target}
		profile.Basic.Encryption = profileModel.Pipeline.Encryption.Choice().Type
		profile.Basic.Key = profileModel.Pipeline.Encryption.Choice().Key
		// profile.Basic.Protocol = profileModel.Pipeline.Type
		// profile.Basic.TLS.Enable = profileModel.Pipeline.Tls.Enable

		if profileModel.Pipeline.Secure != nil && profileModel.Pipeline.Secure.Enable {
			// Profile must store the keys needed at implant compile time:
			// - server public key: used by implant to encrypt data sent to server
			// - implant private key: used by implant to decrypt data from server

			profile.Basic.Secure = &implanttypes.SecureProfile{
				Enable:            true,
				ServerPublicKey:   profileModel.Pipeline.Secure.ServerPublicKey,
				ImplantPrivateKey: profileModel.Pipeline.Secure.ImplantPrivateKey,
			}
		}
		// Note: protocol field was removed; protocol is now determined by target-specific configuration.
	}
	if params := profileModel.Params; params != nil {
		profile.Basic.Cron = profileModel.Params.Cron
		profile.Basic.Jitter = profileModel.Params.Jitter
		if params.REMPipeline != "" {
			// For REM protocol, add to targets list.
			pipeline, err := FindPipeline(params.REMPipeline)
			if err != nil {
				return nil, err
			}
			// Append REM target to the targets list.
			profile.Basic.Targets = append(profile.Basic.Targets, implanttypes.Target{
				Address: pipeline.Address(),
				REM: &implanttypes.REMProfile{
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

// GetProfileContent reads implant.yaml from disk
func GetProfileContent(profileName string) ([]byte, error) {
	profileModel, err := GetProfileByName(profileName)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(profileModel.DiskPath(), "implant.yaml"))
}

func GetProfiles() (Profiles, error) {
	return NewProfileQuery().WithPipeline().OrderByCreated().Find()
}

func GetProfileByName(profileName string) (*models.Profile, error) {
	return NewProfileQuery().WhereName(profileName).WithPipeline().OrderByCreated().First()
}

// FindBuildersByPipelineID returns artifacts whose profile belongs to the given pipeline.
func FindBuildersByPipelineID(pipelineID string) ([]*models.Artifact, error) {
	return NewArtifactQuery().WherePipelineID(pipelineID).WithProfile().Find()
}

func DeleteProfileByName(profileName string) error {
	// Check if profile exists first
	existingProfile, err := NewProfileQuery().WhereName(profileName).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("profile '%s' not found", profileName)
	} else if err != nil {
		return err
	}

	// Delete disk files using UUID path.
	profilePath := existingProfile.DiskPath()
	if fileutils.Exist(profilePath) {
		if err := fileutils.ForceRemoveAll(profilePath); err != nil {
			return fmt.Errorf("could not remove profile '%s': %w", profileName, err)
		}
	}

	// Execute deletion
	if err := NewProfileQuery().WhereName(profileName).Delete(); err != nil {
		return fmt.Errorf("failed to delete profile '%s': %v", profileName, err)
	}

	return nil
}

// UpdateProfileDisk updates the profile's configuration files on disk.
func UpdateProfileDisk(profileName string, implantConfig []byte, preludeConfig []byte, resources *clientpb.BuildResources) error {
	profileModel, err := GetProfileByName(profileName)
	if err != nil {
		return err
	}
	profilePath := profileModel.DiskPath()

	// implant.yaml must be written.
	if err := os.MkdirAll(profilePath, 0700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(profilePath, "implant.yaml"), implantConfig, 0644); err != nil {
		return err
	}
	if preludeConfig != nil {
		if err := os.WriteFile(filepath.Join(profilePath, "prelude.yaml"), preludeConfig, 0644); err != nil {
			return err
		}
	}
	if resources != nil && len(resources.Entries) > 0 {
		resourcesDir := filepath.Join(profilePath, "resources")
		if err := os.MkdirAll(resourcesDir, 0755); err != nil {
			return err
		}
		for _, entry := range resources.Entries {
			if err := os.WriteFile(filepath.Join(resourcesDir, entry.Filename), entry.Content, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetProfileFullConfig reads all profile configuration files from disk.
func GetProfileFullConfig(profileName string) (implantConfig []byte, preludeConfig []byte, resources *clientpb.BuildResources, err error) {
	profileModel, err := GetProfileByName(profileName)
	if err != nil {
		return nil, nil, nil, err
	}
	return readProfileDisk(profileModel.DiskPath())
}

// GetProfileByNameWithConfig returns a Profile protobuf with full disk configuration.
func GetProfileByNameWithConfig(profileName string) (*clientpb.Profile, error) {
	profileModel, err := GetProfileByName(profileName)
	if err != nil {
		return nil, err
	}
	pb := profileModel.ToProtobuf()

	implant, prelude, resources, err := readProfileDisk(profileModel.DiskPath())
	if err != nil {
		return nil, err
	}
	pb.ImplantConfig = implant
	pb.PreludeConfig = prelude
	pb.Resources = resources

	return pb, nil
}

// ============================================
// Artifact Operations
// ============================================

func SaveArtifactFromConfig(req *clientpb.BuildConfig) (*models.Artifact, error) {
	target, ok := consts.GetBuildTarget(req.Target)
	if !ok {
		return nil, types.ErrInvalidateTarget
	}
	format := resolveArtifactFormat(target.OS, req.BuildType, req.OutputType)
	builder := models.Artifact{
		Name:        req.BuildName,
		ProfileName: req.ProfileName,
		Target:      req.Target,
		Type:        req.BuildType,
		Source:      req.Source,
		Arch:        target.Arch,
		Os:          target.OS,
		Format:      format,
		Comment:     req.Comment,
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
		return nil, types.ErrInvalidateTarget
	}
	format := resolveArtifactFormat(target.OS, req.BuildType, req.OutputType)
	artifact := models.Artifact{
		ID:          ID,
		Name:        req.BuildName,
		ProfileName: req.ProfileName,
		Target:      req.Target,
		Type:        req.BuildType,
		Source:      req.Source,
		Arch:        target.Arch,
		Os:          target.OS,
		Format:      format,
		Comment:     req.Comment,
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

// resolveArtifactFormat returns file extension (with dot) based on OS/buildType/outputType.
// outputType: "" or "executable" (default), "lib", "shellcode"
func resolveArtifactFormat(osName, buildType string, outputType string) string {
	isLib := outputType == "lib"
	isShellcode := outputType == "shellcode"

	switch osName {
	case consts.Windows:
		// modules/3rd are always DLL
		if buildType == consts.CommandBuildModules || buildType == consts.CommandBuild3rdModules {
			return consts.DllFile
		}
		if isShellcode {
			return consts.ShellcodeFile
		}
		if isLib {
			return consts.DllFile
		}
		return consts.PEFile
	case consts.Linux:
		if isLib {
			return ".so"
		}
		return ""
	case consts.Darwin:
		if isLib {
			return ".dylib"
		}
		return ""
	default:
		return ""
	}
}

func GetValidArtifacts() ([]*models.Artifact, error) {
	artifacts, err := NewArtifactQuery().WherePathNotEmpty().WithProfilePipeline().Find()
	if err != nil {
		return nil, err
	}
	// Filter by filesystem existence (file may have been deleted from disk).
	var validArtifacts []*models.Artifact
	for _, artifact := range artifacts {
		if _, err := os.Stat(artifact.Path); err == nil {
			validArtifacts = append(validArtifacts, artifact)
		}
	}
	return validArtifacts, nil
}

// FindArtifact finds an artifact by various criteria
func FindArtifact(target *clientpb.Artifact, bin bool) (*clientpb.Artifact, error) {
	var artifact *models.Artifact
	var err error

	// Find builder by ID or name.
	if target.Id != 0 {
		artifact, err = NewArtifactQuery().WhereID(target.Id).WhereStatus(consts.BuildStatusCompleted).Last()
	} else if target.Name != "" {
		artifact, err = NewArtifactQuery().WhereName(target.Name).WhereStatus(consts.BuildStatusCompleted).Last()
	} else if target.Profile != "" {
		artifact, err = NewArtifactQuery().WhereProfileName(target.Profile).WhereStatus(consts.BuildStatusCompleted).Last()
	} else {
		var builders Artifacts
		builders, err = NewArtifactQuery().
			WhereOs(target.Platform).WhereArch(target.Arch).WhereType(target.Type).WhereStatus(consts.BuildStatusCompleted).
			WithProfilePipeline().Find()
		if err == nil {
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
	}
	if err != nil {
		return nil, fmt.Errorf("error finding artifact: %v, target: %+v", err, target)
	}
	if artifact == nil {
		return nil, types.ErrNotFoundArtifact
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
	return NewArtifactQuery().
		WhereType(consts.CommandBuildBeacon).
		WherePipelineID(pipelineName).
		WithProfile().
		Last()
}

func GetArtifactByName(name string) (*models.Artifact, error) {
	return NewArtifactQuery().WhereName(name).WithProfile().First()
}

func GetArtifactById(id uint32) (*models.Artifact, error) {
	return NewArtifactQuery().WhereID(id).WithProfile().First()
}

func GetArtifactWithSaas() (Artifacts, error) {
	return NewArtifactQuery().WhereSource(consts.ArtifactFromSaas).Find()
}

// GetBeaconBuilderByRelinkID finds beacon artifacts with a matching RelinkBeaconID.
// NOTE: RelinkBeaconID is stored inside the JSON ParamsData field. Filtering at the
// DB level would require dialect-specific JSON operators (json_extract for SQLite,
// jsonb for Postgres). The in-memory filter is acceptable because the WhereType("beacon")
// clause limits the dataset to a small subset.
func GetBeaconBuilderByRelinkID(relinkID uint32) ([]*models.Artifact, error) {
	artifacts, err := NewArtifactQuery().WhereType("beacon").Find()
	if err != nil {
		return nil, err
	}

	var result []*models.Artifact
	for _, b := range artifacts {
		var params implanttypes.ProfileParams
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
	artifact, err := NewArtifactQuery().WhereName(artifactName).First()
	if err != nil {
		return err
	}
	if artifact.Path != "" {
		if err := os.Remove(artifact.Path); err != nil {
			return err
		}
	}
	return Delete(artifact)
}

func UpdateBuilderLog(name string, logEntry string) {
	if Session() == nil {
		return
	}
	err := NewArtifactQuery().WhereName(name).Update("log", Adapter.AppendLogExpr(logEntry))
	if err != nil {
		logs.Log.Errorf("Error updating log for Artifact name %s: %v", name, err)
	}
}

func GetBuilderLogs(builderName string, limit int) (string, error) {
	builder, err := NewArtifactQuery().WhereName(builderName).First()
	if err != nil {
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
	err := NewArtifactQuery().WhereID(builderID).Update("status", status)
	if err != nil {
		logs.Log.Errorf("Error updating log for Artifact id %d: %v", builderID, err)
	}
}
