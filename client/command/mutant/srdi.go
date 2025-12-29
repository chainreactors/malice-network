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

func SrdiCmd(cmd *cobra.Command, con *core.Console) error {
	input, _ := cmd.Flags().GetString("input")
	output, _ := cmd.Flags().GetString("output")
	arch, _ := cmd.Flags().GetString("arch")
	functionName, _ := cmd.Flags().GetString("function-name")
	platform, _ := cmd.Flags().GetString("platform")
	srdiType, _ := cmd.Flags().GetString("type")
	userdataPath, _ := cmd.Flags().GetString("userdata-path")

	// Validate input file
	if input == "" {
		return fmt.Errorf("input file is required")
	}

	// Read input file
	inputBin, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	con.Log.Infof("Read %d bytes from %s\n", len(inputBin), input)

	// Read userdata if provided
	var userdata []byte
	if userdataPath != "" {
		userdata, err = os.ReadFile(userdataPath)
		if err != nil {
			return fmt.Errorf("failed to read userdata file: %w", err)
		}
		con.Log.Infof("Read %d bytes of userdata from %s\n", len(userdata), userdataPath)
	}

	// Call RPC
	con.Log.Info("Calling MutantSrdi RPC...\n")
	resp, err := con.Rpc.MutantSrdi(con.Context(), &clientpb.MutantSrdiRequest{
		Bin:          inputBin,
		Arch:         arch,
		FunctionName: functionName,
		Platform:     platform,
		Type:         srdiType,
		Userdata:     userdata,
	})
	if err != nil {
		return fmt.Errorf("MutantSrdi RPC failed: %w", err)
	}

	con.Log.Infof("Generated %d bytes of shellcode\n", len(resp.Bin))

	// Determine output file
	if output == "" {
		output = filepath.Join(assets.GetTempDir(), filepath.Base(input)+".bin")
	}

	// Write output file
	err = os.WriteFile(output, resp.Bin, 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	con.Log.Infof("Saved shellcode to %s\n", output)
	return nil
}
