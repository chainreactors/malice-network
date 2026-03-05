package mutant

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/configs"
)

type PatchConfigRequest struct {
	TemplateBin []byte
	ImplantYaml []byte
	Output      string
}

// PatchConfig patches a pre-compiled template binary with runtime config
// by invoking malefic-mutant patch-config.
func PatchConfig(req *PatchConfigRequest) ([]byte, error) {
	logs.Log.Infof("[mutant-patch] Patching template binary: %d bytes", len(req.TemplateBin))

	templateFile, err := os.CreateTemp(configs.TempPath, "mutant-patch-template-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp template file: %w", err)
	}
	templatePath := templateFile.Name()
	defer os.Remove(templatePath)

	if _, err := templateFile.Write(req.TemplateBin); err != nil {
		templateFile.Close()
		return nil, fmt.Errorf("failed to write template file: %w", err)
	}
	templateFile.Close()

	yamlFile, err := os.CreateTemp(configs.TempPath, "mutant-patch-implant-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp yaml file: %w", err)
	}
	yamlPath := yamlFile.Name()
	defer os.Remove(yamlPath)

	if _, err := yamlFile.Write(req.ImplantYaml); err != nil {
		yamlFile.Close()
		return nil, fmt.Errorf("failed to write yaml file: %w", err)
	}
	yamlFile.Close()

	outputFile, err := os.CreateTemp(configs.TempPath, "mutant-patch-output-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp output file: %w", err)
	}
	outputPath := outputFile.Name()
	outputFile.Close()
	defer os.Remove(outputPath)

	mutantBin := "malefic-mutant"
	if runtime.GOOS == "windows" {
		mutantBin = "malefic-mutant.exe"
	}
	mutantPath := filepath.Join(configs.BinPath, mutantBin)

	args := []string{
		"tool", "patch-config",
		"-f", templatePath,
		"--from-implant", yamlPath,
		"-o", outputPath,
	}

	logs.Log.Infof("[mutant-patch] Executing: %s %v", mutantPath, args)
	cmd := exec.Command(mutantPath, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		logs.Log.Errorf("[mutant-patch] Command failed: %s", string(output))
		return nil, fmt.Errorf("malefic-mutant patch-config failed: %v, output: %s", err, string(output))
	}

	if len(output) > 0 {
		logs.Log.Debugf("[mutant-patch] Output: %s", string(output))
	}

	patched, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read patched output: %w", err)
	}

	if len(patched) == 0 {
		return nil, fmt.Errorf("patched output is empty")
	}

	logs.Log.Infof("[mutant-patch] Successfully patched binary: %d bytes", len(patched))
	return patched, nil
}
