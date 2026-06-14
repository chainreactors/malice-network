package mutant

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/configs"
)

const defaultToolTimeout = 5 * time.Minute
const maxToolTimeout = time.Hour

type ToolFile struct {
	Name string
	Bin  []byte
}

type ToolRequest struct {
	Args           []string
	Inputs         []ToolFile
	Outputs        []string
	TimeoutSeconds uint32
}

type ToolResponse struct {
	Stdout []byte
	Files  []ToolFile
}

func safeRelPath(name string) (string, error) {
	name = filepath.Clean(filepath.FromSlash(strings.TrimSpace(name)))
	if name == "." || name == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	if filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe relative path: %s", name)
	}
	return name, nil
}

func Tool(req *ToolRequest) (*ToolResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if len(req.Args) == 0 {
		return nil, fmt.Errorf("tool args are required")
	}

	workDir, err := os.MkdirTemp(configs.TempPath, "mutant-tool-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create mutant tool temp dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	for _, input := range req.Inputs {
		rel, err := safeRelPath(input.Name)
		if err != nil {
			return nil, fmt.Errorf("invalid input path %q: %w", input.Name, err)
		}
		dst := filepath.Join(workDir, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
			return nil, fmt.Errorf("failed to create input parent dir: %w", err)
		}
		if err := os.WriteFile(dst, input.Bin, 0600); err != nil {
			return nil, fmt.Errorf("failed to write input %q: %w", input.Name, err)
		}
	}

	timeout := defaultToolTimeout
	if req.TimeoutSeconds > 0 {
		timeout = time.Duration(req.TimeoutSeconds) * time.Second
		if timeout > maxToolTimeout {
			timeout = maxToolTimeout
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	binaryPath := mutantPath()
	if err := CheckBinaryExecutable(binaryPath); err != nil {
		return nil, err
	}

	args := append([]string{"tool"}, req.Args...)
	logs.Log.Infof("[mutant-tool] Executing: %s %v", binaryPath, args)
	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Dir = workDir
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("malefic-mutant tool timed out after %s, output: %s", timeout, string(output))
	}
	if err != nil {
		logs.Log.Errorf("[mutant-tool] Command failed: %s", string(output))
		return nil, fmt.Errorf("malefic-mutant tool failed: %w, output: %s", err, string(output))
	}
	if len(output) > 0 {
		logs.Log.Debugf("[mutant-tool] Output: %s", string(output))
	}

	resp := &ToolResponse{Stdout: output}
	for _, outputName := range req.Outputs {
		rel, err := safeRelPath(outputName)
		if err != nil {
			return nil, fmt.Errorf("invalid output path %q: %w", outputName, err)
		}
		fullPath := filepath.Join(workDir, rel)
		info, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to stat output %q: %w", outputName, err)
		}
		if info.IsDir() {
			err = filepath.WalkDir(fullPath, func(path string, d os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if d.IsDir() {
					return nil
				}
				bin, readErr := os.ReadFile(path)
				if readErr != nil {
					return readErr
				}
				name, relErr := filepath.Rel(workDir, path)
				if relErr != nil {
					return relErr
				}
				resp.Files = append(resp.Files, ToolFile{Name: filepath.ToSlash(name), Bin: bin})
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("failed to collect output dir %q: %w", outputName, err)
			}
			continue
		}

		bin, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read output %q: %w", outputName, err)
		}
		resp.Files = append(resp.Files, ToolFile{Name: filepath.ToSlash(rel), Bin: bin})
	}
	return resp, nil
}
