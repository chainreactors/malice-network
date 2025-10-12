package build

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
)

var (
	generateConfig = "implant.yaml"
	autoRunYaml    = "prelude.yaml"
	release        = "release"
	malefic        = "malefic"
	modules        = "malefic_modules"
	modules3rd     = "malefic_3rd"
	prelude        = "malefic-prelude"
	pulse          = "malefic-pulse"
)

// GenerateProfile - Generate profile
// first recover profile from database
// then use Generate overwrite profile
//func GenerateProfile(req *clientpb.BuildConfig) ([]byte, error) {
//	var profile *types.ProfileConfig
//	var err error
//	if req.Type == consts.CommandBuildModules {
//		profileByte := consts.DefaultProfile
//		profile, err = types.LoadProfile(profileByte)
//		if err != nil {
//			return nil, err
//		}
//		profileParams, err := types.UnmarshalProfileParams(req.ParamsBytes)
//		if err != nil {
//			return nil, err
//		}
//		if profileParams.Modules != "" {
//			profile.Implant.Modules = strings.Split(profileParams.Modules, ",")
//		}
//	} else {
//		profile, err = db.GetProfile(req.ProfileName)
//		if err != nil {
//			return nil, err
//		}
//	}
//	err = UpdateGeneratorConfig(req, profile)
//	if err != nil {
//		return nil, err
//	}
//	data, err := yaml.Marshal(profile)
//	if err != nil {
//		return nil, err
//	}
//	if req.Source == consts.ArtifactFromDocker {
//		path := filepath.Join(configs.SourceCodePath, generateConfig)
//		err = os.WriteFile(path, data, 0644)
//		if err != nil {
//			return nil, err
//		}
//	}
//	return data, nil
//}

func MoveBuildOutput(target, buildType string, enable3RD bool) (string, string, error) {
	var sourcePath string
	name := encoders.UUID()
	switch {
	case strings.Contains(target, "windows"):
		if buildType == consts.CommandBuildModules {
			if enable3RD {
				sourcePath = filepath.Join(configs.TargetPath, target, release, modules3rd+consts.DllFile)
			} else {
				sourcePath = filepath.Join(configs.TargetPath, target, release, modules+consts.DllFile)
			}
		} else if buildType == consts.CommandBuildPrelude {
			sourcePath = filepath.Join(configs.TargetPath, target, release, prelude+consts.PEFile)
		} else if buildType == consts.CommandBuildPulse {
			sourcePath = filepath.Join(configs.TargetPath, target, release, pulse+consts.PEFile)
		} else {
			sourcePath = filepath.Join(configs.TargetPath, target, release, malefic+consts.PEFile)
		}
	case strings.Contains(target, "darwin"):
		sourcePath = filepath.Join(configs.TargetPath, target, release, malefic)
	case strings.Contains(target, "linux"):
		if buildType == consts.CommandBuildPrelude {
			sourcePath = filepath.Join(configs.TargetPath, target, release, prelude)
		} else {
			sourcePath = filepath.Join(configs.TargetPath, target, release, malefic)
		}
	}
	dstPath := filepath.Join(configs.TempPath, name)
	err := fileutils.MoveFile(sourcePath, dstPath)
	if err != nil {
		return "", "", err
	}
	return sourcePath, dstPath, nil
}

func GetFilePath(name, target, buildType, format string) string {
	switch {
	case strings.Contains(target, "windows"):
		if buildType == consts.CommandBuildModules {
			name = name + consts.DllFile
		} else if buildType == consts.CommandBuildPrelude {
			name = name + consts.PEFile
		} else if buildType == consts.CommandBuildPulse {
			name = name + consts.PEFile
		} else {
			name = name + consts.PEFile
		}
	}
	return name
}

