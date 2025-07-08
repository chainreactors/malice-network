package formatutils

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/wabzsy/gonut"
	"os"
	"os/exec"
	"path/filepath"
)

// ObjcopyPulse extracts shellcode from compiled artifact using objcopy
func ObjcopyPulse(path, platform, arch string) ([]byte, error) {
	absBuildOutputPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve absolute path for build output directory '%s': %w", path, err)
	}

	// Create temporary file with unique name for objcopy output
	dstPath := absBuildOutputPath + ".temp_objcopy_file"

	// Ensure cleanup of temporary file after processing
	defer func() {
		if err := os.Remove(dstPath); err != nil && !os.IsNotExist(err) {
			logs.Log.Warnf("Unable to cleanup temporary objcopy file '%s' - manual cleanup may be required: %v", dstPath, err)
		}
	}()

	// Prepare objcopy command to extract .text section as binary
	//objcopyCommand := []string{"objcopy", "--only-section=.text", "-O", "binary", builder.Path, dstPath}
	objcopyCommand := []string{"objcopy", "-O", "binary", absBuildOutputPath, dstPath}
	logs.Log.Infof("Executing objcopy command: %v\n", objcopyCommand)

	// Execute objcopy command with proper working directory
	cmd := exec.Command(objcopyCommand[0], objcopyCommand[1:]...)
	cmd.Dir = filepath.Dir(absBuildOutputPath)
	output, err := cmd.CombinedOutput()

	if len(output) > 0 {
		logs.Log.Debugf("Objcopy command output: %s", string(output))
	}

	if err != nil {
		return nil, fmt.Errorf("objcopy failed to extract shellcode %s")
	}

	// Read the extracted binary shellcode
	bin, err := os.ReadFile(dstPath)
	if err != nil || len(bin) == 0 {
		return nil, fmt.Errorf("cannot read objcopy generated shellcode file '%s': %w", dstPath, err)
	}
	return bin, nil
}

func SRDIArtifact(path, platform, arch string, useobjcopy bool) ([]byte, error) {
	if useobjcopy {
		return ObjcopyPulse(path, platform, arch)
	} else {
		return gonut.DonutShellcodeFromFile(path, arch, "")
	}
}
