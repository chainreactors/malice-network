package generate

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	ErrFailedToEncode = errors.New("failed to encode shellcode")
)

const (
	defaultSGNIterations     = 1
	defaultSGNMaxObfuscation = 20
)

// SGNConfig - Configuration for sgn
type SGNConfig struct {
	AppDir string

	Architecture   string // Binary architecture (32/64) (default 32)
	Asci           bool   // Generates a full ASCII printable payload (takes very long time to bruteforce)
	BadChars       []byte // Don't use specified bad characters given in hex format (\x00\x01\x02...)
	Iterations     int    // Number of times to encode the binary (increases overall size) (default 1)
	MaxObfuscation int    // Maximum number of bytes for obfuscation (default 20)
	PlainDecoder   bool   // Do not encode the decoder stub
	Safe           bool   // Do not modify any register values

	Verbose bool

	Output string
	Input  string
}

// sgnCmd - Execute a sgn command
func sgnCmd(appDir string, cwd string, command []string) ([]byte, error) {
	sgnName := "sgn"
	if runtime.GOOS == "windows" {
		sgnName = "sgn.exe"
	}
	sgnBinPath := filepath.Join(appDir, "go", "bin", sgnName)

	cmd := exec.Command(sgnBinPath, command...)
	cmd.Dir = cwd
	cmd.Env = []string{
		fmt.Sprintf("PATH=%s:%s", filepath.Join(appDir, "go", "bin"), os.Getenv("PATH")),
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	logs.Log.Infof("sgn cmd: '%v'", cmd)
	err := cmd.Run()
	if err != nil {
		logs.Log.Infof("--- env ---\n")
		for _, envVar := range cmd.Env {
			logs.Log.Infof("%s\n", envVar)
		}
		logs.Log.Infof("--- stdout ---\n%s\n", stdout.String())
		logs.Log.Infof("--- stderr ---\n%s\n", stderr.String())
		logs.Log.Info(err.Error())
	}
	return stdout.Bytes(), err
}

// // EncodeShellcode - Encode a shellcode
func EncodeShellcode(shellcode []byte, arch string, iterations int, badChars []byte) ([]byte, error) {
	logs.Log.Infof("[sgn] EncodeShellcode: %d bytes", len(shellcode))
	inputFile, err := os.CreateTemp("", "sgn")
	if err != nil {
		logs.Log.Error(err.Error())
		return nil, ErrFailedToEncode
	}
	if _, err = inputFile.Write(shellcode); err != nil {
		inputFile.Close()
		logs.Log.Error(err.Error())
		return nil, ErrFailedToEncode
	}
	if err = inputFile.Close(); err != nil {
		logs.Log.Error(err.Error())
		return nil, ErrFailedToEncode
	}
	defer os.Remove(inputFile.Name())
	outputFile, err := os.CreateTemp("", "sgn")
	if err != nil {
		logs.Log.Error(err.Error())
		return nil, ErrFailedToEncode
	}
	outputFile.Close()
	defer os.Remove(outputFile.Name())

	config := SGNConfig{
		AppDir: assets.GetRootAppDir(),

		Architecture:   strings.ToLower(arch),
		Iterations:     iterations,
		MaxObfuscation: 20,
		Safe:           false,
		PlainDecoder:   false,
		Asci:           false,
		BadChars:       badChars,
		Verbose:        false,

		Input:  inputFile.Name(),
		Output: outputFile.Name(),
	}
	_, err = sgnCmd(config.AppDir, ".", configToArgs(config))
	if err != nil {
		logs.Log.Error(err.Error())
		return nil, ErrFailedToEncode
	}
	data, err := os.ReadFile(outputFile.Name())
	if err != nil {
		logs.Log.Error(err.Error())
		return nil, ErrFailedToEncode
	}
	logs.Log.Infof("[sgn] successfully encoded to %d bytes", len(data))
	return data, nil
}

func configToArgs(config SGNConfig) []string {
	args := []string{}

	args = append(args, "--input", config.Input)
	args = append(args, "--out", config.Output)
	args = append(args, "--arch", normalizeSGNArchitecture(config.Architecture))
	args = append(args, "--enc", fmt.Sprintf("%d", normalizePositiveInt(config.Iterations, defaultSGNIterations)))
	args = append(args, "--max", fmt.Sprintf("%d", normalizePositiveInt(config.MaxObfuscation, defaultSGNMaxObfuscation)))

	if config.Safe {
		args = append(args, "--safe")
	}

	if config.PlainDecoder {
		args = append(args, "--plain")
	}

	if config.Asci {
		args = append(args, "--ascii")
	}

	if 0 < len(config.BadChars) {
		badChars := []string{}
		for _, b := range config.BadChars {
			badChars = append(badChars, fmt.Sprintf("\\x%02x", b))
		}
		args = append(args, "--badchars", strings.Join(badChars, ""))
	}

	if config.Verbose {
		args = append(args, "--verbose")
	}

	logs.Log.Infof("[sgn] input file: %s", config.Input)
	return args
}

func normalizeSGNArchitecture(arch string) string {
	switch strings.ToLower(strings.TrimSpace(arch)) {
	case "386", "32", "x86", "i386":
		return "32"
	default:
		return "64"
	}
}

func normalizePositiveInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
