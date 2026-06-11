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

func StripCmd(cmd *cobra.Command, con *core.Console) error {
	input, _ := cmd.Flags().GetString("input")
	output, _ := cmd.Flags().GetString("output")
	customPaths, _ := cmd.Flags().GetString("custom-paths")

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

	// Parse custom paths
	var customPathsList []string
	if customPaths != "" {
		customPathsList = strings.Split(customPaths, ",")
		con.Log.Infof("Custom paths: %v\n", customPathsList)
	}

	// Call RPC
	con.Log.Info("Calling MutantStrip RPC...\n")
	resp, err := con.Rpc.MutantStrip(con.Context(), &clientpb.MutantStripRequest{
		Bin:         inputBin,
		CustomPaths: customPathsList,
	})
	if err != nil {
		return fmt.Errorf("MutantStrip RPC failed: %w", err)
	}

	con.Log.Infof("Stripped binary: %d bytes\n", len(resp.Bin))

	// Determine output file
	if output == "" {
		output = filepath.Join(assets.GetTempDir(), filepath.Base(input)+".stripped")
	}

	// Write output file
	err = os.WriteFile(output, resp.Bin, 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	con.Log.Infof("Saved stripped binary to %s\n", output)
	return nil
}
