package mutant

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/configs"
)

// StripRequest represents the parameters for the Strip tool
type StripRequest struct {
	Bin         []byte
	CustomPaths []string
}

// Strip removes paths from binary files using malefic-mutant strip tool
func Strip(req *StripRequest) ([]byte, error) {
	logs.Log.Infof("[mutant-strip] Stripping paths from binary: %d bytes", len(req.Bin))

	// Create temporary input file
	inputFile, err := os.CreateTemp(configs.TempPath, "mutant-strip-input-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp input file: %w", err)
	}
	inputPath := inputFile.Name()
	defer os.Remove(inputPath)

	if _, err := inputFile.Write(req.Bin); err != nil {
		inputFile.Close()
		return nil, fmt.Errorf("failed to write input file: %w", err)
	}
	inputFile.Close()

	// Create temporary output file
	outputFile, err := os.CreateTemp(configs.TempPath, "mutant-strip-output-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp output file: %w", err)
	}
	outputPath := outputFile.Name()
	outputFile.Close()
	defer os.Remove(outputPath)

	// Build command arguments
	args := []string{"tool", "strip", "-i", inputPath, "-o", outputPath}

	// Add custom paths if provided
	if len(req.CustomPaths) > 0 {
		args = append(args, "--custom-paths", strings.Join(req.CustomPaths, ","))
	}

	// Execute malefic-mutant
	mutantBin := "malefic-mutant"
	if runtime.GOOS == "windows" {
		mutantBin = "malefic-mutant.exe"
	}
	mutantPath := filepath.Join(configs.BinPath, mutantBin)

	logs.Log.Infof("[mutant-strip] Executing: %s %v", mutantPath, args)
	cmd := exec.Command(mutantPath, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		logs.Log.Errorf("[mutant-strip] Command failed: %s", string(output))
		return nil, fmt.Errorf("malefic-mutant strip failed: %v, output: %s", err, string(output))
	}

	if len(output) > 0 {
		logs.Log.Debugf("[mutant-strip] Output: %s", string(output))
	}

	// Read the processed binary
	stripped, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read output file: %w", err)
	}

	if len(stripped) == 0 {
		return nil, fmt.Errorf("stripped binary is empty")
	}

	logs.Log.Infof("[mutant-strip] Successfully stripped binary: %d bytes", len(stripped))
	return stripped, nil
}
