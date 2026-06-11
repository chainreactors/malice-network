package build

import (
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/utils/pe"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/wabzsy/gonut"
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
		return nil, fmt.Errorf("objcopy failed to extract shellcode: %w", err)
	}

	// Read the extracted binary shellcode
	bin, err := os.ReadFile(dstPath)
	if err != nil || len(bin) == 0 {
		return nil, fmt.Errorf("cannot read objcopy generated shellcode file '%s': %w", dstPath, err)
	}
	return bin, nil
}

func MutantSrdi(path string) ([]byte, error) {
	absBuildOutputPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve absolute path for build output directory '%s': %w", path, err)
	}

	// Create temporary file with unique name for mutant output
	dstPath := absBuildOutputPath + ".temp_mutant_file"

	// Ensure cleanup of temporary file after processing
	defer func() {
		if err := os.Remove(dstPath); err != nil && !os.IsNotExist(err) {
			logs.Log.Warnf("Unable to cleanup temporary mutant file '%s' - manual cleanup may be required: %v", dstPath, err)
		}
	}()

	mutantExePath := filepath.Join(configs.BinPath, "malefic-mutant")
	mutantExeAbsPath, err := filepath.Abs(mutantExePath)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve absolute path for malefic-mutant.exe: %w", err)
	}

	mutantCommand := []string{
		mutantExeAbsPath,
		"tool", "srdi",
		"-i", absBuildOutputPath,
		"-o", dstPath,
	}
	logs.Log.Infof("Executing mutant command: %v\n", mutantCommand)

	cmd := exec.Command(mutantCommand[0], mutantCommand[1:]...)
	cmd.Dir = filepath.Dir(absBuildOutputPath)
	output, err := cmd.CombinedOutput()

	if len(output) > 0 {
		logs.Log.Debugf("Mutant command output: %s", string(output))
	}

	if err != nil {
		return nil, fmt.Errorf("malefic-mutant.exe failed to extract shellcode: %v", err)
	}

	bin, err := os.ReadFile(dstPath)
	if err != nil || len(bin) == 0 {
		return nil, fmt.Errorf("cannot read mutant generated shellcode file '%s': %w", dstPath, err)
	}
	return bin, nil
}

func SRDIArtifact(bin []byte, platform, arch string, RDIType string) ([]byte, error) {
	if RDIType == consts.RDIObjcopy {
		filename := filepath.Join(encoders.UUID())
		if err := os.WriteFile(filename, bin, 0644); err != nil {
			return nil, err
		}
		defer os.Remove(filename)
		return ObjcopyPulse(filename, platform, arch)
	} else if RDIType == consts.RDIDonut {
		switch pe.CheckPEType(bin) {
		case consts.DLLFile:
			return gonut.DonutShellcodeFromPE("bin"+consts.DLL, bin, arch, "", false, true)
		case consts.EXEFile:
			return gonut.DonutShellcodeFromPE("bin"+consts.PEFile, bin, arch, "", false, true)
		default:
			return nil, fmt.Errorf("unsupported file type")
		}
	} else {
		filename := filepath.Join(encoders.UUID())
		if err := os.WriteFile(filename, bin, 0644); err != nil {
			return nil, err
		}
		defer os.Remove(filename)
		return MutantSrdi(filename)
	}
}

func ConvertArtifact(artifact *clientpb.Artifact, format string, rdi string) (*clientpb.Artifact, error) {
	if format == "" || format == consts.FormatExecutable {
		return artifact, nil
	}
	// Artifact already built as shellcode (e.g. pulse --shellcode), skip SRDI conversion
	if artifact.Format == consts.ShellcodeFile {
		convert, err := output.Convert(artifact.Bin, format)
		if err != nil {
			return nil, err
		}
		artifact.Bin = convert.Data
		artifact.Format = format
		return artifact, nil
	}
	if artifact.Platform != consts.Windows {
		convert, err := output.Convert(artifact.Bin, format)
		if err != nil {
			return nil, err
		}
		artifact.Bin = convert.Data
		artifact.Format = format
		return artifact, nil
	}
	if rdi == "" {
		if artifact.Type == consts.CommandBuildPulse {
			rdi = consts.RDIObjcopy
		} else {
			rdi = consts.DefaultRDI
		}
	}
	shellcode, err := SRDIArtifact(artifact.Bin, artifact.Platform, artifact.Arch, rdi)
	if err != nil {
		return nil, fmt.Errorf("failed to convert: %s", err)
	}

	convert, err := output.Convert(shellcode, format)
	if err != nil {
		return nil, err
	}

	artifact.Bin = convert.Data
	artifact.Format = format
	return artifact, nil
}
