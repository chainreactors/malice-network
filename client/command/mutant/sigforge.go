package mutant

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func SigforgeCmd(cmd *cobra.Command, con *core.Console) error {
	operation, _ := cmd.Flags().GetString("operation")
	source, _ := cmd.Flags().GetString("source")
	target, _ := cmd.Flags().GetString("target")
	signature, _ := cmd.Flags().GetString("signature")
	output, _ := cmd.Flags().GetString("output")

	// Validate operation
	validOps := map[string]bool{
		"extract": true,
		"copy":    true,
		"inject":  true,
		"remove":  true,
		"check":   true,
	}
	if !validOps[operation] {
		return fmt.Errorf("invalid operation: %s (must be extract, copy, inject, remove, or check)", operation)
	}

	// Validate source file
	if source == "" {
		return fmt.Errorf("source file is required")
	}

	// Read source file
	sourceBin, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	con.Log.Infof("Read %d bytes from source: %s\n", len(sourceBin), source)

	// Prepare request
	req := &clientpb.MutantSigforgeRequest{
		Operation: operation,
		SourceBin: sourceBin,
	}

	// Handle operation-specific inputs
	switch operation {
	case "copy":
		if target == "" {
			return fmt.Errorf("target file is required for copy operation")
		}
		targetBin, err := os.ReadFile(target)
		if err != nil {
			return fmt.Errorf("failed to read target file: %w", err)
		}
		con.Log.Infof("Read %d bytes from target: %s\n", len(targetBin), target)
		req.TargetBin = targetBin

	case "inject":
		if signature == "" {
			return fmt.Errorf("signature file is required for inject operation")
		}
		sigBin, err := os.ReadFile(signature)
		if err != nil {
			return fmt.Errorf("failed to read signature file: %w", err)
		}
		con.Log.Infof("Read %d bytes from signature: %s\n", len(sigBin), signature)
		req.Signature = sigBin
	}

	// Call RPC
	con.Log.Infof("Calling MutantSigforge RPC with operation: %s\n", operation)
	resp, err := con.Rpc.MutantSigforge(con.Context(), req)
	if err != nil {
		return fmt.Errorf("MutantSigforge RPC failed: %w", err)
	}

	// Handle response based on operation
	if operation == "check" {
		// Check operation returns text output
		con.Log.Console(string(resp.Bin))
		return nil
	}

	con.Log.Infof("Operation %s completed: %d bytes\n", operation, len(resp.Bin))

	// Determine output file
	if output == "" {
		switch operation {
		case "extract":
			output = filepath.Join(assets.GetTempDir(), "signature.bin")
		case "copy", "inject":
			output = filepath.Join(assets.GetTempDir(), filepath.Base(source)+".signed")
		case "remove":
			output = filepath.Join(assets.GetTempDir(), filepath.Base(source)+".unsigned")
		}
	}

	// Write output file
	err = os.WriteFile(output, resp.Bin, 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	con.Log.Infof("Saved result to %s\n", output)
	return nil
}
