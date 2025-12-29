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

// SigforgeRequest represents the parameters for the Sigforge tool
type SigforgeRequest struct {
	Operation string // extract, copy, inject, remove, check
	SourceBin []byte // source PE file
	TargetBin []byte // target PE file (for copy operation)
	Signature []byte // signature data (for inject operation)
}

// Sigforge manipulates PE file signatures using malefic-mutant sigforge tool
func Sigforge(req *SigforgeRequest) ([]byte, error) {
	logs.Log.Infof("[mutant-sigforge] Operation: %s", req.Operation)

	operation := strings.ToLower(req.Operation)
	switch operation {
	case "extract", "copy", "inject", "remove", "check":
		// Valid operations
	default:
		return nil, fmt.Errorf("invalid operation: %s (must be extract, copy, inject, remove, or check)", req.Operation)
	}

	// Create temporary source file
	sourceFile, err := os.CreateTemp(configs.TempPath, "mutant-sigforge-source-*.exe")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp source file: %w", err)
	}
	sourcePath := sourceFile.Name()
	defer os.Remove(sourcePath)

	if _, err := sourceFile.Write(req.SourceBin); err != nil {
		sourceFile.Close()
		return nil, fmt.Errorf("failed to write source file: %w", err)
	}
	sourceFile.Close()

	// Build command arguments based on operation
	mutantBin := "malefic-mutant"
	if runtime.GOOS == "windows" {
		mutantBin = "malefic-mutant.exe"
	}
	mutantPath := filepath.Join(configs.BinPath, mutantBin)

	var args []string
	var resultPath string

	switch operation {
	case "extract":
		// Extract signature from signed PE
		outputPath := filepath.Join(configs.TempPath, "signature.bin")
		defer os.Remove(outputPath)
		args = []string{"tool", "sigforge", "extract", sourcePath, outputPath}
		resultPath = outputPath

	case "copy":
		// Copy signature from source to target
		if len(req.TargetBin) == 0 {
			return nil, fmt.Errorf("target binary is required for copy operation")
		}

		targetFile, err := os.CreateTemp(configs.TempPath, "mutant-sigforge-target-*.exe")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp target file: %w", err)
		}
		targetPath := targetFile.Name()
		defer os.Remove(targetPath)

		if _, err := targetFile.Write(req.TargetBin); err != nil {
			targetFile.Close()
			return nil, fmt.Errorf("failed to write target file: %w", err)
		}
		targetFile.Close()

		outputPath := filepath.Join(configs.TempPath, "signed-output.exe")
		defer os.Remove(outputPath)
		args = []string{"tool", "sigforge", "copy", sourcePath, targetPath, outputPath}
		resultPath = outputPath

	case "inject":
		// Inject signature into PE
		if len(req.Signature) == 0 {
			return nil, fmt.Errorf("signature data is required for inject operation")
		}

		sigFile, err := os.CreateTemp(configs.TempPath, "mutant-sigforge-sig-*.bin")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp signature file: %w", err)
		}
		sigPath := sigFile.Name()
		defer os.Remove(sigPath)

		if _, err := sigFile.Write(req.Signature); err != nil {
			sigFile.Close()
			return nil, fmt.Errorf("failed to write signature file: %w", err)
		}
		sigFile.Close()

		outputPath := filepath.Join(configs.TempPath, "signed-output.exe")
		defer os.Remove(outputPath)
		args = []string{"tool", "sigforge", "inject", sigPath, sourcePath, outputPath}
		resultPath = outputPath

	case "remove":
		// Remove signature from PE
		outputPath := filepath.Join(configs.TempPath, "unsigned-output.exe")
		defer os.Remove(outputPath)
		args = []string{"tool", "sigforge", "remove", sourcePath, outputPath}
		resultPath = outputPath

	case "check":
		// Check if PE has signature
		args = []string{"tool", "sigforge", "check", sourcePath}
		// Check operation returns text output, not a file
		resultPath = ""
	}

	// Execute malefic-mutant
	logs.Log.Infof("[mutant-sigforge] Executing: %s %v", mutantPath, args)
	cmd := exec.Command(mutantPath, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		logs.Log.Errorf("[mutant-sigforge] Command failed: %s", string(output))
		return nil, fmt.Errorf("malefic-mutant sigforge failed: %v, output: %s", err, string(output))
	}

	if len(output) > 0 {
		logs.Log.Debugf("[mutant-sigforge] Output: %s", string(output))
	}

	// For check operation, return the command output as result
	if operation == "check" {
		logs.Log.Infof("[mutant-sigforge] Check result: %s", string(output))
		return output, nil
	}

	// Read the result file for other operations
	result, err := os.ReadFile(resultPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read result file: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("result is empty")
	}

	logs.Log.Infof("[mutant-sigforge] Successfully completed %s operation: %d bytes", operation, len(result))
	return result, nil
}
