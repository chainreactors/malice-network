package mutant

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

type toolInput struct {
	Remote string
	Local  string
}

type toolOutput struct {
	Remote string
	Local  string
}

func defaultOutputPath(input, suffix string) string {
	return filepath.Join(assets.GetTempDir(), filepath.Base(input)+suffix)
}

func remoteFileName(name, fallback string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "." || base == string(filepath.Separator) || base == "" {
		return fallback
	}
	return base
}

func appendFlag(args []string, name, value string) []string {
	if strings.TrimSpace(value) == "" {
		return args
	}
	return append(args, name, value)
}

func appendBoolFlag(args []string, name string, enabled bool) []string {
	if enabled {
		return append(args, name)
	}
	return args
}

func callMutantTool(con *core.Console, args []string, inputs []toolInput, outputs []toolOutput, timeout uint32) (*clientpb.MutantToolResponse, error) {
	req := &clientpb.MutantToolRequest{
		Args:           args,
		Outputs:        make([]string, 0, len(outputs)),
		TimeoutSeconds: timeout,
	}

	for _, input := range inputs {
		bin, err := os.ReadFile(input.Local)
		if err != nil {
			return nil, fmt.Errorf("failed to read input %s: %w", input.Local, err)
		}
		req.Inputs = append(req.Inputs, &clientpb.MutantToolFile{
			Name: input.Remote,
			Bin:  bin,
		})
	}

	for _, output := range outputs {
		req.Outputs = append(req.Outputs, output.Remote)
	}

	con.Log.Infof("Calling MutantTool RPC: tool %s\n", strings.Join(args, " "))
	resp, err := con.Rpc.MutantTool(con.Context(), req)
	if err != nil {
		return nil, fmt.Errorf("MutantTool RPC failed: %w", err)
	}
	return resp, nil
}

func writeToolOutputs(resp *clientpb.MutantToolResponse, outputs []toolOutput, required ...string) error {
	outputMap := make(map[string]string, len(outputs))
	outputDirs := make(map[string]string, len(outputs))
	for _, output := range outputs {
		remote := filepath.ToSlash(output.Remote)
		outputMap[remote] = output.Local
		if strings.HasSuffix(remote, "/") || filepath.Ext(remote) == "" {
			outputDirs[strings.TrimSuffix(remote, "/")+"/"] = output.Local
		}
	}
	requiredSet := make(map[string]bool, len(required))
	for _, name := range required {
		requiredSet[filepath.ToSlash(name)] = true
	}

	seen := make(map[string]bool)
	for _, file := range resp.GetFiles() {
		if file == nil {
			continue
		}
		name := filepath.ToSlash(file.GetName())
		local, ok := outputMap[name]
		if !ok {
			for remoteDir, localDir := range outputDirs {
				if strings.HasPrefix(name, remoteDir) {
					local = filepath.Join(localDir, strings.TrimPrefix(name, remoteDir))
					ok = true
					break
				}
			}
		}
		if !ok {
			local = filepath.Join(assets.GetTempDir(), filepath.Base(name))
		}
		if err := os.MkdirAll(filepath.Dir(local), 0700); err != nil {
			return fmt.Errorf("failed to create output directory for %s: %w", local, err)
		}
		if err := os.WriteFile(local, file.GetBin(), 0644); err != nil {
			return fmt.Errorf("failed to write output %s: %w", local, err)
		}
		seen[name] = true
	}

	for name := range requiredSet {
		if !seen[name] {
			return fmt.Errorf("expected mutant output %s was not produced", name)
		}
	}
	return nil
}

func printToolStdout(con *core.Console, resp *clientpb.MutantToolResponse) {
	if len(resp.GetStdout()) > 0 {
		con.Log.Console(string(resp.GetStdout()))
	}
}

func getTimeout(cmd *cobra.Command) uint32 {
	timeout, _ := cmd.Flags().GetUint32("timeout")
	return timeout
}

func bindToolTimeout(cmd *cobra.Command) {
	cmd.Flags().Uint32("timeout", 300, "mutant tool timeout in seconds")
}
