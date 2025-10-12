package profile

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainreactors/malice-network/helper/consts"

	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
)

// ProcessAutorunZip
func ProcessAutorunZip(zipPath string) (*clientpb.BuildConfig, error) {
	zipData, err := os.ReadFile(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read zip file: %w", err)
	}
	return ProcessAutorunZipFromBytes(zipData)
}

// ProcessAutorunZipFromBytes
func ProcessAutorunZipFromBytes(zipData []byte) (*clientpb.BuildConfig, error) {
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to read zip data: %w", err)
	}

	var autorunContent []byte
	var configContent []byte
	var resourceFiles []*clientpb.ResourceEntry

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %w", f.Name, err)
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", f.Name, err)
		}

		switch {
		case f.Name == "prelude.yaml":
			autorunContent = content
		case f.Name == "implant.yaml":
			configContent = content
		case strings.HasPrefix(f.Name, "resources/") && !f.FileInfo().IsDir():
			filename := strings.TrimPrefix(f.Name, "resources/")
			if filename == "prelude.yaml" {
				autorunContent = content
			} else if filename != "" {
				entry := &clientpb.ResourceEntry{
					Filename: filename,
					Content:  content,
				}
				resourceFiles = append(resourceFiles, entry)
			}
		}
	}

	// 3.
	if autorunContent == nil {
		return nil, fmt.Errorf("prelude.yaml is required in zip file")
	}

	buildConfig := &clientpb.BuildConfig{
		BuildType:     consts.CommandBuildPrelude,
		MaleficConfig: configContent,
		PreludeConfig: autorunContent,
		Resources: &clientpb.BuildResources{
			Entries: resourceFiles,
		},
	}

	return buildConfig, nil
}

// WriteBuildConfigToPath
func WriteBuildConfigToPath(buildConfig *clientpb.BuildConfig, srcPath string) error {
	// 1. prelude.yaml (PreludeConfig)
	if buildConfig.PreludeConfig != nil {
		autorunPath := filepath.Join(srcPath, "prelude.yaml")
		if err := os.WriteFile(autorunPath, buildConfig.PreludeConfig, 0644); err != nil {
			return fmt.Errorf("failed to write prelude.yaml: %w", err)
		}
	}

	// 2. implant.yaml (MaleficConfig)
	if buildConfig.MaleficConfig != nil {
		configPath := filepath.Join(srcPath, "implant.yaml")
		if err := os.WriteFile(configPath, buildConfig.MaleficConfig, 0644); err != nil {
			return fmt.Errorf("failed to write implant.yaml: %w", err)
		}
	}

	// 3. resources
	if buildConfig.Resources != nil && len(buildConfig.Resources.Entries) > 0 {
		resourcesDir := filepath.Join(srcPath, "resources")
		if err := os.MkdirAll(resourcesDir, 0755); err != nil {
			return fmt.Errorf("failed to create resources directory: %w", err)
		}

		for _, entry := range buildConfig.Resources.Entries {
			if entry.Filename != "" && entry.Content != nil {
				resourcePath := filepath.Join(resourcesDir, entry.Filename)
				if err := os.WriteFile(resourcePath, entry.Content, 0644); err != nil {
					return fmt.Errorf("failed to write resource file %s: %w", entry.Filename, err)
				}
			}
		}
	}

	return nil
}