// UpdateGeneratorConfig - Update the generator config
//func UpdateGeneratorConfig(req *clientpb.BuildConfig, config *types.ProfileConfig) error {
//	if config.Basic != nil {
//		if req.BuildName != "" {
//			config.Basic.Name = req.BuildName
//		}
//
//		if len(req.ParamsBytes) > 0 {
//			params, err := types.UnmarshalProfileParams(req.ParamsBytes)
//			if err != nil {
//				return err
//			}
//			if params.Cron != "" {
//				config.Basic.Cron = params.Cron
//			}
//
//			if params.Jitter != -1 {
//				config.Basic.Jitter = params.Jitter
//			}
//			if params.Proxy != "" {
//				if config.Basic.Proxy == nil {
//					config.Basic.Proxy = &types.ProxyProfile{}
//				}
//				config.Basic.Proxy.URL = params.Proxy
//			}
//
//			if params.Enable3RD {
//				config.Implant.ThirdModules = strings.Split(params.Modules, ",")
//				config.Implant.Enable3rd = true
//				config.Implant.Modules = []string{}
//			} else if params.Modules != "" {
//				config.Implant.Modules = strings.Split(params.Modules, ",")
//			}
//			if params.Address != "" {
//				target := types.Target{
//					Address: params.Address,
//				}
//				// 如果有HTTP配置，设置Host
//				if len(config.Basic.Targets) > 0 && config.Basic.Targets[0].Http != nil {
//					target.Http = &types.HttpProfile{
//						Method:  config.Basic.Targets[0].Http.Method,
//						Path:    config.Basic.Targets[0].Http.Path,
//						Version: config.Basic.Targets[0].Http.Version,
//						Headers: config.Basic.Targets[0].Http.Headers,
//					}
//				}
//				// 如果有TLS配置，设置SNI
//				if len(config.Basic.Targets) > 0 && config.Basic.Targets[0].TLS != nil {
//					target.TLS = &types.TLSProfile{
//						Enable:           config.Basic.Targets[0].TLS.Enable,
//						SNI:              params.Address,
//						SkipVerification: config.Basic.Targets[0].TLS.SkipVerification,
//					}
//				}
//				config.Basic.Targets = []types.Target{target}
//				config.Pulse.Target = params.Address
//				if config.Pulse.Http != nil {
//					config.Pulse.Http.Headers = make(map[string]string)
//					for k, v := range config.Pulse.Http.Headers {
//						config.Pulse.Http.Headers[k] = v
//					}
//				}
//			}
//			if params.AutoRunFile != "" {
//				config.Implant.AutoRun = ContainerAutoRunPath
//			}
//		}
//	}
//	if req.ArtifactId != 0 && config.Pulse.Flags.ArtifactID == 0 {
//		config.Pulse.Flags.ArtifactID = req.ArtifactId
//	}
//
//	if req.Type == consts.CommandBuildBind {
//		config.Implant.Mod = consts.CommandBuildBind
//	}
//	return nil
//}

// ProcessAutorunZip processes autorun.zip file, extracting only files from the resource directory to the target root directory
func ProcessAutorunZip(zipData []byte, targetPath string) error {
	return fileutils.DecompressZipSubdirToRoot(zipData, "resources", targetPath)
}

// CopyProfileFilesExceptConfig copies all files from the profile directory except implant.yaml to the target path
func CopyProfileFilesExceptConfig(profilePath, targetPath string) error {
	return fileutils.CopyDirectoryExcept(profilePath, targetPath, []string{"implant.yaml"})
}

// ProcessAutorunWithProfile processes autorun.zip and copies profile files
func ProcessAutorunWithProfile(paramsBytes []byte, profilePath, targetPath string) error {
	if len(paramsBytes) == 0 {
		return nil
	}

	params, err := types.UnmarshalProfileParams(paramsBytes)
	if err != nil {
		return err
	}

	if params.AutoRunFile != "" {
		zipData, err := fileutils.DecodeBase64OrRaw(params.AutoRunFile)
		if err != nil {
			return fmt.Errorf("failed to decode autorun file: %w", err)
		}

		if err := ProcessAutorunZip(zipData, targetPath); err != nil {
			return fmt.Errorf("failed to process autorun zip: %w", err)
		}
	}

	if _, err := os.Stat(profilePath); !os.IsNotExist(err) {
		if err = CopyProfileFilesExceptConfig(profilePath, targetPath); err != nil {
			return fmt.Errorf("failed to copy profile files: %w", err)
		}
	}

	return nil
}

// extractPreludeYamlBase64 从 zipData 中提取 prelude.yaml 并返回 base64 内容
func extractPreludeYamlBase64(zipData []byte) (string, error) {
	var result string
	err := fileutils.WithTempDir("prelude_temp_*", func(tempDir string) error {
		preludeYamlPath := filepath.Join(tempDir, autoRunYaml)

		if err := fileutils.ExtractZipWithFilter(zipData, tempDir, func(filename string) bool {
			return filename == autoRunYaml
		}); err != nil {
			return fmt.Errorf("failed to extract prelude.yaml from zip: %w", err)
		}

		if !fileutils.Exist(preludeYamlPath) {
			return fmt.Errorf("prelude.yaml not found in zip content")
		}

		content, err := os.ReadFile(preludeYamlPath)
		if err != nil {
			return fmt.Errorf("failed to read prelude.yaml: %w", err)
		}

		result = base64.StdEncoding.EncodeToString(content)
		return nil
	})
	return result, err
}

