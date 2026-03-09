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

// SrdiRequest represents the parameters for the SRDI tool
type SrdiRequest struct {
	Bin          []byte
	Arch         string // x86 or x64
	FunctionName string
	Platform     string // win
	Type         string // link or malefic
	Userdata     []byte
}

// Srdi converts DLL to shellcode using malefic-mutant srdi tool
func Srdi(req *SrdiRequest) ([]byte, error) {
	logs.Log.Infof("[mutant-srdi] Converting DLL to shellcode: %d bytes", len(req.Bin))

	// Create temporary input file
	inputFile, err := os.CreateTemp(configs.TempPath, "mutant-srdi-input-*.dll")
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
	outputFile, err := os.CreateTemp(configs.TempPath, "mutant-srdi-output-*.bin")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp output file: %w", err)
	}
	outputPath := outputFile.Name()
	outputFile.Close()
	defer os.Remove(outputPath)

	// Handle userdata if provided
	var userdataPath string
	if len(req.Userdata) > 0 {
		userdataFile, err := os.CreateTemp(configs.TempPath, "mutant-srdi-userdata-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp userdata file: %w", err)
		}
		userdataPath = userdataFile.Name()
		defer os.Remove(userdataPath)

		if _, err := userdataFile.Write(req.Userdata); err != nil {
			userdataFile.Close()
			return nil, fmt.Errorf("failed to write userdata file: %w", err)
		}
		userdataFile.Close()
	}

	// Build command arguments
	args := []string{"tool", "srdi", "-i", inputPath, "-o", outputPath}

	// Add optional parameters
	if req.Arch != "" {
		args = append(args, "-a", strings.ToLower(req.Arch))
	}
	if req.FunctionName != "" {
		args = append(args, "--function-name", req.FunctionName)
	}
	if req.Platform != "" {
		args = append(args, "-p", strings.ToLower(req.Platform))
	}
	if req.Type != "" {
		args = append(args, "-t", strings.ToLower(req.Type))
	}
	if userdataPath != "" {
		args = append(args, "--userdata-path", userdataPath)
	}

	// Execute malefic-mutant
	mutantBin := "malefic-mutant"
	if runtime.GOOS == "windows" {
		mutantBin = "malefic-mutant.exe"
	}
	mutantPath := filepath.Join(configs.BinPath, mutantBin)

	logs.Log.Infof("[mutant-srdi] Executing: %s %v", mutantPath, args)
	cmd := exec.Command(mutantPath, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		logs.Log.Errorf("[mutant-srdi] Command failed: %s", string(output))
		return nil, fmt.Errorf("malefic-mutant srdi failed: %v, output: %s", err, string(output))
	}

	if len(output) > 0 {
		logs.Log.Debugf("[mutant-srdi] Output: %s", string(output))
	}

	// Read the generated shellcode
	shellcode, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read output file: %w", err)
	}

	if len(shellcode) == 0 {
		return nil, fmt.Errorf("generated shellcode is empty")
	}

	logs.Log.Infof("[mutant-srdi] Successfully generated %d bytes of shellcode", len(shellcode))
	return shellcode, nil
}
