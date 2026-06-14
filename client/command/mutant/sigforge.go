package mutant

import (
	"fmt"
	"path/filepath"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func SigforgeCmd(cmd *cobra.Command, con *core.Console) error {
	operation, _ := cmd.Flags().GetString("operation")
	source, _ := cmd.Flags().GetString("source")
	target, _ := cmd.Flags().GetString("target")
	signature, _ := cmd.Flags().GetString("signature")
	host, _ := cmd.Flags().GetString("host")
	certFile, _ := cmd.Flags().GetString("cert-file")
	port, _ := cmd.Flags().GetUint16("port")
	output, _ := cmd.Flags().GetString("output")

	// Validate operation
	validOps := map[string]bool{
		"extract":     true,
		"copy":        true,
		"inject":      true,
		"remove":      true,
		"check":       true,
		"carbon-copy": true,
	}
	if !validOps[operation] {
		return fmt.Errorf("invalid operation: %s (must be extract, copy, inject, remove, check, or carbon-copy)", operation)
	}

	var args []string
	var inputs []toolInput
	var outputs []toolOutput
	var required []string

	// Handle operation-specific inputs
	switch operation {
	case "extract":
		if source == "" {
			return fmt.Errorf("source file is required")
		}
		if output == "" {
			output = filepath.Join(assets.GetTempDir(), "signature.bin")
		}
		args = []string{"sigforge", "extract", "-i", "source.exe", "-o", "signature.bin"}
		inputs = append(inputs, toolInput{Remote: "source.exe", Local: source})
		outputs = append(outputs, toolOutput{Remote: "signature.bin", Local: output})
		required = append(required, "signature.bin")

	case "copy":
		if source == "" {
			return fmt.Errorf("source file is required")
		}
		if target == "" {
			return fmt.Errorf("target file is required for copy operation")
		}
		if output == "" {
			output = filepath.Join(assets.GetTempDir(), filepath.Base(target)+".signed")
		}
		remoteOutput := remoteFileName(output, "signed-output.exe")
		args = []string{"sigforge", "copy", "-s", "source.exe", "-t", "target.exe", "-o", remoteOutput}
		inputs = append(inputs, toolInput{Remote: "source.exe", Local: source}, toolInput{Remote: "target.exe", Local: target})
		outputs = append(outputs, toolOutput{Remote: remoteOutput, Local: output})
		required = append(required, remoteOutput)

	case "inject":
		if signature == "" {
			return fmt.Errorf("signature file is required for inject operation")
		}
		if target == "" {
			return fmt.Errorf("target file is required for inject operation")
		}
		if output == "" {
			output = filepath.Join(assets.GetTempDir(), filepath.Base(target)+".signed")
		}
		remoteOutput := remoteFileName(output, "signed-output.exe")
		args = []string{"sigforge", "inject", "-s", "signature.bin", "-t", "target.exe", "-o", remoteOutput}
		inputs = append(inputs, toolInput{Remote: "signature.bin", Local: signature}, toolInput{Remote: "target.exe", Local: target})
		outputs = append(outputs, toolOutput{Remote: remoteOutput, Local: output})
		required = append(required, remoteOutput)

	case "remove":
		if source == "" {
			return fmt.Errorf("source file is required")
		}
		if output == "" {
			output = filepath.Join(assets.GetTempDir(), filepath.Base(source)+".unsigned")
		}
		remoteOutput := remoteFileName(output, "unsigned-output.exe")
		args = []string{"sigforge", "remove", "-i", "source.exe", "-o", remoteOutput}
		inputs = append(inputs, toolInput{Remote: "source.exe", Local: source})
		outputs = append(outputs, toolOutput{Remote: remoteOutput, Local: output})
		required = append(required, remoteOutput)

	case "check":
		if source == "" {
			return fmt.Errorf("source file is required")
		}
		args = []string{"sigforge", "check", "-i", "source.exe"}
		inputs = append(inputs, toolInput{Remote: "source.exe", Local: source})

	case "carbon-copy":
		if host == "" && certFile == "" {
			return fmt.Errorf("host or cert-file is required for carbon-copy operation")
		}
		if target == "" {
			return fmt.Errorf("target file is required for carbon-copy operation")
		}
		if output == "" {
			output = filepath.Join(assets.GetTempDir(), filepath.Base(target)+".carbon")
		}
		remoteOutput := remoteFileName(output, "carbon-output.exe")
		args = []string{"sigforge", "carbon-copy", "-t", "target.exe", "-o", remoteOutput, "--port", fmt.Sprintf("%d", port)}
		if host != "" {
			args = append(args, "--host", host)
		}
		if certFile != "" {
			args = append(args, "--cert-file", "cert-file")
			inputs = append(inputs, toolInput{Remote: "cert-file", Local: certFile})
		}
		inputs = append(inputs, toolInput{Remote: "target.exe", Local: target})
		outputs = append(outputs, toolOutput{Remote: remoteOutput, Local: output})
		required = append(required, remoteOutput)
	}

	resp, err := callMutantTool(con, args, inputs, outputs, getTimeout(cmd))
	if err != nil {
		return err
	}
	printToolStdout(con, resp)
	return writeToolOutputs(resp, outputs, required...)
}