// processProfileOnlyCase handles the case where only profile exists (no autorun file)
func processProfileOnlyCase(profilePath string, params *types.ProfileParams) (string, string, error) {
	filePaths, err := fileutils.CollectFilePaths(profilePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to walk profilePath: %w", err)
	}

	if len(filePaths) == 0 {
		return "", "", errs.ErrNoAutoRunFile
	}

	zipData, err := fileutils.CompressFilesZip(filePaths)
	if err != nil {
		return "", "", fmt.Errorf("failed to create zip from profilePath: %w", err)
	}
	zipBase64 := fileutils.EncodeBase64OrRaw(zipData)
	params.AutoRunFile = zipBase64
	return "", params.String(), nil
}

// processPreludeWithOptionalProfile handles the case where prelude file exists
func processPreludeWithOptionalProfile(zipData []byte, profilePath string, profileExists bool, params *types.ProfileParams) (string, string, error) {
	preludeYamlBase64, err := extractPreludeYamlBase64(zipData)
	if err != nil {
		return "", "", err
	}

	if profileExists {
		newZipBase64, err := createCombinedZip(zipData, profilePath)
		if err != nil {
			return "", "", fmt.Errorf("failed to create combined zip: %w", err)
		}
		params.AutoRunFile = newZipBase64
	}

	return preludeYamlBase64, params.String(), nil
}

func ProcessAutorunZipToBase64(paramsByte []byte, profileName string) (string, string, error) {
	if len(paramsByte) == 0 {
		return "", "", nil
	}

	params, err := types.UnmarshalProfileParams(paramsByte)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse params: %w", err)
	}

	profilePath := filepath.Join(configs.ProfilePath, profileName)
	profileExists := fileutils.Exist(profilePath)
	autoRunFileEmpty := params.AutoRunFile == ""

	switch {
	case autoRunFileEmpty && profileExists:
		return processProfileOnlyCase(profilePath, params)

	case !autoRunFileEmpty:
		zipData, err := base64.StdEncoding.DecodeString(params.AutoRunFile)
		if err != nil {
			return "", "", fmt.Errorf("failed to decode prelude.zip base64: %w", err)
		}
		return processPreludeWithOptionalProfile(zipData, profilePath, profileExists, params)

	case autoRunFileEmpty && !profileExists:
		return "", params.String(), nil

	default:
		return "", "", fmt.Errorf("unexpected branch")
	}
}

// createCombinedZip creates a new zip containing files from prelude.zip and profile directory
func createCombinedZip(zipData []byte, profilePath string) (string, error) {
	var result string
	err := fileutils.WithTempDir("combined_zip_*", func(tempDir string) error {
		// Extract all files from prelude.zip except prelude.yaml
		if err := fileutils.ExtractZipWithFilter(zipData, tempDir, func(filename string) bool {
			return filename != autoRunYaml // Exclude prelude.yaml
		}); err != nil {
			return fmt.Errorf("failed to extract files from prelude.zip: %w", err)
		}

		// Copy profile files except implant.yaml
		if profilePath != "" {
			if err := fileutils.CopyDirectoryExcept(profilePath, tempDir, []string{"implant.yaml"}); err != nil {
				return fmt.Errorf("failed to copy profile files: %w", err)
			}
		}

		// Collect all file paths and create zip
		filePaths, err := fileutils.CollectFilePaths(tempDir)
		if err != nil {
			return fmt.Errorf("failed to collect file paths: %w", err)
		}

		zip, err := fileutils.CompressFilesZip(filePaths)
		if err != nil {
			return fmt.Errorf("failed to create zip: %w", err)
		}
		zipBase64 := fileutils.EncodeBase64OrRaw(zip)

		result = zipBase64
		return nil
	})
	return result, err
}

func WriteProfile(data []byte) error {
	path := filepath.Join(configs.SourceCodePath, generateConfig)
	return os.WriteFile(path, data, 0644)
}

func WriteAutoYaml(data []byte) error {
	path := filepath.Join(configs.SourceCodePath, autoRunYaml)
	return os.WriteFile(path, data, 0644)
}
