package build

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// BuildInputFlagSet defines file input flags for beacon/bind builds.
// Includes all four input flags: --implant-path, --prelude-path, --resources-path, --archive-path.
func BuildInputFlagSet(f *pflag.FlagSet) {
	f.String("implant-path", "", "path to implant.yaml file")
	f.String("prelude-path", "", "path to prelude.yaml file")
	f.String("resources-path", "", "path to resources directory")
	f.String("archive-path", "", "path to build archive (zip)")
	common.SetFlagSetGroup(f, "input")
}

// PreludeInputFlagSet defines file input flags for prelude builds.
// Includes --prelude-path, --resources-path, --archive-path (no --implant-path).
func PreludeInputFlagSet(f *pflag.FlagSet) {
	f.String("prelude-path", "", "path to prelude.yaml file")
	f.String("resources-path", "", "path to resources directory")
	f.String("archive-path", "", "path to build archive (zip)")
	common.SetFlagSetGroup(f, "input")
}

// ImplantInputFlagSet defines the implant-path flag for pulse builds.
func ImplantInputFlagSet(f *pflag.FlagSet) {
	f.String("implant-path", "", "path to implant.yaml file")
	common.SetFlagSetGroup(f, "input")
}

// loadBuildInputs reads build configuration files from command flags.
// Override chain: archive < individual files (--implant-path, --prelude-path, --resources-path).
func loadBuildInputs(cmd *cobra.Command) (implant []byte, prelude []byte, resources *clientpb.BuildResources, err error) {
	// Layer 1: Archive (base layer from file inputs)
	if cmd.Flags().Changed("archive-path") {
		archivePath, _ := cmd.Flags().GetString("archive-path")
		var data []byte
		data, err = os.ReadFile(archivePath)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to read archive %s: %w", archivePath, err)
		}
		implant, prelude, resources, err = parseArchive(data)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	// Layer 2: Individual files override archive contents
	if cmd.Flags().Changed("implant-path") {
		implantPath, _ := cmd.Flags().GetString("implant-path")
		implant, err = os.ReadFile(implantPath)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to read implant file %s: %w", implantPath, err)
		}
	}

	if cmd.Flags().Changed("prelude-path") {
		preludePath, _ := cmd.Flags().GetString("prelude-path")
		prelude, err = os.ReadFile(preludePath)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to read prelude file %s: %w", preludePath, err)
		}
	}

	if cmd.Flags().Changed("resources-path") {
		resourcesPath, _ := cmd.Flags().GetString("resources-path")
		resources, err = readResourcesDir(resourcesPath)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return
}

// parseArchive extracts implant.yaml, prelude.yaml, and resources from a zip archive.
// Unlike ProcessAutorunZipFromBytes, this function does not require any specific file to be present.
func parseArchive(data []byte) (implant []byte, prelude []byte, resources *clientpb.BuildResources, err error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read zip archive: %w", err)
	}

	var resourceEntries []*clientpb.ResourceEntry

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		rc, openErr := f.Open()
		if openErr != nil {
			return nil, nil, nil, fmt.Errorf("failed to open %s in archive: %w", f.Name, openErr)
		}

		content, readErr := io.ReadAll(rc)
		rc.Close()
		if readErr != nil {
			return nil, nil, nil, fmt.Errorf("failed to read %s in archive: %w", f.Name, readErr)
		}

		switch {
		case f.Name == "implant.yaml":
			implant = content
		case f.Name == "prelude.yaml":
			prelude = content
		case strings.HasPrefix(f.Name, "resources/"):
			filename := strings.TrimPrefix(f.Name, "resources/")
			if filename != "" {
				resourceEntries = append(resourceEntries, &clientpb.ResourceEntry{
					Filename: filename,
					Content:  content,
				})
			}
		}
	}

	if len(resourceEntries) > 0 {
		resources = &clientpb.BuildResources{Entries: resourceEntries}
	}

	return implant, prelude, resources, nil
}

// readResourcesDir reads all files from a directory as resource entries.
func readResourcesDir(dirPath string) (*clientpb.BuildResources, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read resources directory %s: %w", dirPath, err)
	}

	var resourceEntries []*clientpb.ResourceEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dirPath, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read resource file %s: %w", e.Name(), err)
		}
		resourceEntries = append(resourceEntries, &clientpb.ResourceEntry{
			Filename: e.Name(),
			Content:  content,
		})
	}

	if len(resourceEntries) == 0 {
		return nil, nil
	}
	return &clientpb.BuildResources{Entries: resourceEntries}, nil
}
